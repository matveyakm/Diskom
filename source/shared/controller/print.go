package restController

// Печать документов: JSON-шаблон → PDF (folio) и DOCX (stdlib).
//
// Один шаблон описывает оба формата (поведение — в данных): новый
// документ/отчёт = новый файл templates/print.json в папке сервиса,
// кода трогать не надо. Шаблон печётся в образ (WORKDIR сервиса),
// путь — env PRINT_TEMPLATE (дефолт "templates/print.json").
// Нет файла — честный 404, а не падение.
//
// Формат шаблона v2:
//
//	{
//	  "title": "Приказ {{dnumber}}",
//	  "header": "...", "footer": "Стр. ...",
//	  "blocks": [
//	    {"type": "heading", "text": "...", "align": "center",
//	     "bold": true, "size": 13, "color": "000000",
//	     "indent": 0, "firstLine": 0},
//	    {"type": "field", "label": "Номер", "field": "dnumber"},
//	    {"type": "table", "title": "...",
//	     "columns": [{"field": "last_name", "label": "Фамилия",
//	                  "width": 2, "align": "left", "ref": "employee_directory"}]},
//	    {"type": "table", "title": "Реквизиты",
//	     "rows": [{"label": "Отдел", "field": "id_department",
//	               "ref": "department_directory"}]},
//	    {"type": "list", "ordered": false, "items": ["п.1 {{note}}"]},
//	    {"type": "image", "src": "logo.png", "width": 40, "align": "center"},
//	    {"type": "text", "text": "Основание: {{note}}"}
//	  ]
//	}
//
// {{field}} резолвится из строки документа, даты — в 02.01.2006,
// отсутствующее поле — пустая строка, пустой field-блок и пустой пункт
// списка не печатаются. "ref" подменяет id именем из справочника
// (fail-open: нет связи/упал справочник — сырой id).
// Структура валидируется fail-closed (битый шаблон — 500, это ошибка
// оператора, запечённая в образ).

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// defaultPrintTemplate ищется относительно WORKDIR сервиса
	// (/app/document|directory|.../ms_* — см. Dockerfile.work).
	defaultPrintTemplate = "templates/print.json"
	// printTokenTTL — жизнь готового файла между шагами "Распечатать"
	// и скачиванием. Инстанс один, рестарт между шагами убьёт файл.
	printFileTTL = 5 * time.Minute
	// maxPrintTemplateBytes — кап шаблона (свой синтаксис, свои капы).
	maxPrintTemplateBytes = 1 << 20 // 1 MiB
	// maxPrintImageBytes — кап картинки (свои капы, чужой файл).
	maxPrintImageBytes = 2 << 20 // 2 MiB
)

var (
	rePrintPlaceholder = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*(!?)\s*\}\}`)
	rePrintColor       = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)
	printTokenStore    = newPrintStore()
)

// printTextStyle — общие оформительские опции текстовых блоков.
// font: arial (дефолт) или times (Tinos, метрически Times New Roman).
// align: left|center|right|justify. size: пункты (0 = дефолт типа).
// indent: левый отступ мм; firstLine: абзацный отступ мм (точный в обоих
// форматах: DOCX — w:firstLine, PDF — folio SetFirstLineIndent).
// rightIndent: правый отступ мм (точный в обоих форматах).
// spaceBefore/spaceAfter: мм сверху/снизу (дефолт after — 2.1 мм, как раньше);
// line: межстрочный интервал 1|1.5|2 (точный в обоих форматах).
type printTextStyle struct {
	Font        string  `json:"font,omitempty"`
	Align       string  `json:"align,omitempty"`
	Bold        bool    `json:"bold,omitempty"`
	Italic      bool    `json:"italic,omitempty"`
	Underline   bool    `json:"underline,omitempty"`
	Color       string  `json:"color,omitempty"`
	Size        float64 `json:"size,omitempty"`
	Indent      float64 `json:"indent,omitempty"`
	FirstLine   float64 `json:"firstLine,omitempty"`
	RightIndent float64 `json:"rightIndent,omitempty"`
	SpaceBefore float64 `json:"spaceBefore,omitempty"`
	SpaceAfter  float64 `json:"spaceAfter,omitempty"`
	Line        float64 `json:"line,omitempty"`
}

// printFontFamily резолвит шрифт блока: свой → дефолт шаблона → arial.
func printFontFamily(tpl *printTemplate, s printTextStyle) string {
	if s.Font != "" {
		return s.Font
	}
	if tpl.Font != "" {
		return tpl.Font
	}
	return "arial"
}

// printPageNum — где нумеровать страницы (дефолт footer, как раньше).
func (tpl *printTemplate) printPageNum() string {
	if tpl.PageNum == "" {
		return "footer"
	}
	return tpl.PageNum
}

// printColumnValue резолвит ячейку: литерал text (с подстановками)
// либо значение поля строки. Пустое — пропускается вызывающим.
func printColumnValue(c printBlockColumn, row map[string]any) string {
	if c.Text != "" {
		return renderPrintText(c.Text, row)
	}
	return stringifyPrintValue(row[c.Field])
}

// printBlock — один презентационный блок шаблона.
// Table-режим выбирается полями Columns (горизонтальная) или Rows
// (вертикальная); Header/Borders — указатели, чтобы отличать "не задано"
// (дефолт true) от явного false.
type printBlock struct {
	printTextStyle
	Type    string             `json:"type"`
	Text    string             `json:"text,omitempty"`
	Label   string             `json:"label,omitempty"`
	Field   string             `json:"field,omitempty"`
	Ref     string             `json:"ref,omitempty"`
	Title   string             `json:"title,omitempty"`
	Columns []printBlockColumn `json:"columns,omitempty"`
	Rows    []printBlockRow    `json:"rows,omitempty"`
	Items   []string           `json:"items,omitempty"`
	Ordered bool               `json:"ordered,omitempty"`
	Src     string             `json:"src,omitempty"`
	Width   float64            `json:"width,omitempty"`
	// Float: картинка поверх текста: в DOCX — wp:anchor, в PDF — folio
	// Div.AddOverlay (настоящий оверлей, не поток). x — мм от левого края
	// текстовой колонки, y — мм ниже верха абзаца-якоря (позиция блока
	// image в потоке; так пишет сам Word). Без float — в потоке текста
	// (wp:inline / обычный элемент).
	Float   bool    `json:"float,omitempty"`
	X       float64 `json:"x,omitempty"`
	Y       float64 `json:"y,omitempty"`
	Header  *bool   `json:"header,omitempty"`
	Borders *bool   `json:"borders,omitempty"`
}

// printBlockShow решает tri-state флаг (nil = дефолт true).
func printBlockShow(v *bool) bool {
	return v == nil || *v
}

// imageKey — ключ картинки в map байтов: src для файлов шаблона,
// "field:<имя>" для картинок из строки (id файла из таблицы files).
func (b printBlock) imageKey() string {
	if b.Src != "" {
		return b.Src
	}
	return "field:" + b.Field
}

type printBlockColumn struct {
	Field     string  `json:"field,omitempty"`
	Text      string  `json:"text,omitempty"`
	Label     string  `json:"label"`
	Ref       string  `json:"ref,omitempty"`
	Width     float64 `json:"width,omitempty"`
	Align     string  `json:"align,omitempty"`
	Bold      bool    `json:"bold,omitempty"`
	Italic    bool    `json:"italic,omitempty"`
	Underline bool    `json:"underline,omitempty"`
	Color     string  `json:"color,omitempty"`
	Size      float64 `json:"size,omitempty"`
	Font      string  `json:"font,omitempty"`
}

// printBlockRow — строка вертикальной таблицы (label | value).
type printBlockRow struct {
	Label string `json:"label"`
	Field string `json:"field"`
	Ref   string `json:"ref,omitempty"`
}

// printMargins — поля страницы мм (0 = дефолт движка: PDF 10/10/10/20,
// DOCX 1440 twips со всех сторон). Пример: {"left": 30, "top": 17.5}.
type printMargins struct {
	Left   float64 `json:"left,omitempty"`
	Top    float64 `json:"top,omitempty"`
	Right  float64 `json:"right,omitempty"`
	Bottom float64 `json:"bottom,omitempty"`
}

// printTemplate — распарсенный и проверенный шаблон. Title необязателен:
// шапка документа (организация) может идти обычными блоками до заголовка,
// тогда title пропускается и в PDF, и в DOCX.
// pageNum: header|footer|none (дефолт footer) — где нумеровать страницы.
type printTemplate struct {
	Title   string        `json:"title"`
	Font    string        `json:"font,omitempty"`
	Header  string        `json:"header,omitempty"`
	Footer  string        `json:"footer,omitempty"`
	PageNum string        `json:"pageNum,omitempty"`
	Margins *printMargins `json:"margins,omitempty"`
	Blocks  []printBlock  `json:"blocks"`
}

// validatePrintStyle проверяет оформительские опции (общие для блоков).
func validatePrintStyle(where string, s printTextStyle, family string) error {
	if s.Font != "" && s.Font != "arial" && s.Font != "times" {
		return fmt.Errorf("%s: invalid font %q (want arial|times)", where, s.Font)
	}
	if s.Italic && family != "times" {
		return fmt.Errorf("%s: italic needs times (arial has no italic face embedded)", where)
	}
	switch s.Align {
	case "", "left", "center", "right", "justify":
	default:
		return fmt.Errorf("%s: invalid align %q (want left|center|right|justify)", where, s.Align)
	}
	if s.Color != "" && !rePrintColor.MatchString(s.Color) {
		return fmt.Errorf("%s: invalid color %q (want RRGGBB hex)", where, s.Color)
	}
	if s.Size != 0 && (s.Size < 6 || s.Size > 72) {
		return fmt.Errorf("%s: invalid size %v (want 6..72 pt)", where, s.Size)
	}
	if s.Indent < 0 || s.Indent > 100 || s.FirstLine < 0 || s.FirstLine > 100 {
		return fmt.Errorf("%s: invalid indent/firstLine (want 0..100 mm)", where)
	}
	if s.RightIndent < 0 || s.RightIndent > 100 {
		return fmt.Errorf("%s: invalid rightIndent (want 0..100 mm)", where)
	}
	if s.SpaceBefore < 0 || s.SpaceBefore > 50 || s.SpaceAfter < 0 || s.SpaceAfter > 50 {
		return fmt.Errorf("%s: invalid spaceBefore/spaceAfter (want 0..50 mm)", where)
	}
	if s.Line != 0 && s.Line != 1 && s.Line != 1.5 && s.Line != 2 {
		return fmt.Errorf("%s: invalid line %v (want 1|1.5|2)", where, s.Line)
	}
	return nil
}

func validatePrintRef(where, ref string) error {
	if ref == "" {
		return nil
	}
	if !reExpandEndpoint.MatchString(ref) {
		return fmt.Errorf("%s: invalid ref %q", where, ref)
	}
	return nil
}

// parsePrintTemplate валидирует структуру (fail-closed: ошибка — 500,
// шаблон запечён оператором в образ, это не ввод пользователя).
func parsePrintTemplate(raw []byte) (*printTemplate, error) {
	if len(raw) > maxPrintTemplateBytes {
		return nil, fmt.Errorf("print template too large (max %d bytes)", maxPrintTemplateBytes)
	}
	var tpl printTemplate
	if err := json.Unmarshal(raw, &tpl); err != nil {
		return nil, fmt.Errorf("invalid print template JSON: %w", err)
	}
	// Title необязателен (шапка организации может идти блоками до заголовка).
	if len(tpl.Blocks) == 0 || len(tpl.Blocks) > 50 {
		return nil, fmt.Errorf("print template: blocks 1..50 required, got %d", len(tpl.Blocks))
	}
	if tpl.Font != "" && tpl.Font != "arial" && tpl.Font != "times" {
		return nil, fmt.Errorf("print template: invalid font %q (want arial|times)", tpl.Font)
	}
	switch tpl.PageNum {
	case "", "header", "footer", "none":
	default:
		return nil, fmt.Errorf("print template: invalid pageNum %q (want header|footer|none)", tpl.PageNum)
	}
	if tpl.Margins != nil {
		for _, m := range []float64{tpl.Margins.Left, tpl.Margins.Top, tpl.Margins.Right, tpl.Margins.Bottom} {
			if m < 0 || m > 50 {
				return nil, fmt.Errorf("print template: invalid margins (want 0..50 mm)")
			}
		}
	}
	for i, b := range tpl.Blocks {
		where := fmt.Sprintf("print template: block %d", i+1)
		family := b.Font
		if family == "" {
			family = tpl.Font
		}
		if family == "" {
			family = "arial"
		}
		if err := validatePrintStyle(where, b.printTextStyle, family); err != nil {
			return nil, err
		}
		if err := validatePrintRef(where, b.Ref); err != nil {
			return nil, err
		}
		switch b.Type {
		case "heading", "text":
			if strings.TrimSpace(b.Text) == "" {
				return nil, fmt.Errorf("%s: %s needs non-empty text", where, b.Type)
			}
		case "field":
			if strings.TrimSpace(b.Label) == "" {
				return nil, fmt.Errorf("%s: field needs label", where)
			}
			if !reFilterColumn.MatchString(b.Field) {
				return nil, fmt.Errorf("%s: invalid field %q", where, b.Field)
			}
		case "table":
			hasColumns := len(b.Columns) > 0
			hasRows := len(b.Rows) > 0
			if hasColumns == hasRows {
				return nil, fmt.Errorf("%s: table needs either columns or rows (not both/neither)", where)
			}
			if hasColumns {
				if len(b.Columns) > 12 {
					return nil, fmt.Errorf("%s: table needs 1..12 columns", where)
				}
				for _, c := range b.Columns {
					// Label нужен только при видимой шапке таблицы.
					if printBlockShow(b.Header) && strings.TrimSpace(c.Label) == "" {
						return nil, fmt.Errorf("%s: table column needs label", where)
					}
					if c.Text == "" && !reFilterColumn.MatchString(c.Field) {
						return nil, fmt.Errorf("%s: invalid column field %q", where, c.Field)
					}
					if c.Text != "" && c.Field != "" {
						return nil, fmt.Errorf("%s: column takes either text or field, not both", where)
					}
					if c.Text != "" && c.Ref != "" {
						return nil, fmt.Errorf("%s: column takes either text or ref, not both", where)
					}
					if err := validatePrintRef(where, c.Ref); err != nil {
						return nil, err
					}
					if c.Width != 0 && (c.Width <= 0 || c.Width > 12) {
						return nil, fmt.Errorf("%s: invalid column width %v", where, c.Width)
					}
					switch c.Align {
					case "", "left", "center", "right", "justify":
					default:
						return nil, fmt.Errorf("%s: invalid column align %q", where, c.Align)
					}
					cFam := c.Font
					if cFam == "" {
						cFam = family
					}
					if err := validatePrintStyle(where, printTextStyle{
						Font: cFam, Bold: c.Bold, Italic: c.Italic, Underline: c.Underline,
						Color: c.Color, Size: c.Size,
					}, cFam); err != nil {
						return nil, err
					}
				}
			} else {
				if len(b.Rows) > 30 {
					return nil, fmt.Errorf("%s: table needs 1..30 rows", where)
				}
				for _, r := range b.Rows {
					if strings.TrimSpace(r.Label) == "" {
						return nil, fmt.Errorf("%s: table row needs label", where)
					}
					if !reFilterColumn.MatchString(r.Field) {
						return nil, fmt.Errorf("%s: invalid row field %q", where, r.Field)
					}
					if err := validatePrintRef(where, r.Ref); err != nil {
						return nil, err
					}
				}
			}
		case "list":
			if len(b.Items) == 0 || len(b.Items) > 50 {
				return nil, fmt.Errorf("%s: list needs 1..50 items", where)
			}
			for _, item := range b.Items {
				if len(item) > 500 {
					return nil, fmt.Errorf("%s: list item too long (max 500 chars)", where)
				}
			}
		case "image":
			// Картинка — либо файл из папки шаблона (src), либо байты
			// из поля строки (field = id файла из таблицы files того же
			// сервиса). Ровно один из двух — иначе не понять, что рисовать.
			// (Селекторы ролей/версий — отдельной итерацией дизайна печати,
			// без хардкода в shared.)
			if (b.Src == "") == (b.Field == "") {
				return nil, fmt.Errorf("%s: image needs either src or field (not both/neither)", where)
			}
			if b.Src != "" {
				if strings.TrimSpace(b.Src) == "" || b.Src != filepath.Base(b.Src) {
					return nil, fmt.Errorf("%s: image src must be a plain file name", where)
				}
				ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(b.Src), "."))
				if ext != "png" && ext != "jpg" && ext != "jpeg" {
					return nil, fmt.Errorf("%s: image must be png|jpg", where)
				}
			} else if !reFilterColumn.MatchString(b.Field) {
				return nil, fmt.Errorf("%s: invalid image field %q", where, b.Field)
			}
			if b.Width < 5 || b.Width > 190 {
				return nil, fmt.Errorf("%s: image width 5..190 mm required", where)
			}
			if b.Float {
				// Координаты обязательны: ноль означает "забыли указать",
				// а не нулевое смещение. x — от края колонки (уже страницы
				// не бывает), y — вниз от абзаца-якоря.
				if b.X <= 0 || b.X > 210 || b.Y <= 0 || b.Y > 297 {
					return nil, fmt.Errorf("%s: float image needs x in 0..210 and y in 0..297 mm", where)
				}
			}
		default:
			return nil, fmt.Errorf("%s: unknown type %q (want heading|field|table|list|image|text)", where, b.Type)
		}
	}
	return &tpl, nil
}

// printTemplatePath — путь к шаблону (env или дефолт от WORKDIR).
func printTemplatePath() string {
	path := strings.TrimSpace(os.Getenv("PRINT_TEMPLATE"))
	if path == "" {
		path = defaultPrintTemplate
	}
	return path
}

// loadPrintTemplate читает шаблон сервиса с диска (per-request: файл
// маленький, зато без кеш-инвалидации). Нет файла — (nil, false, nil):
// вызывающий отвечает 404.
func loadPrintTemplate() (*printTemplate, bool, error) {
	raw, err := os.ReadFile(printTemplatePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to read print template: %w", err)
	}
	tpl, err := parsePrintTemplate(raw)
	if err != nil {
		return nil, false, err
	}
	return tpl, true, nil
}

// loadPrintImage читает картинку из папки шаблона (только basename,
// без путей — выход за папку невозможен конструкцией).
func loadPrintImage(src string) ([]byte, string, error) {
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(printTemplatePath()), src))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read print image %q: %w", src, err)
	}
	if len(raw) > maxPrintImageBytes {
		return nil, "", fmt.Errorf("print image %q too large (max %d bytes)", src, maxPrintImageBytes)
	}
	if len(raw) < 4 {
		return nil, "", fmt.Errorf("print image %q is not an image", src)
	}
	var ext string
	switch {
	case raw[0] == 0x89 && raw[1] == 'P' && raw[2] == 'N' && raw[3] == 'G':
		ext = "png"
	case raw[0] == 0xFF && raw[1] == 0xD8:
		ext = "jpg"
	default:
		return nil, "", fmt.Errorf("print image %q must be png|jpg (magic mismatch)", src)
	}
	return raw, ext, nil
}

// stringifyPrintValue приводит значение ячейки к строке для печати.
// time.Time (ddate из pgx) — в 02.01.2006, nil — в "".
func stringifyPrintValue(v any) string {
	if v == nil {
		return ""
	}
	if t, ok := v.(time.Time); ok {
		if t.IsZero() {
			return ""
		}
		return t.Format("02.01.2006")
	}
	if s, ok := v.(string); ok {
		return s
	}
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return fmt.Sprintf("%v", v)
}

// renderPrintText подставляет {{field}} из строки (отсутствующее — "").
// Суффикс ! ({{field!}}) помечает обязательное поле: само значение всё равно
// подставляется как есть (пустое — пусто), а проверка обязательности делается
// отдельно в validatePrintRow ДО рендера — молчаливый пропуск запрещён.
func renderPrintText(s string, row map[string]any) string {
	return rePrintPlaceholder.ReplaceAllStringFunc(s, func(m string) string {
		name := rePrintPlaceholder.FindStringSubmatch(m)[1]
		return stringifyPrintValue(row[name])
	})
}

// validatePrintRow проверяет обязательные плейсхолдеры ({{field!}}):
// отсутствующее или пустое значение — это ошибка сборки документа, а не
// пустая строка. Честная ошибка вместо молчаливого неверного документа:
// пользователь раньше замечает и зовёт администратора, а не подписывает
// приказ с дырами. Пустая строка считается отсутствием (пробелы тримятся).
func validatePrintRow(tpl *printTemplate, row map[string]any) error {
	var texts []string
	texts = append(texts, tpl.Title, tpl.Header, tpl.Footer)
	for _, b := range tpl.Blocks {
		texts = append(texts, b.Text)
		texts = append(texts, b.Items...)
		for _, c := range b.Columns {
			texts = append(texts, c.Text)
		}
	}
	seen := make(map[string]bool)
	for _, t := range texts {
		for _, m := range rePrintPlaceholder.FindAllStringSubmatch(t, -1) {
			if m[2] != "!" {
				continue
			}
			name := strings.TrimSpace(m[1])
			if seen[name] {
				continue
			}
			seen[name] = true
			if strings.TrimSpace(stringifyPrintValue(row[name])) == "" {
				return fmt.Errorf("print template: required field %q is empty", name)
			}
		}
	}
	return nil
}

// printID64 приводит id из строки к int64 (pgx даёт int64, JSON —
// float64, query/контекст — строки). Нечисловое/неположительное — false.
func printID64(v any) (int64, bool) {
	switch t := v.(type) {
	case int64:
		if t > 0 {
			return t, true
		}
	case int32:
		if t > 0 {
			return int64(t), true
		}
	case int:
		if t > 0 {
			return int64(t), true
		}
	case float64:
		if t == float64(int64(t)) && t > 0 {
			return int64(t), true
		}
	case string:
		if id, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64); err == nil && id > 0 {
			return id, true
		}
	}
	return 0, false
}

// collectPrintRefs собирает (поле → справочник) из шаблона.
func collectPrintRefs(tpl *printTemplate) [][2]string {
	var out [][2]string
	seen := make(map[string]struct{})
	add := func(field, ref string) {
		if ref == "" {
			return
		}
		if _, dup := seen[field]; dup {
			return
		}
		seen[field] = struct{}{}
		out = append(out, [2]string{field, ref})
	}
	for _, b := range tpl.Blocks {
		switch b.Type {
		case "field":
			add(b.Field, b.Ref)
		case "table":
			for _, c := range b.Columns {
				add(c.Field, c.Ref)
			}
			for _, r := range b.Rows {
				add(r.Field, r.Ref)
			}
		}
	}
	return out
}

// resolvePrintRefs подменяет id именами из справочников (fail-open:
// нет связи/нет env/упал справочник — сырой id остаётся как был).
// Возвращает копию строки, оригинал не трогает.
func resolvePrintRefs(row map[string]any, tpl *printTemplate, client *http.Client) map[string]any {
	out := make(map[string]any, len(row))
	for k, v := range row {
		out[k] = v
	}
	for _, fr := range collectPrintRefs(tpl) {
		field, endpoint := fr[0], fr[1]
		id, ok := printID64(out[field])
		if !ok {
			continue
		}
		base, ok := resolveRefBase(endpoint)
		if !ok {
			continue
		}
		names, err := fetchRefMap(client, base, endpoint, []int64{id})
		if err != nil {
			log.Printf("[print] WARN: %s: %v, leaving raw id", endpoint, err)
			continue
		}
		if name, ok := names[strconv.FormatInt(id, 10)]; ok {
			out[field] = name
		}
	}
	return out
}

// printFile — готовый файл между шагами двухшаговой выдачи.
type printFile struct {
	Bytes    []byte
	Filename string
	Mime     string
	Expires  time.Time
}

type printStore struct {
	mu    sync.Mutex
	files map[string]printFile
}

func newPrintStore() *printStore {
	return &printStore{files: make(map[string]printFile)}
}

// put кладёт файл, возвращает токен; заодно выметает протухшее.
func (s *printStore) put(f printFile) string {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		panic("print store: no randomness: " + err.Error())
	}
	token := hex.EncodeToString(nonce[:])
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	f.Expires = now.Add(printFileTTL)
	s.files[token] = f
	for t, e := range s.files {
		if now.After(e.Expires) {
			delete(s.files, t)
		}
	}
	return token
}

// get забирает файл разово (повторный GET того же токена — 404, файл не лежит вечно).
func (s *printStore) get(token string) (printFile, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.files[token]
	if !ok || time.Now().After(f.Expires) {
		delete(s.files, token)
		return printFile{}, false
	}
	delete(s.files, token)
	return f, true
}

// printResponse — ответ шага "Распечатать" (метаданные, не байты).
type printResponse struct {
	Token    string `json:"token"`
	Size     int    `json:"size"`
	Filename string `json:"filename"`
	Mime     string `json:"mime"`
}
