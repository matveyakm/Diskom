package restController

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"shared/report"
	"strings"
	"testing"
)

func mustParsePrint(t *testing.T, raw string) *printTemplate {
	t.Helper()
	tpl, err := parsePrintTemplate([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v\n%s", err, raw)
	}
	return tpl
}

func TestPrintIfValidation(t *testing.T) {
	mustParsePrint(t, `{"blocks": [
	  {"type": "if", "equals": {"status": 2}, "then": [{"type": "text", "text": "A"}],
	   "else": [{"type": "text", "text": "B"}]}
	]}`)
	mustParsePrint(t, `{"blocks": [
	  {"type": "if", "equals": {"status": 2}, "then": [{"type": "text", "text": "A"}], "else": []}
	]}`)
	// Вложенный if — ок.
	mustParsePrint(t, `{"blocks": [
	  {"type": "if", "equals": {"status": 2},
	   "then": [{"type": "if", "equals": {"x": "y"},
	             "then": [{"type": "text", "text": "A"}], "else": [{"type": "text", "text": "B"}]}],
	   "else": []}
	]}`)
  // Слот-картинка — ок.
  mustParsePrint(t, `{"blocks": [
    {"type": "image", "slot": "emblem", "params": {"committee": "1"}, "width": 22}
  ]}`)
  // Текст-слот — ок; текст+слот вместе — нет.
  mustParsePrint(t, `{"blocks": [
    {"type": "text", "slot": "career", "size": 14}
  ]}`)
	for _, bad := range []string{
		// if без if-объекта/equals/then/else.
		`{"blocks": [{"type": "if", "then": [{"type": "text", "text": "A"}], "else": []}]}`,
		`{"blocks": [{"type": "if", "equals": {}, "then": [{"type": "text", "text": "A"}], "else": []}]}`,
		`{"blocks": [{"type": "if", "equals": {"status": 2}, "else": []}]}`,
		`{"blocks": [{"type": "if", "equals": {"status": 2}, "then": [{"type": "text", "text": "A"}]}}}`,
		// Нескалярное значение и плохое имя поля.
		`{"blocks": [{"type": "if", "equals": {"status": [1, 2]}, "then": [{"type": "text", "text": "A"}], "else": []}]}`,
		`{"blocks": [{"type": "if", "equals": {"a-b": 1}, "then": [{"type": "text", "text": "A"}], "else": []}]}`,
		// ref у if запрещён.
		`{"blocks": [{"type": "if", "ref": "x", "equals": {"status": 2}, "then": [{"type": "text", "text": "A"}], "else": []}]}`,
		// Битый then рекурсивно ловится.
		`{"blocks": [{"type": "if", "equals": {"status": 2}, "then": [{"type": "text", "text": ""}], "else": []}]}`,
		// Два источника картинки и слот не на image.
		`{"blocks": [{"type": "image", "src": "a.png", "slot": "x", "width": 10}]}`,
		`{"blocks": [{"type": "text", "text": "A", "slot": "x"}]}`,
		`{"blocks": [{"type": "image", "slot": "a b", "width": 10}]}`,
	} {
		if _, err := parsePrintTemplate([]byte(bad)); err == nil {
			t.Fatalf("must fail: %s", bad)
		}
	}
}

func materializeForTest(t *testing.T, raw string, record map[string]any, slots report.Slots) (*printTemplate, map[string][]byte) {
	t.Helper()
	tpl := mustParsePrint(t, raw)
	if slots == nil {
		slots = report.Slots{}
	}
	r := &CRUDRepository[struct{}]{ReportSlots: slots}
	got, images, _, err := r.materializePrint(tpl, record, r.ReportSlots)
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	return got, images
}

func TestMaterializeIf(t *testing.T) {
	raw := `{"blocks": [
	  {"type": "if", "equals": {"status": 2}, "then": [{"type": "text", "text": "YES"}],
	   "else": [{"type": "text", "text": "NO"}]}
	]}`
	got, _ := materializeForTest(t, raw, map[string]any{"status": float64(2)}, nil)
	if len(got.Blocks) != 1 || got.Blocks[0].Text != "YES" {
		t.Fatalf("taken then = %+v", got.Blocks)
	}
	// int16 из драйвера против float64 из JSON — равны.
	got, _ = materializeForTest(t, raw, map[string]any{"status": int16(2)}, nil)
	if len(got.Blocks) != 1 || got.Blocks[0].Text != "YES" {
		t.Fatalf("int16 taken = %+v", got.Blocks)
	}
	got, _ = materializeForTest(t, raw, map[string]any{"status": float64(1)}, nil)
	if len(got.Blocks) != 1 || got.Blocks[0].Text != "NO" {
		t.Fatalf("taken else = %+v", got.Blocks)
	}
	// Отсутствующее поле — громкая ошибка, а не false.
	tpl := mustParsePrint(t, raw)
	r := &CRUDRepository[struct{}]{}
	if _, _, _, err := r.materializePrint(tpl, map[string]any{}, r.ReportSlots); err == nil {
		t.Fatal("missing field must fail loudly")
	}
	// Вложенный if.
	nested := `{"blocks": [
	  {"type": "if", "equals": {"a": 1},
	   "then": [{"type": "if", "equals": {"b": 1},
	             "then": [{"type": "text", "text": "AB"}], "else": [{"type": "text", "text": "A"}]}],
	   "else": [{"type": "text", "text": "N"}]}
	]}`
	got, _ = materializeForTest(t, nested, map[string]any{"a": float64(1), "b": float64(1)}, nil)
	if len(got.Blocks) != 1 || got.Blocks[0].Text != "AB" {
		t.Fatalf("nested = %+v", got.Blocks)
	}
	got, _ = materializeForTest(t, nested, map[string]any{"a": float64(0)}, nil)
	if len(got.Blocks) != 1 || got.Blocks[0].Text != "N" {
		t.Fatalf("nested else = %+v", got.Blocks)
	}
	// Шаблон без if/slot — не сборка (тот же указатель блоков, без движка).
	plain := mustParsePrint(t, `{"blocks": [{"type": "text", "text": "A"}]}`)
	if templateNeedsAssembly(plain) {
		t.Fatal("plain template must not need assembly")
	}
	assembled := mustParsePrint(t, raw)
	if !templateNeedsAssembly(assembled) {
		t.Fatal("if template must need assembly")
	}
}

func TestMaterializeSlot(t *testing.T) {
	raw := `{"blocks": [
	  {"type": "image", "slot": "emblem", "params": {"committee": "1"}, "width": 22},
	  {"type": "image", "slot": "emblem", "width": 22}
	]}`
	png := []byte{0x89, 'P', 'N', 'G'}
	calls := 0
	slots := report.Slots{
		"emblem": func(ctx report.Ctx, p report.Params) (any, error) {
			calls++
			if p["committee"] != "1" {
				t.Fatalf("params = %v", p)
			}
			return png, nil
		},
	}
	got, images := materializeForTest(t, raw, map[string]any{}, slots)
	if calls != 1 {
		t.Fatalf("filler called %d times, want 1 (dedupe)", calls)
	}
	if string(images["slot:emblem"]) != string(png) {
		t.Fatal("slot bytes not stored")
	}
	if len(got.Blocks) != 2 {
		t.Fatalf("blocks = %d, want 2 (slot stays, bytes aside)", len(got.Blocks))
	}
	// Нет filler-а — громко.
	tpl := mustParsePrint(t, raw)
	r := &CRUDRepository[struct{}]{ReportSlots: report.Slots{}}
	if _, _, _, err := r.materializePrint(tpl, map[string]any{}, r.ReportSlots); err == nil {
		t.Fatal("missing filler must fail loudly")
	}
	// Не-байты — громко.
	r2 := &CRUDRepository[struct{}]{ReportSlots: report.Slots{
		"emblem": func(ctx report.Ctx, p report.Params) (any, error) {
			return "not-bytes", nil
		},
	}}
	if _, _, _, err := r2.materializePrint(mustParsePrint(t, raw), map[string]any{}, r2.ReportSlots); err == nil {
		t.Fatal("non-bytes slot must fail")
	}
}

func TestPrintTableSlotValidation(t *testing.T) {
	mustParsePrint(t, `{"blocks": [
	  {"type": "table", "slot": "staff", "header": false, "borders": false,
	   "columns": [{"field": "d", "label": "Дата"}, {"field": "n", "label": "N"}]}
	]}`)
	for _, bad := range []string{
		// Слот без колонок и слот+строки.
		`{"blocks": [{"type": "table", "slot": "s", "rows": [{"label": "A", "field": "a"}]}]}`,
		`{"blocks": [{"type": "table", "slot": "s"}]}`,
		// ref и строгие плейсхолдеры в колонках слот-таблицы запрещены.
		`{"blocks": [{"type": "table", "slot": "s", "columns": [{"field": "a", "label": "A", "ref": "x"}]}]}`,
		`{"blocks": [{"type": "table", "slot": "s", "columns": [{"text": "{{a!}}", "label": "A"}]}]}`,
	} {
		if _, err := parsePrintTemplate([]byte(bad)); err == nil {
			t.Fatalf("must fail: %s", bad)
		}
	}
}

func TestMaterializeTableSlot(t *testing.T) {
	raw := `{"blocks": [
	  {"type": "table", "slot": "staff", "header": true, "borders": false,
	   "columns": [{"field": "d", "label": "Дата"}, {"field": "n", "label": "N"}]}
	]}`
	rows := []map[string]any{
		{"d": "21.10.2026", "n": float64(10)},
		{"d": "12.10.2026", "n": float64(9)},
	}
	r := &CRUDRepository[struct{}]{ReportSlots: report.Slots{
		"staff": func(ctx report.Ctx, p report.Params) (any, error) { return rows, nil },
	}}
	got, _, _, err := r.materializePrint(mustParsePrint(t, raw), map[string]any{}, r.ReportSlots)
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if len(got.Blocks) != 1 || len(got.Blocks[0].RowsData) != 2 {
		t.Fatalf("rows not materialized: %+v", got.Blocks)
	}
	// Шаблон без if/slot не требует сборки; со слотом — требует.
	plain := mustParsePrint(t, `{"blocks": [{"type": "table", "columns": [{"field": "a", "label": "A"}]}]}`)
	if templateNeedsAssembly(plain) {
		t.Fatal("plain table must not need assembly")
	}
	// Не-строки — громко; пусто — валидно (только шапка).
	r2 := &CRUDRepository[struct{}]{ReportSlots: report.Slots{
		"s": func(ctx report.Ctx, p report.Params) (any, error) { return "nope", nil },
	}}
	slotRaw := mustParsePrint(t, `{"blocks": [
	  {"type": "table", "slot": "s", "columns": [{"field": "a", "label": "A"}]}]}`)
	if _, _, _, err := r2.materializePrint(slotRaw, map[string]any{}, r2.ReportSlots); err == nil {
		t.Fatal("non-rows must fail")
	}
	r3 := &CRUDRepository[struct{}]{ReportSlots: report.Slots{
		"s": func(ctx report.Ctx, p report.Params) (any, error) { return []map[string]any{}, nil },
	}}
	got3, _, _, err3 := r3.materializePrint(slotRaw, map[string]any{}, r3.ReportSlots)
	if err3 != nil {
		t.Fatalf("empty materialize: %v", err3)
	}
	if len(got3.Blocks) != 1 || got3.Blocks[0].RowsData == nil || len(got3.Blocks[0].RowsData) != 0 {
		t.Fatalf("empty must stay empty-non-nil: %+v", got3.Blocks)
	}
	// Рендеры собирают слот-таблицу (шапка + 2 строки, плейсхолдеры по ряду).
	tpl, _, _, err := r.materializePrint(mustParsePrint(t, `{"blocks": [
	  {"type": "table", "slot": "s", "header": true, "borders": false,
	   "columns": [{"field": "d", "label": "Дата"}, {"text": "N={{n}}", "label": "N"}]}
	]}`), map[string]any{}, report.Slots{
		"s": func(ctx report.Ctx, p report.Params) (any, error) { return rows, nil },
	})
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	docx, err := renderPrintDOCX(tpl, map[string]any{}, map[string][]byte{})
	if err != nil {
		t.Fatalf("docx: %v", err)
	}
	doc := mustZipDoc(t, docx)
	for _, want := range []string{"21.10.2026", "N=10", "N=9", "Дата"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("docx lacks %q", want)
		}
	}
	pdf, err := renderPrintPDF(tpl, map[string]any{}, map[string][]byte{})
	if err != nil {
		t.Fatalf("pdf: %v", err)
	}
	if len(pdf) < 1000 {
		t.Fatalf("pdf too small: %d", len(pdf))
	}
}

func TestMaterializeTextSlot(t *testing.T) {
	raw := `{"blocks": [{"type": "text", "slot": "career", "size": 14}]}`
	r := &CRUDRepository[struct{}]{ReportSlots: report.Slots{
		"career": func(ctx report.Ctx, p report.Params) (any, error) {
			return "Иванов Иван Иванович\n- В период с 02.03.2026 по настоящее время является Активистом", nil
		},
	}}
	got, _, _, err := r.materializePrint(mustParsePrint(t, raw), map[string]any{}, r.ReportSlots)
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if len(got.Blocks) != 1 || !strings.Contains(got.Blocks[0].Text, "Иванов") {
		t.Fatalf("text not materialized: %+v", got.Blocks)
	}
	// Не-строка — громко; пусто — валидно; ошибка filler-а — problems, не 500.
	r2 := &CRUDRepository[struct{}]{ReportSlots: report.Slots{
		"s": func(ctx report.Ctx, p report.Params) (any, error) { return 42, nil },
	}}
	slotRaw := mustParsePrint(t, `{"blocks": [{"type": "text", "slot": "s", "size": 14}]}`)
	if _, _, _, err := r2.materializePrint(slotRaw, map[string]any{}, r2.ReportSlots); err == nil {
		t.Fatal("non-string must fail")
	}
	r3 := &CRUDRepository[struct{}]{ReportSlots: report.Slots{
		"s": func(ctx report.Ctx, p report.Params) (any, error) { return "", nil },
	}}
	got3, _, _, err3 := r3.materializePrint(slotRaw, map[string]any{}, r3.ReportSlots)
	if err3 != nil || len(got3.Blocks) != 1 || got3.Blocks[0].Text != "" {
		t.Fatalf("empty must stay empty: %+v %v", got3.Blocks, err3)
	}
	r4 := &CRUDRepository[struct{}]{ReportSlots: report.Slots{
		"s": func(ctx report.Ctx, p report.Params) (any, error) { return nil, errors.New("no data") },
	}}
	got4, _, problems, err4 := r4.materializePrint(slotRaw, map[string]any{}, r4.ReportSlots)
	if err4 != nil {
		t.Fatalf("filler error must not fail: %v", err4)
	}
	if len(got4.Blocks) != 1 || got4.Blocks[0].Text != "" {
		t.Fatalf("failed slot must be empty: %+v", got4.Blocks)
	}
	if len(problems) != 1 || problems[0].Kind != "text" || problems[0].Ref != "s" {
		t.Fatalf("want one text problem, got %+v", problems)
	}
	// Рендеры режут \n как обычный текст.
	docx, err := renderPrintDOCX(got, map[string]any{}, map[string][]byte{})
	if err != nil {
		t.Fatalf("docx: %v", err)
	}
	doc := mustZipDoc(t, docx)
	for _, want := range []string{"Иванов Иван Иванович", "02.03.2026", "Активистом"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("docx lacks %q", want)
		}
	}
}

func mustZipDoc(t *testing.T, raw []byte) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatalf("not a zip: %v", err)
	}
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(rc)
			rc.Close()
			return buf.String()
		}
	}
	t.Fatal("no document.xml")
	return ""
}
func TestMaterializeSlotLazy(t *testing.T) {
	// Дырка в невзятой ветке не вызывается вообще (черновик без логотипа
	// не должен падать из-за невидимой шапки).
	raw := `{"blocks": [
	  {"type": "if", "equals": {"status": 2},
	   "then": [{"type": "image", "slot": "emblem", "width": 22}],
	   "else": [{"type": "text", "text": "draft"}]}
	]}`
	called := false
	slots := report.Slots{
		"emblem": func(ctx report.Ctx, p report.Params) (any, error) {
			called = true
			return nil, fmt.Errorf("must not be called")
		},
	}
	got, images := materializeForTest(t, raw, map[string]any{"status": float64(1)}, slots)
	if called {
		t.Fatal("untaken slot must not be filled")
	}
	if len(images) != 0 || len(got.Blocks) != 1 || got.Blocks[0].Text != "draft" {
		t.Fatalf("else branch = %+v, images = %v", got.Blocks, images)
	}
}

func TestReportClientFetch(t *testing.T) {
	page := func(n int) []map[string]any {
		out := make([]map[string]any, n)
		for i := range out {
			out[i] = map[string]any{"id": float64(i + 1)}
		}
		return out
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/thing", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Query().Get("page")
		var data []map[string]any
		if p == "" || p == "1" {
			data = page(100) // полная страница → walk дальше
		} else if p == "2" {
			data = page(3)
		} else {
			data = page(1) // ровно одна строка — для FetchOne
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	})
	mux.HandleFunc("/api/single", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": float64(7)})
	})
	mux.HandleFunc("/api/empty", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	})
	mux.HandleFunc("/api/two", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": page(2)})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	t.Setenv("REF_THING", srv.URL)
	t.Setenv("REF_SINGLE", srv.URL)
	t.Setenv("REF_EMPTY", srv.URL)
	t.Setenv("REF_TWO", srv.URL)

	c := &reportClient{client: srv.Client(), record: map[string]any{}}
	rows, err := c.Fetch("thing", map[string]string{"status": "2"})
	if err != nil || len(rows) != 103 {
		t.Fatalf("fetch walk = %d,%v", len(rows), err)
	}
	one, err := c.FetchOne("single", nil)
	if err != nil || one["id"] != float64(7) {
		t.Fatalf("fetch one bare = %v,%v", one, err)
	}
	one, err = c.FetchOne("thing", map[string]string{"page": "3"})
	if err != nil {
		t.Fatalf("fetch one envelope: %v", err)
	}
	if one["id"] != float64(1) {
		t.Fatalf("fetch one envelope = %v", one)
	}
	if _, err := c.FetchOne("empty", nil); err == nil {
		t.Fatal("empty must fail")
	}
	if _, err := c.FetchOne("two", nil); err == nil {
		t.Fatal("two rows must fail")
	}
	if _, err := c.Fetch("missing", nil); err == nil {
		t.Fatal("missing REF_ must fail")
	}
	if _, err := c.FetchOne("missing", nil); err == nil {
		t.Fatal("missing REF_ must fail")
	}
}

func TestReportClientFiles(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/doc/files", func(w http.ResponseWriter, r *http.Request) {
		if id := r.URL.Query().Get("content"); id != "" {
			if id != "9" {
				http.Error(w, "nope", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte{0x89, 'P', 'N', 'G'})
			return
		}
		if r.URL.Query().Get("id_document") == "" {
			http.Error(w, "missing", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{
			map[string]any{"id": float64(9), "file_name": "a.png", "mime": "image/png", "size": float64(4)},
		}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	t.Setenv("REF_DOC", srv.URL)
	c := &reportClient{client: srv.Client(), record: map[string]any{}}
	files, err := c.FilesOf("doc", 5)
	if err != nil || len(files) != 1 || files[0].ID != 9 || files[0].Name != "a.png" {
		t.Fatalf("files = %+v,%v", files, err)
	}
	data, mime, err := c.FileBytes("doc", 9, 5)
	if err != nil || mime != "image/png" || len(data) != 4 {
		t.Fatalf("bytes = %d,%q,%v", len(data), mime, err)
	}
	if _, _, err := c.FileBytes("doc", 404, 5); err == nil {
		t.Fatal("404 must fail")
	}
}
