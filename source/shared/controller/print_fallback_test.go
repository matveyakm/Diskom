package restController

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"strings"
	"testing"

	"shared/report"
)

// Сборщик собирает ВСЕ пустые strict, а не роняет на первом; лейблы —
// человеческие имена из шаблона, техническое имя — запасной вариант.
func TestCollectPrintProblems(t *testing.T) {
	tpl, err := parsePrintTemplate([]byte(`{"blocks": [
	  {"type": "field", "label": "Основание", "field": "note"},
	  {"type": "text", "text": "Принять {{last_genitive!}} {{first_genitive!}} {{middle_name}} в состав"},
	  {"type": "table", "columns": [{"field": "d", "label": "Дата", "end_point": "x"}]}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	got := collectPrintProblems(tpl, map[string]any{"last_genitive": "Иванова"})
	if len(got) != 1 || got[0].Ref != "first_genitive" || got[0].Kind != "field" {
		t.Fatalf("want [first_genitive], got %+v", got)
	}
	got = collectPrintProblems(tpl, map[string]any{})
	if len(got) != 2 {
		t.Fatalf("want both strict fields, got %+v", got)
	}
	// Без strict — пусто, даже если всё пусто.
	plain, _ := parsePrintTemplate([]byte(`{"blocks": [{"type": "text", "text": "Привет {{name}}"}]}`))
	if got := collectPrintProblems(plain, map[string]any{}); len(got) != 0 {
		t.Fatalf("non-strict must not complain: %+v", got)
	}
}

// Подстановка: окружающий текст цел, ошибка — отдельным красным куском
// с человеческим именем; без ошибок — как раньше (склейка без изменений).
func TestRenderPrintTextSpans(t *testing.T) {
	labels := map[string]string{"first_genitive": "Имя (род.)"}
	segs := renderPrintTextSpans("Принять {{last_genitive!}} {{first_genitive!}} в состав", map[string]any{"last_genitive": "Иванова"}, labels)
	if len(segs) != 5 || !segs[3].Error || segs[3].Text != "ОШИБКА (Имя (род.))" {
		t.Fatalf("bad spans: %+v", segs)
	}
	if segs[0].Text != "Принять " || segs[1].Text != "Иванова" || segs[2].Text != " " || segs[4].Text != " в состав" {
		t.Fatalf("context lost: %+v", segs)
	}
	for i, sg := range segs {
		if i != 3 && sg.Error {
			t.Fatalf("only middle span errors: %+v", segs)
		}
	}
	// Техническое имя, если лейбла нет.
	segs = renderPrintTextSpans("{{x!}}", map[string]any{}, nil)
	if len(segs) != 1 || !segs[0].Error || segs[0].Text != "ОШИБКА (x)" {
		t.Fatalf("want technical name fallback: %+v", segs)
	}
	// Без ошибок — та же строка, что renderPrintText.
	segs = renderPrintTextSpans("A {{b}} C", map[string]any{"b": "B"}, labels)
	if joinPrintSpans(segs) != "A B C" {
		t.Fatalf("join mismatch: %+v", segs)
	}
	for _, sg := range segs {
		if sg.Error {
			t.Fatalf("no errors expected: %+v", segs)
		}
	}
}

// Красный квадрат: валидный PNG ожидаемого размера, детерминированный.
// Красный и непрозрачный — маркер должен бросаться в глаза.
func TestPrintFallbackSquare(t *testing.T) {
	a := printFallbackSquare()
	b := printFallbackSquare()
	if !bytes.Equal(a, b) {
		t.Fatal("square must be deterministic")
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(a))
	if err != nil {
		t.Fatalf("not a PNG: %v", err)
	}
	if cfg.Width != 100 || cfg.Height != 100 {
		t.Fatalf("want 100x100, got %dx%d", cfg.Width, cfg.Height)
	}
	img, _, err := image.Decode(bytes.NewReader(a))
	if err != nil {
		t.Fatal(err)
	}
	r, g, bl, al := img.At(50, 50).RGBA()
	if r>>8 != 255 || g>>8 != 0 || bl>>8 != 0 || al>>8 != 255 {
		t.Fatalf("want opaque red, got %d,%d,%d,%d", r>>8, g>>8, bl>>8, al>>8)
	}
	if _, _, err := downscalePrintImage([]byte("not an image at all............")); err == nil {
		t.Fatal("want error on garbage")
	}
}

// DOCX с ошибками: красные раны только на подстановках, данные на месте.
func TestRenderPrintDOCXFallback(t *testing.T) {
	tpl, err := parsePrintTemplate([]byte(`{"font": "times", "blocks": [
	  {"type": "heading", "text": "О приказе"},
	  {"type": "text", "text": "Принять {{last_genitive!}} {{first_genitive!}} в состав"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	row := map[string]any{"last_genitive": "Иванова"}
	labels := printFieldLabels(tpl)
	var body strings.Builder
	_ = body
	var out strings.Builder
	for _, b := range tpl.Blocks {
		out.WriteString(string(printDocxParaSpans(renderPrintTextSpans(b.Text, row, labels), b.printTextStyle, 11, false, "Times New Roman")))
	}
	doc := out.String()
	if !strings.Contains(doc, "Иванова") {
		t.Fatal("good value lost")
	}
	if !strings.Contains(doc, "ОШИБКА (first_genitive)") {
		t.Fatal("no error marker")
	}
	if got := strings.Count(doc, `w:color w:val="FF0000"`); got != 1 {
		t.Fatalf("want exactly 1 red run, got %d", got)
	}
}

// Слот-филлер упал: проблема собрана (не 500), шапка без байтов — квадрат,
// таблица без строк — только шапка.
func TestMaterializeCollectsProblems(t *testing.T) {
	bad := func(ctx report.Ctx, p report.Params) (any, error) {
		return nil, errors.New("boom")
	}
	_ = bad
	tpl, err := parsePrintTemplate([]byte(`{"blocks": [
	  {"type": "image", "slot": "emblem", "width": 22},
	  {"type": "table", "slot": "staff", "header": true, "columns": [{"field": "d", "label": "Дата"}]}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	r := &CRUDRepository[struct{}]{ReportSlots: report.Slots{
		"emblem": func(ctx report.Ctx, p report.Params) (any, error) { return nil, errors.New("boom") },
		"staff":  func(ctx report.Ctx, p report.Params) (any, error) { return nil, errors.New("boom") },
	}}
	_, images, problems, err := r.materializePrint(tpl, map[string]any{}, r.ReportSlots)
	if err != nil {
		t.Fatalf("filler errors must not abort: %v", err)
	}
	if len(problems) != 2 {
		t.Fatalf("want 2 problems, got %+v", problems)
	}
	if len(images["slot:emblem"]) == 0 {
		t.Fatal("square missing for failed image slot")
	}
	if _, err := png.DecodeConfig(bytes.NewReader(images["slot:emblem"])); err != nil {
		t.Fatalf("square must be a PNG: %v", err)
	}
}
