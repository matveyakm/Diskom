package restController

// PDF-рендер печатной формы на folio (plan-based layout, Apache 2.0).
//
// Maroto здесь был до v0.12: grid «строки-колонки» без доступа к низкому
// уровню — оверлей подписи невозможен конструкцией, justify съедал ведущие
// пробелы, line-height и first-line только имитировались. Folio даёт прямой
// доступ: абсолютное позиционирование (Div.AddOverlay — тот же якорь, что
// wp:anchor в DOCX), настоящий firstLine-индент, множитель интерлиньяжа.
// DSL-контракт шаблона при этом не меняется: те же блоки, стили, x/y.

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"sync"

	"github.com/carlos7ags/folio/document"
	"github.com/carlos7ags/folio/font"
	folioimage "github.com/carlos7ags/folio/image"
	"github.com/carlos7ags/folio/layout"
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

// printFolioFonts — распарсенные TTF (парсинг один раз на процесс,
// EmbeddedFont при этом новый... нет: EmbeddedFont переиспользуем —
// сабсеттер документа собирает глифы сам, состояние на документ не течёт).
type printFolioFonts struct {
	arial           *font.EmbeddedFont
	arialBold       *font.EmbeddedFont
	times           *font.EmbeddedFont
	timesBold       *font.EmbeddedFont
	timesItalic     *font.EmbeddedFont
	timesBoldItalic *font.EmbeddedFont
}

// Шрифты грузятся один раз на процесс (парсинг TTF per-request расточителен,
// документ при этом всё равно новый на каждый вызов).
var printFolioFontsOnce = sync.OnceValue(func() *printFolioFonts {
	load := func(data []byte, name string) *font.EmbeddedFont {
		face, err := font.ParseFont(data)
		if err != nil {
			panic("print: parse font " + name + ": " + err.Error())
		}
		return font.NewEmbeddedFont(face)
	}
	return &printFolioFonts{
		arial:           load(printArialRegular, "arial"),
		arialBold:       load(printArialBold, "arial-bold"),
		times:           load(printTinosRegular, "times"),
		timesBold:       load(printTinosBold, "times-bold"),
		timesItalic:     load(printTinosItalic, "times-italic"),
		timesBoldItalic: load(printTinosBoldItalic, "times-bolditalic"),
	}
})

// printFolioFace выбирает начертание: times — все четыре, arial — только
// regular/bold (italic без начертания запрещён валидацией шаблона).
func (f *printFolioFonts) face(family string, bold, italic bool) *font.EmbeddedFont {
	if family == "times" {
		switch {
		case bold && italic:
			return f.timesBoldItalic
		case bold:
			return f.timesBold
		case italic:
			return f.timesItalic
		default:
			return f.times
		}
	}
	if bold {
		return f.arialBold
	}
	return f.arial
}

// printMmToPt переводит мм DSL в пункты folio.
func printMmToPt(mm float64) float64 {
	return mm * 72 / 25.4
}

// printFolioAlign маппит DSL-выравнивание (дефолт — переданный).
func printFolioAlign(a string, def layout.Align) layout.Align {
	switch a {
	case "center":
		return layout.AlignCenter
	case "right":
		return layout.AlignRight
	case "justify":
		return layout.AlignJustify
	case "left":
		return layout.AlignLeft
	default:
		return def
	}
}

// printTabIndentMm — ширина одного ведущего таба в мм (таб-стоп Word
// по умолчанию; тот же масштаб, что в expandPrintTabs и DOCX <w:tab/>).
const printTabIndentMm = 12.7

// splitLeadingTabs отщепляет ведущие табы строки: их ширина уезжает
// в абзацный отступ, а не в пробелы (пробелы в начале строки при justify
// схлопываются; SetFirstLineIndent переживает любое выравнивание).
// Таб в середине строки — пробелами, как раньше (см. expandPrintTabs).
func splitLeadingTabs(s string) (int, string) {
	n := 0
	for strings.HasPrefix(s, "\t") {
		s = s[1:]
		n++
	}
	return n, s
}

// printFolioRun собирает текстовый ран блока: начертание по bold/italic,
// подчёркивание и цвет поверх.
func printFolioRun(fonts *printFolioFonts, tpl *printTemplate, s printTextStyle, text string, size float64, defBold bool) layout.TextRun {
	ef := fonts.face(printFontFamily(tpl, s), defBold || s.Bold, s.Italic)
	run := layout.NewRunEmbedded(text, ef, size)
	if s.Underline {
		run = run.WithUnderline()
	}
	if s.Color != "" {
		run = run.WithColor(layout.Hex(s.Color))
	}
	return run
}

// printFolioPara — абзац из текстового блока: \n режется на отдельные
// абзацы (явный разрыв строки, как DOCX <w:br/>); ведущие табы каждой
// строки — в firstLine-индент (~12.7 мм каждый); indent/rightIndent —
// паддингами контейнера (точные, в отличие от старого движка);
// spaceBefore/spaceAfter — в пунктах (дефолт after 2.1 мм, как в DOCX).
func printFolioPara(fonts *printFolioFonts, tpl *printTemplate, s printTextStyle, text string, defSize float64, defBold bool, defAlign layout.Align) []layout.Element {
	size := defSize
	if s.Size != 0 {
		size = s.Size
	}
	var out []layout.Element
	for _, line := range strings.Split(text, "\n") {
		firstLine := s.FirstLine
		if n, rest := splitLeadingTabs(line); n > 0 {
			line = rest
			firstLine += float64(n) * printTabIndentMm
		}
		line = expandPrintTabs(line, size)
		p := layout.NewStyledParagraph(printFolioRun(fonts, tpl, s, line, size, defBold))
		p.SetAlign(printFolioAlign(s.Align, defAlign))
		if firstLine != 0 {
			p.SetFirstLineIndent(printMmToPt(firstLine))
		}
		if s.Line != 0 {
			p.SetLeading(s.Line)
		}
		if s.SpaceBefore != 0 {
			p.SetSpaceBefore(printMmToPt(s.SpaceBefore))
		}
		after := 2.1
		if s.SpaceAfter != 0 {
			after = s.SpaceAfter
		}
		p.SetSpaceAfter(printMmToPt(after))
		if s.Indent != 0 || s.RightIndent != 0 {
			div := layout.NewDiv()
			div.SetPaddingAll(layout.Padding{Left: printMmToPt(s.Indent), Right: printMmToPt(s.RightIndent)})
			div.Add(p)
			out = append(out, div)
		} else {
			out = append(out, p)
		}
	}
	return out
}

// printTwipsWidths делит total twips по весам колонок (дефолт 1, DOCX;
// total — ширина текстовой зоны: при fixed-раскладке колонки строго по grid).
func printTwipsWidths(cols []printBlockColumn, total int) []int {
	return printGridWidths(cols, total)
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

// printFolioColWidths делит ширину зоны по весам колонок (дефолт 1).
func printFolioColWidths(cols []printBlockColumn, totalPt float64) []float64 {
	sum := 0.0
	for _, c := range cols {
		w := c.Width
		if w == 0 {
			w = 1
		}
		sum += w
	}
	widths := make([]float64, len(cols))
	for i, c := range cols {
		w := c.Width
		if w == 0 {
			w = 1
		}
		widths[i] = w / sum * totalPt
	}
	return widths
}

// printFolioImage декодирует картинку блока (формат — по magic, как в
// валидации загрузки; содержимое ещё раз проверяется внутри folio).
func printFolioImage(raw []byte) (*folioimage.Image, error) {
	if mime, ok := fileMimeByMagic(raw); ok && mime == "image/jpeg" {
		return folioimage.NewJPEG(raw)
	}
	return folioimage.NewPNG(raw)
}

// renderPrintPDF строит PDF байты из шаблона и строки документа (folio).
func renderPrintPDF(tpl *printTemplate, record map[string]any, images map[string][]byte) ([]byte, error) {
	fonts := printFolioFontsOnce()
	doc := document.NewDocument(document.PageSizeA4)
	left, top, right, bottom := 10.0, 10.0, 10.0, 20.0
	if tpl.Margins != nil {
		if tpl.Margins.Left != 0 {
			left = tpl.Margins.Left
		}
		if tpl.Margins.Top != 0 {
			top = tpl.Margins.Top
		}
		if tpl.Margins.Right != 0 {
			right = tpl.Margins.Right
		}
		if tpl.Margins.Bottom != 0 {
			bottom = tpl.Margins.Bottom
		}
	}
	doc.SetMargins(layout.Margins{
		Left: printMmToPt(left), Top: printMmToPt(top),
		Right: printMmToPt(right), Bottom: printMmToPt(bottom),
	})
	contentW := 210.0 - left - right

	gray := layout.Gray(0.5)
	headerText := strings.TrimSpace(renderPrintText(tpl.Header, record))
	if headerText != "" || tpl.printPageNum() == "header" {
		text := headerText
		if tpl.printPageNum() == "header" {
			text += " {page}"
		}
		doc.SetHeaderElement(func(ctx document.PageContext) layout.Element {
			s := text
			if tpl.printPageNum() == "header" {
				s = strings.ReplaceAll(text, "{page}", fmt.Sprintf("%d", ctx.PageIndex+1))
			}
			p := layout.NewStyledParagraph(
				layout.NewRunEmbedded(s, fonts.face(printFontFamily(tpl, printTextStyle{}), false, false), 9).
					WithColor(gray))
			p.SetAlign(layout.AlignCenter)
			return p
		})
	}
	footerText := strings.TrimSpace(renderPrintText(tpl.Footer, record))
	if footerText != "" || tpl.printPageNum() == "footer" {
		text := footerText
		if tpl.printPageNum() == "footer" {
			if text != "" {
				text += " "
			}
			text += "Стр. {page}"
		}
		doc.SetFooterElement(func(ctx document.PageContext) layout.Element {
			s := text
			if tpl.printPageNum() == "footer" {
				s = strings.ReplaceAll(text, "{page}", fmt.Sprintf("%d", ctx.PageIndex+1))
			}
			p := layout.NewStyledParagraph(
				layout.NewRunEmbedded(s, fonts.face(printFontFamily(tpl, printTextStyle{}), false, false), 9).
					WithColor(gray))
			p.SetAlign(layout.AlignCenter)
			return p
		})
	}

	if strings.TrimSpace(renderPrintText(tpl.Title, record)) != "" {
		p := layout.NewStyledParagraph(printFolioRun(fonts, tpl,
			printTextStyle{Bold: true, Align: "center"}, renderPrintText(tpl.Title, record), 16, true))
		p.SetAlign(layout.AlignCenter)
		p.SetSpaceBefore(printMmToPt(5))
		p.SetSpaceAfter(printMmToPt(8))
		doc.Add(p)
	}

	for _, b := range tpl.Blocks {
		switch b.Type {
		case "heading":
			// Жирность у heading НЕ дефолтная (пример требует обычный
			// "О приёме лица в состав" рядом с жирным ПРИКАЗом).
			size := 13.0
			if b.Size != 0 {
				size = b.Size
			}
			for _, e := range printFolioPara(fonts, tpl, b.printTextStyle,
				renderPrintText(b.Text, record), size, false, layout.AlignLeft) {
				doc.Add(e)
			}
		case "field":
			// Пустой field-блок пропускается (приказы не печатают "Примечание: ").
			if value := stringifyPrintValue(record[b.Field]); value != "" {
				for _, e := range printFolioPara(fonts, tpl, b.printTextStyle,
					b.Label+": "+value, 10, false, layout.AlignLeft) {
					doc.Add(e)
				}
			}
		case "text":
			for _, e := range printFolioPara(fonts, tpl, b.printTextStyle,
				renderPrintText(b.Text, record), 10, false, layout.AlignLeft) {
				doc.Add(e)
			}
		case "list":
			size := 10.0
			if b.Size != 0 {
				size = b.Size
			}
			ef := fonts.face(printFontFamily(tpl, b.printTextStyle), b.Bold, b.Italic)
			lst := layout.NewListEmbedded(ef, size)
			if b.Ordered {
				lst.SetStyle(layout.ListOrdered)
			}
			n := 0
			for _, item := range b.Items {
				rendered := renderPrintText(item, record)
				if rendered == "" {
					continue
				}
				n++
				lst.AddItem(rendered)
			}
			if n == 0 {
				continue
			}
			if b.Line != 0 {
				lst.SetLeading(b.Line)
			}
			after := 2.1
			if b.SpaceAfter != 0 {
				after = b.SpaceAfter
			}
			if b.Indent != 0 || b.RightIndent != 0 || b.SpaceBefore != 0 || b.SpaceAfter != 0 {
				div := layout.NewDiv()
				div.SetPaddingAll(layout.Padding{Left: printMmToPt(b.Indent), Right: printMmToPt(b.RightIndent)})
				div.SetSpaceBefore(printMmToPt(b.SpaceBefore))
				div.SetSpaceAfter(printMmToPt(after))
				div.Add(lst)
				doc.Add(div)
			} else {
				doc.Add(lst)
			}
		case "image":
			img, ok := images[b.imageKey()]
			if !ok {
				// Поле-логотип пустое — пропускаем молча, как пустой
				// field-блок. Битая src — 500 ещё на загрузке.
				if b.Src != "" {
					return nil, fmt.Errorf("print template: image %q not loaded", b.Src)
				}
				continue
			}
			fimg, err := printFolioImage(img)
			if err != nil {
				return nil, fmt.Errorf("print template: image %q: %w", b.imageKey(), err)
			}
			hMm := printImageHeight(img, b.Width)
			elem := layout.NewImageElement(fimg)
			elem.SetSize(printMmToPt(b.Width), printMmToPt(hMm))
			if b.Float {
				// Подпись поверх текста: якорь-оверлей относительно
				// позиции блока в потоке (x — от края колонки, y — вниз
				// от верха абзаца-якоря), как wp:anchor в DOCX. Пустой
				// div-контейнер места в потоке не занимает.
				div := layout.NewDiv()
				if b.SpaceBefore != 0 {
					div.SetSpaceBefore(printMmToPt(b.SpaceBefore))
				}
				if b.SpaceAfter != 0 {
					div.SetSpaceAfter(printMmToPt(b.SpaceAfter))
				}
				div.AddOverlay(elem, printMmToPt(b.X), printMmToPt(b.Y), printMmToPt(b.Width), false, 1)
				doc.Add(div)
			} else {
				elem.SetAlign(printFolioAlign(b.Align, layout.AlignCenter))
				if b.SpaceBefore != 0 || b.SpaceAfter != 0 {
					after := 2.1
					if b.SpaceAfter != 0 {
						after = b.SpaceAfter
					}
					div := layout.NewDiv()
					div.SetSpaceBefore(printMmToPt(b.SpaceBefore))
					div.SetSpaceAfter(printMmToPt(after))
					div.Add(elem)
					doc.Add(div)
				} else {
					doc.Add(elem)
				}
			}
		case "table":
			size := 9.0
			if b.Size != 0 {
				size = b.Size
			}
			if b.Title != "" {
				tsize := size + 2
				for _, e := range printFolioPara(fonts, tpl,
					printTextStyle{Bold: true}, renderPrintText(b.Title, record), tsize, true, layout.AlignLeft) {
					doc.Add(e)
				}
			}
			if len(b.Columns) > 0 {
				tbl := layout.NewTable()
				tbl.SetColumnWidths(printFolioColWidths(b.Columns, printMmToPt(contentW)))
				noborder := !printBlockShow(b.Borders)
				if printBlockShow(b.Header) {
					hr := tbl.AddHeaderRow()
					for _, c := range b.Columns {
						style := printColumnStyle(tpl, b.printTextStyle, c)
						csize := size
						if style.Size != 0 {
							csize = style.Size
						}
						cp := layout.NewStyledParagraph(printFolioRun(fonts, tpl, style, c.Label, csize, true))
						cp.SetAlign(printFolioAlign(style.Align, layout.AlignLeft))
						cell := hr.AddCellElement(cp)
						applyFolioCellStyle(cell, noborder)
					}
				}
				dr := tbl.AddRow()
				for _, c := range b.Columns {
					style := printColumnStyle(tpl, b.printTextStyle, c)
					csize := size
					if style.Size != 0 {
						csize = style.Size
					}
					ef := fonts.face(printFontFamily(tpl, style), style.Bold, style.Italic)
					run := layout.NewRunEmbedded(printColumnValue(c, record), ef, csize)
					if style.Underline {
						run = run.WithUnderline()
					}
					if style.Color != "" {
						run = run.WithColor(layout.Hex(style.Color))
					}
					cp := layout.NewStyledParagraph(run)
					cp.SetAlign(printFolioAlign(style.Align, layout.AlignLeft))
					cell := dr.AddCellElement(cp)
					applyFolioCellStyle(cell, noborder)
				}
				doc.Add(tbl)
			} else {
				tbl := layout.NewTable()
				tbl.SetColumnWidths([]float64{printMmToPt(contentW) * 4 / 12, printMmToPt(contentW) * 8 / 12})
				noborder := !printBlockShow(b.Borders)
				for _, r := range b.Rows {
					dr := tbl.AddRow()
					lc := dr.AddCellEmbedded(r.Label,
						fonts.face(printFontFamily(tpl, b.printTextStyle), true, false), size)
					applyFolioCellStyle(lc, noborder)
					vc := dr.AddCellEmbedded(stringifyPrintValue(record[r.Field]),
						fonts.face(printFontFamily(tpl, b.printTextStyle), false, false), size)
					applyFolioCellStyle(vc, noborder)
				}
				doc.Add(tbl)
			}
		}
	}

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// applyFolioCellStyle вешает границы ячейки (или их отсутствие).
func applyFolioCellStyle(cell *layout.Cell, noborder bool) {
	if noborder {
		cell.SetBorders(layout.NoBorders())
		return
	}
	cell.SetBorders(layout.AllBorders(layout.SolidBorder(0.5, layout.Gray(0))))
}

// expandPrintTabs раскрывает табы в пробелы (~12.7 мм, таб-стоп Word
// по умолчанию). Ведущие табы строк обрабатываются отдельно
// (splitLeadingTabs → firstLine-индент); точные таб-стопы — только в DOCX.
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

// printImageHeight оценивает высоту картинки в мм по аспекту.
func printImageHeight(img []byte, widthMm float64) float64 {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(img))
	if err != nil || cfg.Width == 0 {
		return widthMm
	}
	return widthMm * float64(cfg.Height) / float64(cfg.Width)
}
