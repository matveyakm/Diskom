package restController

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

const validPrintTemplate = `{
  "title": "Приказ {{dnumber}}",
  "blocks": [
    {"type": "heading", "text": "Шапка"},
    {"type": "field", "label": "Номер", "field": "dnumber"},
    {"type": "field", "label": "Дата", "field": "ddate"},
    {"type": "table", "title": "Сотрудник",
     "columns": [{"field": "last_name", "label": "Фамилия"}]},
    {"type": "text", "text": "Основание: {{note}}"}
  ]
}`

func TestParsePrintTemplate(t *testing.T) {
	tpl, err := parsePrintTemplate([]byte(validPrintTemplate))
	if err != nil {
		t.Fatalf("valid template: %v", err)
	}
	if tpl.Title != "Приказ {{dnumber}}" || len(tpl.Blocks) != 5 {
		t.Fatalf("unexpected parse: %+v", tpl)
	}
	for _, bad := range []string{
		`{}`,
		`{"title": "x", "blocks": []}`,
		`{"title": "x", "blocks": [{"type": "wat", "text": "y"}]}`,
		`{"title": "x", "blocks": [{"type": "field", "label": "N"}]}`,
		`{"title": "x", "blocks": [{"type": "field", "label": "N", "field": "a-b"}]}`,
		`{"title": "x", "blocks": [{"type": "table", "columns": []}]}`,
		`{"title": "x", "blocks": [{"type": "text"}]}`,
		`{"title": "x", "blocks": [{"type": "text", "text": "y", "align": "diagonal"}]}`,
		`not json`,
	} {
		if _, err := parsePrintTemplate([]byte(bad)); err == nil {
			t.Fatalf("want error for %q", bad)
		}
	}
}

func samplePrintRow() map[string]any {
	return map[string]any{
		"dnumber":   "ПР-71",
		"ddate":     time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC),
		"last_name": "Ибрагимов",
		"note":      "a&b <c>",
	}
}

func TestRenderPrintText(t *testing.T) {
	row := samplePrintRow()
	if got := renderPrintText("Приказ {{dnumber}} от {{ddate}}", row); got != "Приказ ПР-71 от 21.10.2026" {
		t.Fatalf("text = %q", got)
	}
	if got := renderPrintText("X{{missing}}Y", row); got != "XY" {
		t.Fatalf("missing field = %q, want empty", got)
	}
	if got := stringifyPrintValue(nil); got != "" {
		t.Fatalf("nil = %q", got)
	}
}

func TestRenderPrintPDF(t *testing.T) {
	tpl, err := parsePrintTemplate([]byte(validPrintTemplate))
	if err != nil {
		t.Fatal(err)
	}
	out, err := renderPrintPDF(tpl, samplePrintRow(), nil)
	if err != nil {
		t.Fatalf("pdf: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF")) {
		t.Fatal("not a PDF")
	}
	if len(out) < 1000 {
		t.Fatalf("suspiciously small PDF: %d bytes", len(out))
	}
}

func TestRenderPrintDOCX(t *testing.T) {
	tpl, err := parsePrintTemplate([]byte(validPrintTemplate))
	if err != nil {
		t.Fatal(err)
	}
	out, err := renderPrintDOCX(tpl, samplePrintRow(), nil)
	if err != nil {
		t.Fatalf("docx: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatalf("not a zip: %v", err)
	}
	var doc string
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(rc)
			rc.Close()
			doc = buf.String()
		}
	}
	if doc == "" {
		t.Fatal("no word/document.xml")
	}
	for _, want := range []string{"ПР-71", "Ибрагимов", "21.10.2026", "a&amp;b &lt;c&gt;"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("document.xml lacks %q", want)
		}
	}
	if strings.Contains(doc, "a&b") {
		t.Fatal("unescaped & in document.xml")
	}
	if !strings.Contains(doc, `<w:jc w:val="center"/>`) {
		t.Fatal("no centered paragraph (align=center not rendered)")
	}
}

// Пустой field-блок не печатается: приказы не содержат строк вида "Основание: ".
func TestPrintEmptyFieldSkipped(t *testing.T) {
	tpl, err := parsePrintTemplate([]byte(`{
	  "title": "Приказ",
	  "blocks": [
	    {"type": "field", "label": "Основание", "field": "note"},
	    {"type": "field", "label": "Номер", "field": "dnumber"}
	  ]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	row := map[string]any{"note": "", "dnumber": "ПР-71"}
	out, err := renderPrintDOCX(tpl, row, nil)
	if err != nil {
		t.Fatalf("docx: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatalf("not a zip: %v", err)
	}
	var doc string
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(rc)
			rc.Close()
			doc = buf.String()
		}
	}
	if strings.Contains(doc, "Основание") {
		t.Fatal("empty field block must be skipped")
	}
	if !strings.Contains(doc, "ПР-71") {
		t.Fatal("non-empty field block must stay")
	}
	// PDF тоже собирается без пустых строк (проверяем сборку, не пиксели).
	if _, err := renderPrintPDF(tpl, row, nil); err != nil {
		t.Fatalf("pdf: %v", err)
	}
}

func TestPrintStore(t *testing.T) {
	s := newPrintStore()
	token := s.put(printFile{Bytes: []byte("x"), Filename: "a.pdf", Mime: printPdfMime})
	if len(token) != 32 {
		t.Fatalf("token = %q", token)
	}
	f, ok := s.get(token)
	if !ok || string(f.Bytes) != "x" {
		t.Fatalf("get = %+v,%v", f, ok)
	}
	if _, ok := s.get(token); ok {
		t.Fatal("second get must 404 (one-shot)")
	}
	if _, ok := s.get("nope"); ok {
		t.Fatal("unknown token must 404")
	}
}

func TestPrintColumnWidths(t *testing.T) {
	mk := func(weights ...float64) []printBlockColumn {
		cols := make([]printBlockColumn, len(weights))
		for i, w := range weights {
			cols[i] = printBlockColumn{Field: "f", Label: "l", Width: w}
		}
		return cols
	}
	for _, cols := range [][]printBlockColumn{
		mk(0), mk(0, 0), mk(0, 0, 0), mk(2, 1, 1), mk(12),
	} {
		sum := 0
		for _, w := range printColumnWidths(cols) {
			sum += w
		}
		if sum != 12 {
			t.Fatalf("widths(%v) sum = %d, want 12", cols, sum)
		}
	}
}

func v2PrintTemplate() string {
	return `{
  "title": "Приказ {{dnumber}}",
  "header": "Комитет",
  "footer": "Стр.",
  "blocks": [
    {"type": "heading", "text": "Шапка", "align": "center", "italic": true, "color": "333333", "size": 14},
    {"type": "field", "label": "Номер", "field": "dnumber", "bold": true, "align": "right"},
    {"type": "field", "label": "Отдел", "field": "id_department", "ref": "department_directory"},
    {"type": "text", "text": "Принять.", "align": "justify", "underline": true, "indent": 10, "firstLine": 5},
    {"type": "list", "ordered": true, "items": ["п.1 {{note}}", "п.2"]},
    {"type": "list", "items": ["• {{missing}}", "ок"]},
    {"type": "table", "title": "Т", "columns": [{"field": "last_name", "label": "Ф", "width": 2, "align": "center"}]},
    {"type": "table", "title": "Р", "rows": [{"label": "Д", "field": "ddate"}]},
    {"type": "image", "src": "logo.png", "width": 40, "align": "center"},
    {"type": "text", "text": "Подпись"}
  ]
}`
}

func TestParsePrintV2(t *testing.T) {
	tpl, err := parsePrintTemplate([]byte(v2PrintTemplate()))
	if err != nil {
		t.Fatalf("valid v2: %v", err)
	}
	if len(tpl.Blocks) != 10 {
		t.Fatalf("blocks = %d", len(tpl.Blocks))
	}
	for _, bad := range []string{
		`{"title": "x", "blocks": [{"type": "text", "text": "y", "color": "zzz"}]}`,
		`{"title": "x", "blocks": [{"type": "text", "text": "y", "size": 100}]}`,
		`{"title": "x", "blocks": [{"type": "text", "text": "y", "indent": -1}]}`,
		`{"title": "x", "blocks": [{"type": "image", "src": "../x.png", "width": 40}]}`,
		`{"title": "x", "blocks": [{"type": "image", "src": "x.gif", "width": 40}]}`,
		`{"title": "x", "blocks": [{"type": "image", "src": "x.png", "width": 500}]}`,
		`{"title": "x", "blocks": [{"type": "field", "label": "N", "field": "a", "ref": "Bad-Ref!"}]}`,
		`{"title": "x", "blocks": [{"type": "table", "columns": [{"field": "a", "label": "L", "width": 99}]}]}`,
		`{"title": "x", "blocks": [{"type": "table", "columns": [{"field": "a"}], "rows": [{"label": "L", "field": "a"}]}]}`,
		`{"title": "x", "blocks": [{"type": "list", "items": []}]}`,
	} {
		if _, err := parsePrintTemplate([]byte(bad)); err == nil {
			t.Fatalf("want error for %q", bad)
		}
	}
}

// testPNGBytes генерирует 2x1 PNG без внешних файлов.
func testPNGBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	img.Set(1, 0, color.RGBA{0, 0, 255, 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func unzipDoc(t *testing.T, out []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatalf("not a zip: %v", err)
	}
	parts := make(map[string]string)
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(rc)
		rc.Close()
		parts[f.Name] = buf.String()
	}
	return parts
}

func TestRenderPrintV2(t *testing.T) {
	tpl, err := parsePrintTemplate([]byte(v2PrintTemplate()))
	if err != nil {
		t.Fatal(err)
	}
	row := map[string]any{
		"dnumber": "ПР-71", "note": "тест", "last_name": "Ибрагимов",
		"ddate":         time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC),
		"id_department": int64(0),
	}
	images := map[string][]byte{"logo.png": testPNGBytes(t)}
	out, err := renderPrintDOCX(tpl, row, images)
	if err != nil {
		t.Fatalf("docx: %v", err)
	}
	parts := unzipDoc(t, out)
	doc := parts["word/document.xml"]
	for _, want := range []string{
		"w:color", "w:i", "w:u", // стили
		`w:jc w:val="right"`, `w:jc w:val="both"`, `w:jc w:val="center"`, // выравнивания
		`w:ind w:left=`, "w:firstLine", // отступы
		"1. п.1", "2. п.2", "• ок", // списки (пустой пункт пропущен)
		"rIdImg1", // картинка
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("document.xml lacks %q", want)
		}
	}
	// Вертикальная таблица: label и значение.
	if !strings.Contains(doc, "Р") || !strings.Contains(doc, "21.10.2026") {
		t.Fatal("vertical rows table missing")
	}
	if strings.Contains(doc, "• {{missing}}") || strings.Contains(doc, "missing") {
		t.Fatal("empty list item must be skipped")
	}
	if _, ok := parts["word/media/rIdImg1.png"]; !ok {
		t.Fatal("no media part for image")
	}
	if !strings.Contains(parts["word/_rels/document.xml.rels"], "media/rIdImg1.png") {
		t.Fatal("media rel missing")
	}
	if !strings.Contains(parts["word/footer1.xml"], "PAGE") {
		t.Fatal("no PAGE field in footer")
	}
	if !strings.Contains(parts["word/header1.xml"], "Комитет") {
		t.Fatal("no header part")
	}
	if !strings.Contains(parts["word/document.xml"], "headerReference") {
		t.Fatal("header not referenced")
	}
	if _, err := renderPrintPDF(tpl, row, images); err != nil {
		t.Fatalf("pdf: %v", err)
	}
}

func TestResolvePrintRefs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": [{"id": 19, "name": "Иванов"}]}`))
	}))
	defer srv.Close()
	t.Setenv("REF_EMPLOYEE_DIRECTORY", srv.URL)

	tpl, err := parsePrintTemplate([]byte(`{"title": "x", "blocks": [
	  {"type": "field", "label": "С", "field": "id_employee", "ref": "employee_directory"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	got := resolvePrintRefs(map[string]any{"id_employee": int64(19), "x": 1}, tpl, srv.Client())
	if got["id_employee"] != "Иванов" {
		t.Fatalf("ref not resolved: %v", got["id_employee"])
	}
	if got["x"] != 1 {
		t.Fatal("unrelated field changed")
	}
	// Без связи — сырой id, без падения.
	got = resolvePrintRefs(map[string]any{"id_employee": int64(0)}, tpl, srv.Client())
	if got["id_employee"] != int64(0) {
		t.Fatalf("zero id must stay raw: %v", got["id_employee"])
	}
	// Нет env — сырой id, без падения.
	os.Unsetenv("REF_EMPLOYEE_DIRECTORY")
	got = resolvePrintRefs(map[string]any{"id_employee": int64(19)}, tpl, srv.Client())
	if got["id_employee"] != int64(19) {
		t.Fatalf("missing env must leave raw id: %v", got["id_employee"])
	}
}
