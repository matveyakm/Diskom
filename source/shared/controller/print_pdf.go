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
func printTextProps(s printTextStyle, family string, defSize float64, defBold bool, defAlign align.Type) props.Text {
	size := defSize
	if s.Size != 0 {
		size = s.Size
	}
	fam := printPdfFontName(family)
	return props.Text{
		Family: fam,
		Size:   size,
		Style:  printFontStyle(s, defBold),
		Align:  printAlign(s.Align, defAlign),
		Color:  printColor(s.Color),
		// Абзацный отступ в PDF приближаем левым (честно: maroto не умеет
		// first-line; точный — только в DOCX).
		Left: s.Indent + s.FirstLine,
	}
}

func newPrintMaroto() core.Maroto {
	builder := config.NewBuilder()
	builder.WithCustomFonts(printFontsOnce())
	builder.WithDefaultFont(&props.Font{
		Family: "Arial",
		Style:  fontstyle.Normal,
		Size:   10,
	})
	builder.WithPageNumber()
	return maroto.New(builder.Build())
}

// renderPrintPDF строит PDF байты из шаблона и строки документа.
// images: src → байты (уже проверенные loadPrintImage).
func renderPrintPDF(tpl *printTemplate, record map[string]any, images map[string][]byte) ([]byte, error) {
	m := newPrintMaroto()

	if tpl.Header != "" {
		m.AddRow(8, col.New(12).Add(
			text.New(renderPrintText(tpl.Header, record), props.Text{
				Family: printPdfFamily(tpl),
				Size:   9,
				Align:  align.Center,
				Color:  &props.Color{Red: 128, Green: 128, Blue: 128},
			}),
		))
	}

	if strings.TrimSpace(renderPrintText(tpl.Title, record)) != "" {
		m.AddRow(20, col.New(12).Add(
			text.New(renderPrintText(tpl.Title, record), props.Text{
				Family: printPdfFamily(tpl),
				Top:    5,
				Size:   16,
				Style:  fontstyle.Bold,
				Align:  align.Center,
			}),
		))
		m.AddRow(8)
	}

	for _, b := range tpl.Blocks {
		switch b.Type {
		case "heading":
			size := 13.0
			if b.Size != 0 {
				size = b.Size
			}
			m.AddRow(size+3, col.New(12).Add(
				text.New(renderPrintText(b.Text, record), printTextProps(b.printTextStyle, printFontFamily(tpl, b.printTextStyle), size, true, align.Left)),
			))
			m.AddRow(4)
		case "field":
			// Пустой field-блок пропускается (уже проверено вызывателем
			// через stringifyPrintValue, дублируем защиту здесь).
			if value := stringifyPrintValue(record[b.Field]); value != "" {
				m.AddRow(8, col.New(12).Add(
					text.New(b.Label+": "+value, printTextProps(b.printTextStyle, printFontFamily(tpl, b.printTextStyle), 10, false, align.Left)),
				))
			}
		case "text":
			m.AddRow(8, col.New(12).Add(
				text.New(renderPrintText(b.Text, record), printTextProps(b.printTextStyle, printFontFamily(tpl, b.printTextStyle), 10, false, align.Left)),
			))
		case "list":
			for i, item := range b.Items {
				rendered := renderPrintText(item, record)
				if rendered == "" {
					continue
				}
				prefix := "• "
				if b.Ordered {
					prefix = fmt.Sprintf("%d. ", i+1)
				}
				m.AddRow(8, col.New(12).Add(
					text.New(prefix+rendered, printTextProps(b.printTextStyle, printFontFamily(tpl, b.printTextStyle), 10, false, align.Left)),
				))
			}
			m.AddRow(2)
		case "image":
			img, ok := images[b.Src]
			if !ok {
				return nil, fmt.Errorf("print template: image %q not loaded", b.Src)
			}
			ext := "png"
			if lower := strings.ToLower(b.Src); strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") {
				ext = "jpg"
			}
			m.AddRow(printImageHeight(img, b.Width)+4, col.New(12).Add(
				mimage.NewFromBytes(img, printImageExt(ext), props.Rect{
					Percent: b.Width / 190 * 100,
					Center:  b.Align == "" || b.Align == "center",
				}),
			))
			m.AddRow(4)
		case "table":
			fam := printFontFamily(tpl, b.printTextStyle)
			if b.Title != "" {
				m.AddRow(10, col.New(12).Add(
					text.New(renderPrintText(b.Title, record), props.Text{Family: printPdfFontName(fam), Size: 11, Style: fontstyle.Bold}),
				))
			}
			size := 9.0
			if b.Size != 0 {
				size = b.Size
			}
			if len(b.Columns) > 0 {
				widths := printColumnWidths(b.Columns)
				header := row.New(10)
				for i, c := range b.Columns {
					header.Add(col.New(widths[i]).Add(
						text.New(c.Label, props.Text{
							Family: printPdfFontName(fam),
							Size:   size,
							Style:  fontstyle.Bold,
							Align:  printAlign(c.Align, align.Left),
						}),
					))
				}
				m.AddRows(header)
				data := row.New(10)
				for i, c := range b.Columns {
					data.Add(col.New(widths[i]).Add(
						text.New(stringifyPrintValue(record[c.Field]), props.Text{
							Family: printPdfFontName(fam),
							Size:   size,
							Align:  printAlign(c.Align, align.Left),
						}),
					))
				}
				m.AddRows(data)
			} else {
				for _, r := range b.Rows {
					value := stringifyPrintValue(record[r.Field])
					labelRow := row.New(8)
					labelRow.Add(col.New(4).Add(
						text.New(r.Label, props.Text{Family: printPdfFontName(fam), Size: size, Style: fontstyle.Bold}),
					))
					labelRow.Add(col.New(8).Add(
						text.New(value, props.Text{Family: printPdfFontName(fam), Size: size}),
					))
					m.AddRows(labelRow)
				}
			}
			m.AddRow(4)
		}
	}

	if tpl.Footer != "" {
		m.AddRow(8, col.New(12).Add(
			text.New(renderPrintText(tpl.Footer, record), props.Text{
				Family: printPdfFamily(tpl),
				Size:   9,
				Align:  align.Center,
				Color:  &props.Color{Red: 128, Green: 128, Blue: 128},
			}),
		))
	}

	document, err := m.Generate()
	if err != nil {
		return nil, err
	}
	return document.GetBytes(), nil
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
	total := 0.0
	for _, c := range cols {
		w := c.Width
		if w == 0 {
			w = 1
		}
		total += w
	}
	widths := make([]int, len(cols))
	assigned := 0
	for i, c := range cols {
		w := c.Width
		if w == 0 {
			w = 1
		}
		widths[i] = int(w / total * 12)
		if widths[i] < 1 {
			widths[i] = 1
		}
		assigned += widths[i]
	}
	// Раскладываем остаток/перебор на первую колонку (сетка обязана дать 12).
	widths[0] += 12 - assigned
	if widths[0] < 1 {
		widths[0] = 1
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
