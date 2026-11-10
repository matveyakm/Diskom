package restController

// Материализация шаблона перед рендером: вычисляет ветки if по уже загруженной
// строке и заполняет дырки slot filler-ами сервиса (лениво — только взятая
// ветка). Рендер (folio/DOCX) про слоты и условия не знает вовсе: получает
// обычный статический шаблон + готовые байты картинок.
// Нет if/slot в шаблоне — materialize не вызывается вообще (старый путь
// побайтово тот же, что раньше).

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"shared/report"
	"strconv"
	"strings"
	"time"
)

// reportMaxPages caps a report fetch walk (page limit 100 → max 20000 rows).
// Больше — ошибка "сузьте фильтры": отчёт не должен молча резать данные.
// Запас под тысячи проводок карьерного регистра; обход последовательный,
// таймаута на весь отчёт нет (5с — только на один запрос).
const reportMaxPages = 200

// reportPageLimit is the page size for report fetches (server max).
const reportPageLimit = 100

// templateNeedsAssembly reports whether materialize must run at all.
func templateNeedsAssembly(tpl *printTemplate) bool {
	var walk func(blocks []printBlock) bool
	walk = func(blocks []printBlock) bool {
		for _, b := range blocks {
			if b.Type == "if" {
				return true
			}
			if b.Slot != "" {
				return true
			}
		}
		return false
	}
	return walk(tpl.Blocks)
}

// materializePrint resolves if-branches against record and fills slot images
// via filler funcs. Missing filler for a TAKEN branch — громкая 500;
// слот в невзятой ветке не трогается вообще. Возвращает статический шаблон,
// байты слот-картинок (ключ "slot:"+name) и несмертельные проблемы данных:
// DOCX-путь подменяет их заглушками (чёрный квадрат / шапка без строк),
// PDF-путь роняет сборку списком.
func (r *CRUDRepository[T]) materializePrint(
	tpl *printTemplate,
	record map[string]any,
	slots map[string]report.SlotFunc,
) (*printTemplate, map[string][]byte, []PrintProblem, error) {
	images := make(map[string][]byte)
	filled := make(map[string]bool)
	var problems []PrintProblem
	blocks, err := r.materializeBlocks(tpl.Blocks, "print template", record, slots, images, filled, &problems)
	if err != nil {
		return nil, nil, nil, err
	}
	out := *tpl
	out.Blocks = blocks
	return &out, images, problems, nil
}

func (r *CRUDRepository[T]) materializeBlocks(
	blocks []printBlock,
	where string,
	record map[string]any,
	slots map[string]report.SlotFunc,
	images map[string][]byte,
	filled map[string]bool,
	problems *[]PrintProblem,
) ([]printBlock, error) {
	out := make([]printBlock, 0, len(blocks))
	for i, b := range blocks {
		here := fmt.Sprintf("%s.block[%d]", where, i+1)
		if b.Type == "if" {
			taken, err := evalPrintIf(b.Equals, record)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", here, err)
			}
			branch := b.Then
			which := "then"
			if !taken {
				which = "else"
				branch = *b.Else
			}
			sub, err := r.materializeBlocks(branch, here+"."+which, record, slots, images, filled, problems)
			if err != nil {
				return nil, err
			}
			out = append(out, sub...)
			continue
		}
		if b.Type == "image" && b.Slot != "" {
			if !filled[b.Slot] {
				fn, ok := slots[b.Slot]
				if !ok {
					return nil, fmt.Errorf("%s: no filler registered for slot %q", here, b.Slot)
				}
				// Параметры — из первого вхождения дырки (имя = те же байты).
				val, err := fn(r.reportContext(record), report.Params(b.SlotParams))
				if err != nil {
					// Данные битые/нет — не роняем: чёрный квадрат в DOCX,
					// список в PDF (см. handlePrintBuild).
					*problems = append(*problems, PrintProblem{
						Kind:    "image",
						Ref:     b.Slot,
						Message: err.Error(),
					})
					images["slot:"+b.Slot] = printFallbackSquare()
					filled[b.Slot] = true
				} else {
					raw, ok := val.([]byte)
					if !ok || len(raw) == 0 {
						return nil, fmt.Errorf("%s: slot %q must return non-empty image bytes", here, b.Slot)
					}
					images["slot:"+b.Slot] = raw
					filled[b.Slot] = true
				}
			}
		}
		if b.Type == "table" && b.Slot != "" {
			// Слот-таблица: строки готовит filler, колонки (стиль) — из
			// шаблона. Дырка с тем же именем переиспользуется как есть
			// (детерминированность сборки важнее экономии вызова).
			fn, ok := slots[b.Slot]
			if !ok {
				return nil, fmt.Errorf("%s: no filler registered for slot %q", here, b.Slot)
			}
			val, err := fn(r.reportContext(record), report.Params(b.SlotParams))
			if err != nil {
				*problems = append(*problems, PrintProblem{
					Kind:    "table",
					Ref:     b.Slot,
					Message: err.Error(),
				})
				nb := b
				nb.RowsData = []map[string]any{}
				out = append(out, nb)
				continue
			}
			rows, err := normalizeSlotRows(val)
			if err != nil {
				return nil, fmt.Errorf("%s: slot %q: %w", here, b.Slot, err)
			}
			nb := b
			nb.RowsData = rows
			out = append(out, nb)
			continue
		}
		if b.Type == "text" && b.Slot != "" {
			// Текст-слот: filler готовит готовый текст (многострочный
			// через \n); рендеры режут его как обычный text-блок.
			// Пустой текст валиден («нет данных» — рисуется только шапка).
			fn, ok := slots[b.Slot]
			if !ok {
				return nil, fmt.Errorf("%s: no filler registered for slot %q", here, b.Slot)
			}
			val, err := fn(r.reportContext(record), report.Params(b.SlotParams))
			if err != nil {
				*problems = append(*problems, PrintProblem{
					Kind:    "text",
					Ref:     b.Slot,
					Message: err.Error(),
				})
				nb := b
				nb.Text = ""
				out = append(out, nb)
				continue
			}
			s, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("%s: slot %q must return text (string)", here, b.Slot)
			}
			nb := b
			nb.Text = s
			out = append(out, nb)
			continue
		}
		out = append(out, b)
	}
	return out, nil
}

// normalizeSlotRows приводит значение table-слота к строкам:
// []map[string]any как есть (пустой/nil — валидное «нет данных», рисуется
// только шапка); []any из map — нормализуется; всё остальное — громкая 500
// (молчаливый пропуск строк запрещён, как и пропуск байтов у картинок).
func normalizeSlotRows(val any) ([]map[string]any, error) {
	if val == nil {
		return nil, nil
	}
	if rows, ok := val.([]map[string]any); ok {
		return rows, nil
	}
	arr, ok := val.([]any)
	if !ok {
		return nil, fmt.Errorf("must return rows ([]map[string]any)")
	}
	rows := make([]map[string]any, 0, len(arr))
	for i, item := range arr {
		rec, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("row %d is not an object", i+1)
		}
		rows = append(rows, rec)
	}
	return rows, nil
}

// evalPrintIf вычисляет ветку: все равенства сошлись — true.
// Отсутствующее поле — громкая 500 (не false: неверный отчёт хуже
// отсутствующего). Несравнимые типы — false (ветка else, не ошибка).
func evalPrintIf(equals map[string]any, record map[string]any) (bool, error) {
	for k, want := range equals {
		got, ok := record[k]
		if !ok {
			return false, fmt.Errorf("no field %q in record", k)
		}
		if !printValuesEqual(got, want) {
			return false, nil
		}
	}
	return true, nil
}

// printValuesEqual — строгое сравнение: числа с числами (int vs float64
// из JSON — равны), строки со строками, bool с bool. Кросс-типы — false.
// time.Time сравнивается как RFC3339-строка (ddate из БД против литерала).
func printValuesEqual(got, want any) bool {
	if gf, ok := printNumber(got); ok {
		wf, ok := printNumber(want)
		return ok && gf == wf
	}
	if gs, ok := got.(string); ok {
		ws, ok := want.(string)
		return ok && gs == ws
	}
	if gb, ok := got.(bool); ok {
		wb, ok := want.(bool)
		return ok && gb == wb
	}
	// Даты из БД (time.Time) против литералов RFC3339/день.
	if gt, ok := got.(time.Time); ok {
		ws, ok := want.(string)
		return ok && gt.UTC().Format(time.RFC3339) == ws
	}
	return false
}

// printNumber приводит любой числовой скаляр к float64.
func printNumber(v any) (float64, bool) {
	switch t := v.(type) {
	case int:
		return float64(t), true
	case int8:
		return float64(t), true
	case int16:
		return float64(t), true
	case int32:
		return float64(t), true
	case int64:
		return float64(t), true
	case uint:
		return float64(t), true
	case uint8:
		return float64(t), true
	case uint16:
		return float64(t), true
	case uint32:
		return float64(t), true
	case uint64:
		return float64(t), true
	case float32:
		return float64(t), true
	case float64:
		return t, true
	default:
		return 0, false
	}
}

// reportContext строит фреймворк-контекст заполнителя: сырая строка +
// http-клиент сервиса (тот же timeout, что у expand).
func (r *CRUDRepository[T]) reportContext(record map[string]any) report.Ctx {
	return &reportClient{client: r.httpClient, record: record}
}

// reportClient — report.Ctx поверх http: файлы и строки чужих сервисов
// читаются через их публичные эндпоинты (никакого прямого доступа к чужим БД).
type reportClient struct {
	client *http.Client
	record map[string]any
}

func (c *reportClient) Record() map[string]any { return c.record }

func (c *reportClient) refURL(ref string) (string, error) {
	base, ok := resolveRefBase(ref)
	if !ok {
		return "", fmt.Errorf("no REF_%s configured", strings.ToUpper(ref))
	}
	return base + "/api/" + ref, nil
}

func (c *reportClient) getJSON(url string, v any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got %s", resp.Status)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxRefBodyBytes)).Decode(v); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// Fetch — все строки list-эндпоинта (проход страницам limit=100 до короткой
// страницы или капа; больше капа — громкая ошибка вместо молчаливой обрезки).
func (c *reportClient) Fetch(ref string, q map[string]string) ([]map[string]any, error) {
	base, err := c.refURL(ref)
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for page := 1; page <= reportMaxPages; page++ {
		vals := url.Values{}
		for k, v := range q {
			vals.Set(k, v)
		}
		vals.Set("page", strconv.Itoa(page))
		vals.Set("limit", strconv.Itoa(reportPageLimit))
		var env struct {
			Data []map[string]any `json:"data"`
		}
		if err := c.getJSON(base+"?"+vals.Encode(), &env); err != nil {
			return nil, fmt.Errorf("%s: %w", ref, err)
		}
		out = append(out, env.Data...)
		if len(env.Data) < reportPageLimit {
			return out, nil
		}
	}
	return nil, fmt.Errorf("%s: too many rows (over %d), narrow filters", ref, reportMaxPages*reportPageLimit)
}

// FetchOne — ровно одна строка: bare-объект или конверт {data:[...]} из одной.
// Ноль или несколько — громкая ошибка (неоднозначность хуже отсутствия).
func (c *reportClient) FetchOne(ref string, q map[string]string) (map[string]any, error) {
	base, err := c.refURL(ref)
	if err != nil {
		return nil, err
	}
	vals := url.Values{}
	for k, v := range q {
		vals.Set(k, v)
	}
	var raw map[string]any
	if err := c.getJSON(base+"?"+vals.Encode(), &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", ref, err)
	}
	if data, ok := raw["data"]; ok {
		rows, ok := data.([]any)
		if !ok {
			return nil, fmt.Errorf("%s: bad data envelope", ref)
		}
		if len(rows) != 1 {
			return nil, fmt.Errorf("%s: rows %d, want 1", ref, len(rows))
		}
		rec, ok := rows[0].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: bad row shape", ref)
		}
		return rec, nil
	}
	return raw, nil
}

// FilesOf — метаданные файлов чужого документа (?id_document= — только мета).
func (c *reportClient) FilesOf(ref string, docID int64) ([]report.FileMeta, error) {
	base, err := c.refURL(ref)
	if err != nil {
		return nil, err
	}
	var env struct {
		Data []struct {
			ID       any    `json:"id"`
			FileName string `json:"file_name"`
			Mime     string `json:"mime"`
			Size     int64  `json:"size"`
		} `json:"data"`
	}
	u := base + "/files?id_document=" + strconv.FormatInt(docID, 10)
	if err := c.getJSON(u, &env); err != nil {
		return nil, fmt.Errorf("%s files: %w", ref, err)
	}
	out := make([]report.FileMeta, 0, len(env.Data))
	for _, f := range env.Data {
		id, ok := printID64(f.ID)
		if !ok {
			return nil, fmt.Errorf("%s files: bad id %v", ref, f.ID)
		}
		out = append(out, report.FileMeta{ID: id, Name: f.FileName, Mime: f.Mime, Size: f.Size})
	}
	return out, nil
}

// FileBytes — байты одного чужого файла (?content=) + mime из заголовка.
// Принадлежность и magic проверяет отдающий сервис (свои 400/404/500).
func (c *reportClient) FileBytes(ref string, fileID, docID int64) ([]byte, string, error) {
	base, err := c.refURL(ref)
	if err != nil {
		return nil, "", err
	}
	u := base + "/files?content=" + strconv.FormatInt(fileID, 10)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("%s file #%d: %w", ref, fileID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("%s file #%d: got %s", ref, fileID, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, fileMaxBytes()+fileBodySlackBytes))
	if err != nil {
		return nil, "", fmt.Errorf("%s file #%d: %w", ref, fileID, err)
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("%s file #%d: empty", ref, fileID)
	}
	return data, resp.Header.Get("Content-Type"), nil
}
