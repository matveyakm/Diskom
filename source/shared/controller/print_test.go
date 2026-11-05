package restController

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

var (
	rePrintID    = regexp.MustCompile(`<wp:docPr id="(\d+)"`)
	rePrintCnV   = regexp.MustCompile(`<pic:cNvPr id="(\d+)"`)
	rePrintFld   = regexp.MustCompile(`<w:fldSimple[^>]*>.*?</w:fldSimple>|<w:fldSimple[^>]*/>`)
	rePrintRel   = regexp.MustCompile(`<Relationship[^>]*Target="([^"]+)"[^>]*>`)
	rePrintEmbed = regexp.MustCompile(`r:embed="(rId[^"]+)"`)
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
	// Title необязателен: шапка организации идёт блоками до заголовка.
	noTitle, err := parsePrintTemplate([]byte(
		`{"blocks": [{"type": "heading", "text": "ПРИКАЗ", "align": "center"}]}`))
	if err != nil {
		t.Fatalf("title-less template must parse: %v", err)
	}
	if noTitle.Title != "" {
		t.Fatalf("title must stay empty: %q", noTitle.Title)
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

func TestPrintImageFilesTag(t *testing.T) {
	// Селектор files.tag: картинка из файлов документа по роли (документ
	// своей таблицы файлов не знает). Пилотный шаблон символики обязан
	// парситься: шапка — tag logo_old, тело — tag logo.
	raw, err := os.ReadFile("../../document/ms_symbolism_document/templates/print.json")
	if err != nil {
		t.Fatalf("symbolism template unreadable: %v", err)
	}
	tpl, err := parsePrintTemplate(raw)
	if err != nil {
		t.Fatalf("symbolism template invalid: %v", err)
	}
	var tags []string
	for _, b := range tpl.Blocks {
		if b.Type == "image" {
			if b.Files == nil {
				t.Fatalf("symbolism image block must use files selector, got %+v", b)
			}
			tags = append(tags, b.Files.Tag)
			if b.imageKey() != "tag:"+b.Files.Tag {
				t.Fatalf("imageKey = %q", b.imageKey())
			}
		}
	}
	if len(tags) != 2 || tags[0] != "logo_old" || tags[1] != "logo" {
		t.Fatalf("symbolism image tags = %v, want [logo_old logo]", tags)
	}
	// Ровно один источник: src+files.tag и пустота — мимо.
	for _, bad := range []string{
		`{"blocks": [{"type": "image", "src": "a.png", "files": {"tag": "logo"}, "width": 10}]}`,
		`{"blocks": [{"type": "image", "width": 10}]}`,
		`{"blocks": [{"type": "image", "field": "id_logo", "files": {"tag": "logo"}, "width": 10}]}`,
		`{"blocks": [{"type": "image", "files": {"tag": "a b"}, "width": 10}]}`,
		`{"blocks": [{"type": "image", "files": {}, "width": 10}]}`,
	} {
		if _, err := parsePrintTemplate([]byte(bad)); err == nil {
			t.Fatalf("must fail: %s", bad)
		}
	}
	// Валидный files-блок парсится, рендерам без байтов — пропуск молча.
	ok, err := parsePrintTemplate([]byte(`{"blocks": [{"type": "image", "files": {"tag": "logo"}, "width": 10}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := renderPrintDOCX(ok, map[string]any{}, map[string][]byte{}); err != nil {
		t.Fatalf("docx without tag bytes must skip: %v", err)
	}
	if _, err := renderPrintPDF(ok, map[string]any{}, map[string][]byte{}); err != nil {
		t.Fatalf("pdf without tag bytes must skip: %v", err)
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

func TestPrintRequiredPlaceholder(t *testing.T) {
	// Пилотный шаблон приёма обязан парситься: именно его битый JSON
	// однажды дал 500 в проде вместо документа.
	raw, err := os.ReadFile("../../document/ms_acceptance_document/templates/print.json")
	if err != nil {
		t.Fatalf("pilot template unreadable: %v", err)
	}
	if _, err := parsePrintTemplate(raw); err != nil {
		t.Fatalf("pilot template invalid: %v", err)
	}
	// {{x!}} резолвится как обычно, обязательность проверяется отдельно.
	row := map[string]any{"a": "A", "b": ""}
	if got := renderPrintText("{{a!}}|{{b!}}", row); got != "A|" {
		t.Fatalf("strict renders value as-is, got %q", got)
	}
	tpl, err := parsePrintTemplate([]byte(`{"blocks": [
	  {"type": "text", "text": "A {{a!}} B {{b!}}"}
	]}`))
	if err != nil {
		t.Fatalf("valid strict template: %v", err)
	}
	if err := validatePrintRow(tpl, row); err == nil {
		t.Fatal("want error for empty required field b")
	} else if !strings.Contains(err.Error(), `"b"`) {
		t.Fatalf("error must name the field, got: %v", err)
	}
	if err := validatePrintRow(tpl, map[string]any{"a": "A", "b": "x"}); err != nil {
		t.Fatalf("filled required fields must pass: %v", err)
	}
	// Без ! — tolerant, как раньше: пустое растворяется, ошибки нет.
	plain, err := parsePrintTemplate([]byte(`{"blocks": [{"type": "text", "text": "A {{b}}"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := validatePrintRow(plain, row); err != nil {
		t.Fatalf("non-strict empty must not fail: %v", err)
	}
	// Мусор вместо имени — не плейсхолдер вообще, остаётся текстом.
	if got := renderPrintText("{{a-b}} {{a|b}}", row); got != "{{a-b}} {{a|b}}" {
		t.Fatalf("garbage must stay literal, got %q", got)
	}
	// Заполненные склонения — родительный в документе, а не именительный.
	gen, err := parsePrintTemplate([]byte(`{"blocks": [
	  {"type": "text", "text": "Принять {{ln!}} {{fn}}."}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	full := map[string]any{"ln": "Иванова", "fn": "Анну"}
	if err := validatePrintRow(gen, full); err != nil {
		t.Fatalf("filled strict must pass: %v", err)
	}
	if got := renderPrintText("Принять {{ln!}} {{fn}}.", full); got != "Принять Иванова Анну." {
		t.Fatalf("genitive render = %q", got)
	}
}

func TestPrintPDFParaSplits(t *testing.T) {
	// \n режется на отдельные абзацы (явный разрыв строки, как <w:br/>).
	fonts := printFolioFontsOnce()
	tpl, err := parsePrintTemplate([]byte(`{"blocks": [{"type": "text", "text": "A"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	elems := printFolioPara(fonts, tpl, printTextStyle{}, "A\nB\nC", 11, false, 0)
	if len(elems) != 3 {
		t.Fatalf("want 3 elements for 2 breaks, got %d", len(elems))
	}
	// Таб в середине строки раскрывается в пробелы (~12.7 мм).
	if got := expandPrintTabs("\tX", 14); got != strings.Repeat(" ", 10)+"X" {
		t.Fatalf("tab expansion = %q", got)
	}
	if got := expandPrintTabs("ab", 14); got != "ab" {
		t.Fatalf("no tab = unchanged, got %q", got)
	}
	// Ведущие табы отщепляются в firstLine-индент (пробелы в начале
	// строки при justify схлопываются, индент переживает любое выравнивание).
	if n, rest := splitLeadingTabs("\t\tX"); n != 2 || rest != "X" {
		t.Fatalf("splitLeadingTabs = %d,%q", n, rest)
	}
	if n, rest := splitLeadingTabs("A\tB"); n != 0 || rest != "A\tB" {
		t.Fatalf("mid-line tab must stay, got %d,%q", n, rest)
	}
	if n, rest := splitLeadingTabs(""); n != 0 || rest != "" {
		t.Fatalf("empty = 0,empty, got %d,%q", n, rest)
	}
}

func TestPrintPilotTemplate(t *testing.T) {
	// Пилотный шаблон обязан парситься и валидироваться: именно его битый
	// JSON однажды дал 500 в проде вместо документа.
	raw, err := os.ReadFile("../../document/ms_acceptance_document/templates/print.json")
	if err != nil {
		t.Fatalf("pilot template unreadable: %v", err)
	}
	tpl, err := parsePrintTemplate(raw)
	if err != nil {
		t.Fatalf("pilot template invalid: %v", err)
	}
	if tpl.PageNum != "header" {
		t.Fatalf("pilot pageNum = %q, want header", tpl.PageNum)
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
		sum := 0.0
		for _, w := range printFolioColWidths(cols, 465.0) {
			sum += w
		}
		if sum < 464.9 || sum > 465.1 {
			t.Fatalf("widths(%v) sum = %v, want 465", cols, sum)
		}
	}
}

func v2PrintTemplate() string {
	return `{
  "title": "Приказ {{dnumber}}",
  "header": "Комитет",
  "footer": "Стр.",
  "blocks": [
    {"type": "heading", "text": "Шапка", "font": "times", "align": "center", "italic": true, "color": "333333", "size": 14},
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
		`{"title": "x", "blocks": [{"type": "text", "text": "y", "font": "comic"}]}`,
		`{"title": "x", "blocks": [{"type": "text", "text": "y", "italic": true}]}`,
		`{"title": "x", "blocks": [{"type": "text", "text": "y", "size": 100}]}`,
		`{"title": "x", "blocks": [{"type": "text", "text": "y", "indent": -1}]}`,
		`{"title": "x", "blocks": [{"type": "text", "text": "y", "rightIndent": 101}]}`,
		`{"title": "x", "blocks": [{"type": "text", "text": "y", "spaceAfter": -1}]}`,
		`{"title": "x", "blocks": [{"type": "text", "text": "y", "line": 3}]}`,
		`{"title": "x", "blocks": [{"type": "image", "src": "../x.png", "width": 40}]}`,
		`{"title": "x", "blocks": [{"type": "image", "src": "x.gif", "width": 40}]}`,
		`{"title": "x", "blocks": [{"type": "image", "src": "x.png", "width": 500}]}`,
		`{"title": "x", "blocks": [{"type": "field", "label": "N", "field": "a", "ref": "Bad-Ref!"}]}`,
		`{"title": "x", "blocks": [{"type": "table", "columns": [{"field": "a"}], "rows": [{"label": "L", "field": "a"}]}]}`,
		`{"title": "x", "blocks": [{"type": "table", "columns": [{"label": "L", "text": "t", "field": "a"}]}]}`,
		`{"title": "x", "blocks": [{"type": "table", "columns": [{"label": "L", "field": "a-b"}]}]}`,
		`{"title": "x", "blocks": [{"type": "table", "columns": [{"label": "", "text": "t"}]}]}`,
		`{"title": "x", "blocks": [{"type": "list", "items": []}]}`,
		`{"title": "x", "blocks": [{"type": "image", "src": "x.png", "width": 40}], "pageNum": "sideways"}`,
		`{"title": "x", "blocks": [{"type": "text", "text": "y"}], "margins": {"left": 99}}`,
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
	// Каждый XML-кусок обязан быть well-formed, иначе Word предлагает
	// восстановить документ (урок: незакрытый pic:spPr).
	for name, body := range parts {
		if !strings.HasSuffix(name, ".xml") && !strings.HasSuffix(name, ".rels") {
			continue
		}
		dec := xml.NewDecoder(strings.NewReader(body))
		for {
			if _, err := dec.Token(); err != nil {
				if err.Error() == "EOF" {
					break
				}
				t.Fatalf("%s is not well-formed XML: %v", name, err)
			}
		}
	}
	// Все цели rels обязаны существовать в пакете (урок: висячие ссылки
	// Word тоже чинит предложением восстановить документ).
	checkRels := func(relsPath, base string) {
		t.Helper()
		body, ok := parts[relsPath]
		if !ok {
			t.Fatalf("missing %s", relsPath)
		}
		for _, m := range rePrintRel.FindAllStringSubmatch(body, -1) {
			if strings.Contains(m[0], "Mode=") {
				continue // external
			}
			target := m[1]
			if strings.HasPrefix(target, "/") {
				target = strings.TrimPrefix(target, "/")
			} else if base != "" {
				target = base + "/" + target
			}
			if _, ok := parts[target]; !ok {
				t.Fatalf("%s references missing part %q", relsPath, m[1])
			}
		}
	}
	checkRels("_rels/.rels", "")
	checkRels("word/_rels/document.xml.rels", "word")
	// Все r:embed обязаны указывать на существующий media-файл.
	for _, m := range rePrintEmbed.FindAllStringSubmatch(parts["word/document.xml"], -1) {
		rid := m[1]
		found := false
		for _, r := range rePrintRel.FindAllStringSubmatch(parts["word/_rels/document.xml.rels"], -1) {
			if strings.Contains(r[0], `Id="`+rid+`"`) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("r:embed %q has no relationship", rid)
		}
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
		"ddate": time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC),
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
	if !strings.Contains(doc, "Times New Roman") {
		t.Fatal("no Times New Roman in docx")
	}
	if _, err := renderPrintPDF(tpl, row, images); err != nil {
		t.Fatalf("pdf: %v", err)
	}

	// TestPrintLayoutV2 проверяет точные метрики примера: отступы, интервалы,
	// borderless-таблицу без шапки, текстовые колонки, поля страницы.
	v2layout, err := parsePrintTemplate([]byte(`{
	  "font": "times", "pageNum": "header",
	  "margins": {"left": 30, "top": 17.5, "right": 15, "bottom": 20},
	  "blocks": [
	    {"type": "heading", "text": "Шапка", "align": "left", "size": 14, "rightIndent": 85, "firstLine": 12.5, "spaceBefore": 10.59, "spaceAfter": 10.59},
	    {"type": "table", "header": false, "borders": false, "columns": [
	      {"text": "01.01.2026", "align": "left", "color": "333333"},
	      {"text": "№ ПР-1", "align": "right", "underline": true, "bold": true}
	    ]},
	    {"type": "text", "text": "абзац", "align": "justify", "size": 14, "line": 1.5},
	    {"type": "field", "label": "Основания", "field": "missing_field"}
	  ]
	}`))
	if err != nil {
		t.Fatalf("valid layout template: %v", err)
	}
	lrow := map[string]any{"dnumber": "ПР-1"}
	lout, err := renderPrintDOCX(v2layout, lrow, nil)
	if err != nil {
		t.Fatalf("docx: %v", err)
	}
	lparts := unzipDoc(t, lout)
	ldoc := lparts["word/document.xml"]
	for _, want := range []string{
		`w:right="4819"`, `w:firstLine="708"`,
		`w:before="600" w:after="600"`,
		`w:val="both"`, `w:line="360" w:lineRule="auto"`,
		`№ ПР-1`, `w:u w:val="single"`,
		`w:color w:val="333333"`, `w:b/`,
		`w:pgMar w:top="992" w:right="850" w:bottom="1134" w:left="1701"`,
	} {
		if !strings.Contains(ldoc, want) {
			t.Fatalf("document.xml lacks %q", want)
		}
	}
	if strings.Contains(ldoc, "Основания") {
		t.Fatal("empty field must be skipped")
	}
	if strings.Contains(ldoc, "tblBorders") {
		t.Fatal("borders must be absent with borders:false")
	}
	if _, ok := lparts["word/footer1.xml"]; ok {
		t.Fatal("no footer expected with pageNum=header")
	}
	if !strings.Contains(lparts["word/header1.xml"], "PAGE") {
		t.Fatal("no PAGE field in header")
	}
	if _, err := renderPrintPDF(v2layout, lrow, nil); err != nil {
		t.Fatalf("pdf: %v", err)
	}
	// Плавающая картинка: якорь относительно колонки/абзаца (как пишет
	// сам Word), а не абсолют от страницы. Табы и переносы строк в тексте.
	fly, err := parsePrintTemplate([]byte(`{
	  "blocks": [
	    {"type": "image", "src": "a.png", "width": 35, "float": true, "x": 95, "y": 8},
	    {"type": "text", "text": "\tA\tB\n\tC", "align": "justify"}
	  ]
	}`))
	if err != nil {
		t.Fatalf("valid float template: %v", err)
	}
	fout, err := renderPrintDOCX(fly, map[string]any{}, map[string][]byte{"a.png": testPNGBytes(t)})
	if err != nil {
		t.Fatalf("docx float: %v", err)
	}
	fparts := unzipDoc(t, fout)
	fdoc := fparts["word/document.xml"]
	for _, want := range []string{
		`wp:anchor`, `relativeFrom="column"`, `relativeFrom="paragraph"`,
		`<wp:posOffset>3420000</wp:posOffset>`, `<wp:posOffset>288000</wp:posOffset>`,
		`<wp:wrapNone/>`,
		`<w:tab/>`, `<w:br/>`, `w:val="both"`,
		`w:line="1" w:lineRule="exact"`, // контейнер якоря нулевой высоты
	} {
		if !strings.Contains(fdoc, want) {
			t.Fatalf("document.xml lacks %q", want)
		}
	}
	if strings.Contains(fdoc, "<wp:inline") {
		t.Fatal("float image must not use inline placement")
	}
	if strings.Contains(fdoc, `relativeFrom="page"`) {
		t.Fatal("float anchor must be column/paragraph-relative, not page-absolute")
	}
	// Неймспейсы картинок обязаны быть объявлены в корне: без них Word
	// открывает документ с «не удалось прочитать содержимое» и чинит
	// сносом кусков (Go-парсер в unzipDoc такое пропускает молча).
	for _, want := range []string{
		`xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"`,
		`xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"`,
	} {
		if !strings.Contains(fdoc, want) {
			t.Fatalf("document.xml lacks namespace %q (Word shows unreadable-content warning)", want)
		}
	}
	// Валидация float: координаты за пределами A4 запрещены.
	for _, bad := range []string{
		`{"blocks": [{"type": "image", "src": "a.png", "width": 35, "float": true, "x": 500, "y": 10}]}`,
		`{"blocks": [{"type": "image", "src": "a.png", "width": 35, "float": true, "x": 10}]}`,
	} {
		if _, err := parsePrintTemplate([]byte(bad)); err == nil {
			t.Fatalf("want error for %q", bad)
		}
	}
	if _, err := renderPrintPDF(fly, map[string]any{}, map[string][]byte{"a.png": testPNGBytes(t)}); err != nil {
		t.Fatalf("pdf float: %v", err)
	}
	// docPr/cNvPr id обязаны быть уникальны (дубли Word чинит восстановлением).
	seenIDs := make(map[string]bool)
	for _, m := range append(
		rePrintID.FindAllStringSubmatch(ldoc, -1),
		rePrintCnV.FindAllStringSubmatch(ldoc, -1)...) {
		if seenIDs[m[1]] {
			t.Fatalf("duplicate drawing id %q", m[1])
		}
		seenIDs[m[1]] = true
	}
	// Пустых fldSimple быть не должно (только с кэшированным значением).
	for _, m := range rePrintFld.FindAllString(ldoc, -1) {
		if !strings.Contains(m, "<w:r>") {
			t.Fatalf("empty fldSimple: %.60q", m)
		}
	}
	// Два изображения: id обязаны различаться, оба media на месте.
	twoImg, err := parsePrintTemplate([]byte(`{
	  "blocks": [
	    {"type": "image", "src": "a.png", "width": 22, "align": "center"},
	    {"type": "image", "src": "b.png", "width": 35, "align": "left"}
	  ]
	}`))
	if err != nil {
		t.Fatalf("valid image template: %v", err)
	}
	imgBytes := testPNGBytes(t)
	iout, err := renderPrintDOCX(twoImg, map[string]any{}, map[string][]byte{"a.png": imgBytes, "b.png": imgBytes})
	if err != nil {
		t.Fatalf("docx with images: %v", err)
	}
	iparts := unzipDoc(t, iout)
	idoc := iparts["word/document.xml"]
	ids := rePrintID.FindAllStringSubmatch(idoc, -1)
	if len(ids) != 2 || ids[0][1] == ids[1][1] {
		t.Fatalf("drawing ids must be unique, got %v", ids)
	}
	// docPr/@id и pic:cNvPr/@id делят область уникальности документа:
	// совпадение значений — «не удалось прочитать содержимое» в Word.
	allIDs := make(map[string]bool)
	for _, m := range append(
		rePrintID.FindAllStringSubmatch(idoc, -1),
		rePrintCnV.FindAllStringSubmatch(idoc, -1)...) {
		if allIDs[m[1]] {
			t.Fatalf("duplicate drawing id %q across docPr/cNvPr", m[1])
		}
		allIDs[m[1]] = true
	}
	// Default на расширение — один, дубли путают строгие читалки.
	if c := strings.Count(iparts["[Content_Types].xml"], `Extension="png"`); c != 1 {
		t.Fatalf("want single png Default, got %d", c)
	}
	for _, want := range []string{"word/media/rIdImg1.png", "word/media/rIdImg2.png"} {
		if _, ok := iparts[want]; !ok {
			t.Fatalf("missing media part %q", want)
		}
	}
	if _, err := renderPrintPDF(twoImg, map[string]any{}, map[string][]byte{"a.png": imgBytes, "b.png": imgBytes}); err != nil {
		t.Fatalf("pdf with images: %v", err)
	}
	// Arial-italic запрещён валидацией (нет начертания в бинарнике).
	// Пустой title не печатается отдельным абзацем.
	bare, err := parsePrintTemplate([]byte(
		`{"blocks": [{"type": "heading", "text": "ПРИКАЗ", "align": "center"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	bareOut, err := renderPrintDOCX(bare, row, nil)
	if err != nil {
		t.Fatalf("docx: %v", err)
	}
	bareParts := unzipDoc(t, bareOut)
	if c := strings.Count(bareParts["word/document.xml"], "<w:p>"); c != 1 {
		t.Fatalf("want exactly the heading paragraph, got %d <w:p>", c)
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
