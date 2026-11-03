package restController

// PDF-рендер печатной формы на maroto v2 (тот же движок и шрифты,
// что у ms_job_report: Arial + кириллица из коробки).

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strconv"
	"strings"
	"sync"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	mimage "github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/fontrepository"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

//go:embed fonts/ARIAL.TTF
var printArialRegular []byte

//go:embed fonts/ARIALBD.TTF
var printArialBold []byte

//go:embed fonts/TINOS-Regular.ttf
var printTinosRegular []byte

//go:embed fonts/TINOS-Bold.ttf
var printTinosBold []byte

//go:embed fonts/TINOS-Italic.ttf
var printTinosItalic []byte

//go:embed fonts/TINOS-BoldItalic.ttf
var printTinosBoldItalic []byte

// Шрифты грузятся один раз на процесс (per-request грузить — расточительно,
// maroto-инстанс при этом всё равно новый на каждый документ).
var printFontsOnce = sync.OnceValue(func() []entity.CustomFont {
	customFonts, err := fontrepository.New().
		AddUTF8FontFromBytes("Arial", fontstyle.Normal, printArialRegular).
		AddUTF8FontFromBytes("Arial", fontstyle.Bold, printArialBold).
		AddUTF8FontFromBytes("Times", fontstyle.Normal, printTinosRegular).
		AddUTF8FontFromBytes("Times", fontstyle.Bold, printTinosBold).
		AddUTF8FontFromBytes("Times", fontstyle.Italic, printTinosItalic).
		AddUTF8FontFromBytes("Times", fontstyle.BoldItalic, printTinosBoldItalic).
		Load()
	if err != nil {
		panic("print: load fonts: " + err.Error())
	}
	return customFonts
})

// print_pdffamily: Tinos метрически совместим с Times New Roman,
// в PDF фигурирует как "Times" (лицензионно чисто, Apache 2.0).
const printFontTimes = "Times"

// printAlign маппит DSL-выравнивание в maroto (дефолт — переданный).
func printAlign(a string, def align.Type) align.Type {
	switch a {
	case "center":
		return align.Center
	case "right":
		return align.Right
	case "justify":
		return align.Justify
	case "left":
		return align.Left
	default:
		return def
	}
}

// printColor парсит RRGGBB в maroto-цвет (пусто — nil = дефолтный).
func printColor(hex string) *props.Color {
	if hex == "" {
		return nil
	}
	rgb, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return nil
	}
	return &props.Color{Red: int(rgb >> 16), Green: int((rgb >> 8) & 0xFF), Blue: int(rgb & 0xFF)}
}

// printFontStyle собирает стиль gofpdf-строкой ("B", "BI", "BIU"...).
func printFontStyle(s printTextStyle, defBold bool) fontstyle.Type {
	bold := defBold || s.Bold
	style := ""
	if bold {
		style += "B"
	}
	if s.Italic {
		style += "I"
	}
	if s.Underline {
		style += "U"
	}
	return fontstyle.Type(style)
}

// printTextProps — общие текстовые пропсы блока (дефолты — per-type).
// Top/Bottom отступы задаёт вызывающий через printPad*.
func printTextProps(s printTextStyle, family string, defSize float64, defBold bool, defAlign align.Type) props.Text {
	size := defSize
	if s.Size != 0 {
		size = s.Size
	}
	return props.Text{
		Family: printPdfFontName(family),
		Size:   size,
		Style:  printFontStyle(s, defBold),
		Align:  printAlign(s.Align, defAlign),
		Color:  printColor(s.Color),
		// Абзацный отступ в PDF приближаем левым (честно: maroto не умеет
		// first-line; точный — только в DOCX).
		Left: s.Indent + s.FirstLine,
	}
}

// printContentWidth — ширина текстовой зоны A4 (мм) с учётом полей шаблона.
func printContentWidth(tpl *printTemplate) float64 {
	left, right := 10.0, 10.0
	if tpl.Margins != nil {
		if tpl.Margins.Left != 0 {
			left = tpl.Margins.Left
		}
		if tpl.Margins.Right != 0 {
			right = tpl.Margins.Right
		}
	}
	if w := 210.0 - left - right; w > 50 {
		return w
	}
	return 190.0
}

func newPrintMaroto(tpl *printTemplate) core.Maroto {
	builder := config.NewBuilder()
	builder.WithCustomFonts(printFontsOnce())
	builder.WithDefaultFont(&props.Font{
		Family: "Arial",
		Style:  fontstyle.Normal,
		Size:   10,
	})
	if tpl.Margins != nil {
		if tpl.Margins.Left != 0 {
			builder.WithLeftMargin(tpl.Margins.Left)
		}
		if tpl.Margins.Top != 0 {
			builder.WithTopMargin(tpl.Margins.Top)
		}
		if tpl.Margins.Right != 0 {
			builder.WithRightMargin(tpl.Margins.Right)
		}
		if tpl.Margins.Bottom != 0 {
			builder.WithBottomMargin(tpl.Margins.Bottom)
		}
	}
	builder.WithPageNumber()
	return maroto.New(builder.Build())
}

// renderPrintPDF строит PDF байты из шаблона и строки документа.
// images: src → байты (уже проверенные loadPrintImage).
// renderPrintPDF строит PDF байты из шаблона и строки документа.
// images: src → байты (уже проверенные loadPrintImage).
// Строки — авторяды (maroto сам считает высоту с переносами), поэтому
// длинный текст больше не наслаивается; интервалы — через Top/Bottom.
func renderPrintPDF(tpl *printTemplate, record map[string]any, images map[string][]byte) ([]byte, error) {
	m := newPrintMaroto(tpl)
	contentW := printContentWidth(tpl)

	if tpl.Header != "" {
		m.AddRows(printAutoRow(text.New(renderPrintText(tpl.Header, record), props.Text{
			Family: printPdfFamily(tpl),
			Size:   9,
			Align:  align.Center,
			Color:  &props.Color{Red: 128, Green: 128, Blue: 128},
		})))
	}

	if strings.TrimSpace(renderPrintText(tpl.Title, record)) != "" {
		m.AddRows(printAutoRow(text.New(renderPrintText(tpl.Title, record), props.Text{
			Family: printPdfFamily(tpl),
			Top:    5,
			Bottom: 8,
			Size:   16,
			Style:  fontstyle.Bold,
			Align:  align.Center,
		})))
	}

	for _, b := range tpl.Blocks {
		fam := printFontFamily(tpl, b.printTextStyle)
		switch b.Type {
		case "heading":
			// Жирность у heading НЕ дефолтная (в отличие от title): пример
			// требует обычный "О приёме лица в состав" рядом с жирным ПРИКАЗом.
			size := 13.0
			if b.Size != 0 {
				size = b.Size
			}
			m.AddRows(printParaRow(renderPrintText(b.Text, record),
				printTextProps(b.printTextStyle, fam, size, false, align.Left),
				b.printTextStyle.RightIndent, contentW, b.SpaceBefore, orDefault(b.SpaceAfter, 4))...)
		case "field":
			// Пустой field-блок пропускается (приказы не печатают "Примечание: ").
			if value := stringifyPrintValue(record[b.Field]); value != "" {
				m.AddRows(printParaRow(b.Label+": "+value,
					printTextProps(b.printTextStyle, fam, 10, false, align.Left),
					b.printTextStyle.RightIndent, contentW, b.SpaceBefore, orDefault(b.SpaceAfter, 4))...)
			}
		case "text":
			m.AddRows(printParaRow(renderPrintText(b.Text, record),
				printTextProps(b.printTextStyle, fam, 10, false, align.Left),
				b.printTextStyle.RightIndent, contentW, b.SpaceBefore, orDefault(b.SpaceAfter, 4))...)
		case "list":
			n := 0
			for _, item := range b.Items {
				rendered := renderPrintText(item, record)
				if rendered == "" {
					continue
				}
				n++
				prefix := "\u2022 "
				if b.Ordered {
					prefix = fmt.Sprintf("%d. ", n)
				}
				m.AddRows(printParaRow(prefix+rendered,
					printTextProps(b.printTextStyle, fam, 10, false, align.Left),
					b.printTextStyle.RightIndent, contentW, b.SpaceBefore, orDefault(b.SpaceAfter, 4))...)
			}
		case "image":
			img, ok := images[b.Src]
			if !ok {
				return nil, fmt.Errorf("print template: image %q not loaded", b.Src)
			}
			ext := "png"
			if lower := strings.ToLower(b.Src); strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") {
				ext = "jpg"
			}
			if b.SpaceBefore != 0 {
				m.AddRow(b.SpaceBefore, col.New(12))
			}
			m.AddRow(printImageHeight(img, b.Width)+4, col.New(12).Add(
				mimage.NewFromBytes(img, printImageExt(ext), props.Rect{
					Percent:            b.Width / contentW * 100,
					Center:             b.Align == "" || b.Align == "center",
					JustReferenceWidth: true,
				}),
			))
			m.AddRow(orDefault(b.SpaceAfter, 4), col.New(12))
		case "table":
			if b.SpaceBefore != 0 {
				m.AddRow(b.SpaceBefore, col.New(12))
			}
			size := 9.0
			if b.Size != 0 {
				size = b.Size
			}
			if b.Title != "" {
				m.AddRows(printAutoRow(text.New(renderPrintText(b.Title, record), props.Text{Family: printPdfFontName(fam), Size: size + 2, Style: fontstyle.Bold})))
			}
			if len(b.Columns) > 0 {
				widths := printColumnWidths(b.Columns)
				if printBlockShow(b.Header) {
					header := row.New()
					for i, c := range b.Columns {
						header.Add(col.New(widths[i]).Add(
							text.New(c.Label, printCellProps(c, fam, size, true)),
						))
					}
					m.AddRows(header)
				}
				data := row.New()
				for i, c := range b.Columns {
					data.Add(col.New(widths[i]).Add(
						text.New(printColumnValue(c, record), printCellProps(c, fam, size, false)),
					))
				}
				m.AddRows(data)
			} else {
				for _, r := range b.Rows {
					value := stringifyPrintValue(record[r.Field])
					labelRow := row.New()
					labelRow.Add(col.New(4).Add(
						text.New(r.Label, props.Text{Family: printPdfFontName(fam), Size: size, Style: fontstyle.Bold}),
					))
					labelRow.Add(col.New(8).Add(
						text.New(value, props.Text{Family: printPdfFontName(fam), Size: size}),
					))
					m.AddRows(labelRow)
				}
			}
			m.AddRow(orDefault(b.SpaceAfter, 4), col.New(12))
		}
	}

	if tpl.Footer != "" {
		m.AddRows(printAutoRow(text.New(renderPrintText(tpl.Footer, record), props.Text{
			Family: printPdfFamily(tpl),
			Size:   9,
			Align:  align.Center,
			Color:  &props.Color{Red: 128, Green: 128, Blue: 128},
		})))
	}

	document, err := m.Generate()
	if err != nil {
		return nil, err
	}
	return document.GetBytes(), nil
}

// orDefault возвращает v, если задано, иначе def.
func orDefault(v, def float64) float64 {
	if v != 0 {
		return v
	}
	return def
}

// printAutoRow — строка с высотой по содержимому (переносы не наслаиваются).
func printAutoRow(c core.Component) core.Row {
	return row.New().Add(col.New(12).Add(c))
}

// printParaRow — абзац с Top/Bottom интервалами и правым отступом.
// RightIndent в PDF приближён сужением колонки (точный — только в DOCX).
// \t раскрывается в пробелы (~12.7 мм, как таб-стоп Word по умолчанию;
// точные таб-стопы — только в DOCX). \n режется на отдельные строки:
// maroto считает высоту только по пробелам, необработанный \n дал бы
// наслоение строк друг на друга.
func printParaRow(value string, p props.Text, rightIndent, contentW, top, bottom float64) []core.Row {
	lines := strings.Split(expandPrintTabs(value, p.Size), "\n")
	rows := make([]core.Row, 0, len(lines))
	for i, line := range lines {
		tp := p
		if i == 0 {
			tp.Top += top
		}
		if i == len(lines)-1 {
			tp.Bottom += bottom
		}
		t := text.New(line, tp)
		tw := 12
		if rightIndent > 0 && contentW > rightIndent {
			tw = int(12 * (contentW - rightIndent) / contentW)
			if tw < 1 {
				tw = 1
			}
			if tw > 11 {
				tw = 11
			}
		}
		r := row.New()
		r.Add(col.New(tw).Add(t))
		if tw < 12 {
			r.Add(col.New(12 - tw))
		}
		rows = append(rows, r)
	}
	return rows
}

// expandPrintTabs раскрывает табы в пробелы (~12.7 мм, таб-стоп Word
// по умолчанию). Точные таб-стопы — только в DOCX.
func expandPrintTabs(value string, sizePt float64) string {
	if !strings.Contains(value, "\t") {
		return value
	}
	if sizePt <= 0 {
		sizePt = 11
	}
	n := int(12.7/(0.25*sizePt*25.4/72) + 0.5)
	if n < 1 {
		n = 1
	}
	return strings.ReplaceAll(value, "\t", strings.Repeat(" ", n))
}

// printCellProps — пропсы ячейки таблицы (стиль колонки поверх блочного).
func printCellProps(c printBlockColumn, fam string, size float64, bold bool) props.Text {
	if c.Size != 0 {
		size = c.Size
	}
	if c.Font != "" {
		fam = c.Font
	}
	return props.Text{
		Family: printPdfFontName(fam),
		Size:   size,
		Style:  printCellStyle(c, bold),
		Align:  printAlign(c.Align, align.Left),
		Color:  printColor(c.Color),
	}
}

// printCellStyle собирает стиль ячейки (жирность шапки + своё).
func printCellStyle(c printBlockColumn, bold bool) fontstyle.Type {
	style := ""
	if bold || c.Bold {
		style += "B"
	}
	if c.Italic {
		style += "I"
	}
	if c.Underline {
		style += "U"
	}
	return fontstyle.Type(style)
}

// printPdfFamily — семейство по умолчанию шаблона (блоки резолвят своё сами).
func printPdfFamily(tpl *printTemplate) string {
	return printPdfFontName(printFontFamily(tpl, printTextStyle{}))
}

// printPdfFontName маппит DSL-имя в имя семейства maroto.
func printPdfFontName(fam string) string {
	if fam == "times" {
		return printFontTimes
	}
	return "Arial"
}

// printColumnWidths делит сетку 12 по весам колонок (дефолт 1).
func printColumnWidths(cols []printBlockColumn) []int {
	return printGridWidths(cols, 12)
}

// printTwipsWidths делит 9000 twips по весам колонок (дефолт 1).
func printTwipsWidths(cols []printBlockColumn) []int {
	return printGridWidths(cols, 9000)
}

// printGridWidths делит total единиц по весам колонок (дефолт 1).
func printGridWidths(cols []printBlockColumn, total int) []int {
	sum := 0.0
	for _, c := range cols {
		w := c.Width
		if w == 0 {
			w = 1
		}
		sum += w
	}
	widths := make([]int, len(cols))
	assigned := 0
	for i, c := range cols {
		w := c.Width
		if w == 0 {
			w = 1
		}
		widths[i] = int(w / sum * float64(total))
		if widths[i] < 1 {
			widths[i] = 1
		}
		assigned += widths[i]
	}
	// Раскладываем остаток/перебор на первую колонку (сетка обязана сойтись).
	if len(widths) > 0 {
		widths[0] += total - assigned
		if widths[0] < 1 {
			widths[0] = 1
		}
	}
	return widths
}

// printImageHeight оценивает высоту картинки в мм по аспекту (шапка строки).
func printImageHeight(img []byte, widthMm float64) float64 {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(img))
	if err != nil || cfg.Width == 0 {
		return widthMm
	}
	return widthMm * float64(cfg.Height) / float64(cfg.Width)
}

func printImageExt(ext string) extension.Type {
	if ext == "jpg" || ext == "jpeg" {
		return extension.Jpg
	}
	return extension.Png
}
