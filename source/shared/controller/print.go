package restController

// Печать документов: JSON-шаблон → PDF (maroto) и DOCX (stdlib).
//
// Один шаблон описывает оба формата (поведение — в данных): новый
// документ/отчёт = новый файл templates/print.json в папке сервиса,
// кода трогать не надо. Шаблон печётся в образ (WORKDIR сервиса),
// путь — env PRINT_TEMPLATE (дефолт "templates/print.json").
// Нет файла — честный 404, а не падение.
//
// Формат шаблона v1:
//
//	{
//	  "title": "Приказ о приёме {{dnumber}}",
//   "blocks": [
//     {"type": "heading", "text": "...", "align": "center"},
//     {"type": "field", "label": "Номер", "field": "dnumber"},
//     {"type": "table", "title": "...",
//      "columns": [{"field": "last_name", "label": "Фамилия"}]},
//     {"type": "text", "text": "Основание: {{note}}", "align": "left"}
//   ]
//
// {{field}} резолвится из строки документа, даты — в 02.01.2006,
// отсутствующее поле — пустая строка. Пустой field-блок (незаполненное
// поле) в документ не попадает вовсе — настоящие приказы не печатают
// строки вида "Примечание: ". align — только left|center.
//	}
//
// {{field}} резолвится из строки документа, даты — в 02.01.2006,
// отсутствующее поле — пустая строка (презентация fail-open: печать
// не должна падать 500 из-за пустого поля). Структура валидируется
// fail-closed (битый шаблон — 500 с понятной причиной, это ошибка
// оператора, запечённая в образ).

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
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
)

var (
	rePrintPlaceholder = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)
	printTokenStore    = newPrintStore()
)

// printBlock — один презентационный блок шаблона.
type printBlock struct {
	Type    string             `json:"type"`
	Text    string             `json:"text,omitempty"`
	Align   string             `json:"align,omitempty"`
	Label   string             `json:"label,omitempty"`
	Field   string             `json:"field,omitempty"`
	Title   string             `json:"title,omitempty"`
	Columns []printBlockColumn `json:"columns,omitempty"`
}

type printBlockColumn struct {
	Field string `json:"field"`
	Label string `json:"label"`
}

// printTemplate — распарсенный и проверенный шаблон.
type printTemplate struct {
	Title  string       `json:"title"`
	Blocks []printBlock `json:"blocks"`
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
	if strings.TrimSpace(tpl.Title) == "" {
		return nil, fmt.Errorf("print template: title is required")
	}
	if len(tpl.Blocks) == 0 || len(tpl.Blocks) > 50 {
		return nil, fmt.Errorf("print template: blocks 1..50 required, got %d", len(tpl.Blocks))
	}
	for i, b := range tpl.Blocks {
		where := fmt.Sprintf("print template: block %d", i+1)
		if b.Align != "" && b.Align != "left" && b.Align != "center" {
			return nil, fmt.Errorf("%s: invalid align %q (want left|center)", where, b.Align)
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
			if len(b.Columns) == 0 || len(b.Columns) > 12 {
				return nil, fmt.Errorf("%s: table needs 1..12 columns", where)
			}
			for _, c := range b.Columns {
				if strings.TrimSpace(c.Label) == "" {
					return nil, fmt.Errorf("%s: table column needs label", where)
				}
				if !reFilterColumn.MatchString(c.Field) {
					return nil, fmt.Errorf("%s: invalid column field %q", where, c.Field)
				}
			}
		default:
			return nil, fmt.Errorf("%s: unknown type %q (want heading|field|table|text)", where, b.Type)
		}
	}
	return &tpl, nil
}

// loadPrintTemplate читает шаблон сервиса с диска (per-request: файл
// маленький, зато правка = только пересборка образа, без кеш-инвалидации).
// Нет файла — (nil, false, nil): вызывающий отвечает 404.
func loadPrintTemplate() (*printTemplate, bool, error) {
	path := strings.TrimSpace(os.Getenv("PRINT_TEMPLATE"))
	if path == "" {
		path = defaultPrintTemplate
	}
	raw, err := os.ReadFile(path)
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
func renderPrintText(s string, row map[string]any) string {
	return rePrintPlaceholder.ReplaceAllStringFunc(s, func(m string) string {
		name := rePrintPlaceholder.FindStringSubmatch(m)[1]
		return stringifyPrintValue(row[name])
	})
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

// get забирает файл разово (скачивание одноразовое: повторный GET
// того же токена — 404, файл не лежит вечно).
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
