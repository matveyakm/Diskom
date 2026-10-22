package restController

import (
	"archive/zip"
	"bytes"
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
	out, err := renderPrintPDF(tpl, samplePrintRow())
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
	out, err := renderPrintDOCX(tpl, samplePrintRow())
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
	for n, want := range map[int]int{1: 12, 2: 12, 3: 12, 5: 12} {
		sum := 0
		for _, w := range printColumnWidths(n) {
			sum += w
		}
		if sum != want {
			t.Fatalf("widths(%d) sum = %d", n, sum)
		}
	}
}
