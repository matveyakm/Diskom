package restController

// DOCX-рендер печатной формы на stdlib (archive/zip + ручной XML).
//
// Новых зависимостей ноль осознанно: документ простой, а тащить unioffice
// ради него — лишний вес и вопросы лицензирования. Минимальный набор частей,
// который открывают Word и LibreOffice, плюс media/rels для картинок и
// header/footer с нумерацией страниц (fldSimple PAGE).

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"
)

const (
	printDocxMime = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	printPdfMime  = "application/pdf"
)

// xmlEscape экранирует текст для w:t (иначе значение с & или < ломает
// весь пакет — молча битый файл вместо ошибки).
func xmlEscape(s string) string {
	r := strings.NewReplacer(
		`&`, "&amp;",
		`<`, "&lt;",
		`>`, "&gt;",
		`"`, "&quot;",
	)
	return r.Replace(s)
}

// printDocxFont маппит DSL-имя в имя для Word (Tinos метрически
// совместим с Times New Roman, в документе фигурирует как он).
func printDocxFont(fam string) string {
	if fam == "times" {
		return "Times New Roman"
	}
	return "Arial"
}

// printDocxRun — текстовый ран с оформлением блока.
// \n внутри текста — разрыв строки (<w:br/>), \t — табуляция (<w:tab/>):
// так шаблоны выражают и жёсткие переносы, и ведущий таб абзаца.
func printDocxRun(text string, s printTextStyle, defSize float64, defBold bool, fam string) string {
	style := ""
	if defBold || s.Bold {
		style += `<w:b/>`
	}
	if s.Italic {
		style += `<w:i/>`
	}
	if s.Underline {
		style += `<w:u w:val="single"/>`
	}
	size := defSize
	if s.Size != 0 {
		size = s.Size
	}
	if s.Color != "" {
		style += `<w:color w:val="` + strings.ToUpper(s.Color) + `"/>`
	}
	rpr := fmt.Sprintf(`<w:rPr>%s<w:sz w:val="%d"/><w:rFonts w:ascii="%s" w:hAnsi="%s"/></w:rPr>`,
		style, int(size*2), fam, fam)
	var b strings.Builder
	for i, part := range strings.Split(text, "\n") {
		if i > 0 {
			b.WriteString(`<w:r>` + rpr + `<w:br/></w:r>`)
		}
		sub := strings.Split(part, "\t")
		for j, chunk := range sub {
			if j > 0 {
				b.WriteString(`<w:r>` + rpr + `<w:tab/></w:r>`)
			}
			if chunk != "" {
				b.WriteString(`<w:r>` + rpr + `<w:t xml:space="preserve">` + xmlEscape(chunk) + `</w:t></w:r>`)
			}
		}
	}
	return b.String()
}

// printDocxPara — абзац. center/right/justify через jc; отступы и интервалы
// в twips (1 мм = 56.7; line 240/360/480 = single/1.5/double).
func printDocxPara(text string, s printTextStyle, defSize float64, defBold bool, fam string) string {
	jc := ""
	switch s.Align {
	case "center":
		jc = `<w:jc w:val="center"/>`
	case "right":
		jc = `<w:jc w:val="right"/>`
	case "justify":
		jc = `<w:jc w:val="both"/>`
	}
	ind := ""
	if s.Indent != 0 || s.FirstLine != 0 || s.RightIndent != 0 {
		ind = fmt.Sprintf(`<w:ind w:left="%d" w:right="%d" w:firstLine="%d"/>`,
			int(s.Indent*56.7), int(s.RightIndent*56.7), int(s.FirstLine*56.7))
	}
	// Канонический порядок: before/after/line. Дефолт after — ровно 120 twips,
	// как было до параметризации (вычисление 2.1 мм дало бы 119).
	sp := `<w:spacing`
	if s.SpaceBefore != 0 {
		sp += fmt.Sprintf(` w:before="%d"`, int(s.SpaceBefore*56.7))
	}
	after := 120
	if s.SpaceAfter != 0 {
		after = int(s.SpaceAfter * 56.7)
	}
	sp += fmt.Sprintf(` w:after="%d"`, after)
	if s.Line == 1.5 {
		sp += ` w:line="360" w:lineRule="auto"`
	} else if s.Line == 2 {
		sp += ` w:line="480" w:lineRule="auto"`
	}
	sp += `/>`
	return fmt.Sprintf(
		`<w:p><w:pPr>%s%s%s</w:pPr>%s</w:p>`,
		jc, ind, sp, printDocxRun(text, s, defSize, defBold, fam),
	)
}

func printDocxCell(text string, s printTextStyle, defBold bool, size float64, align string, fam string) string {
	style := ""
	if defBold || s.Bold {
		style += `<w:b/>`
	}
	if s.Italic {
		style += `<w:i/>`
	}
	if s.Underline {
		style += `<w:u w:val="single"/>`
	}
	if s.Color != "" {
		style += `<w:color w:val="` + strings.ToUpper(s.Color) + `"/>`
	}
	if s.Size != 0 {
		size = s.Size
	}
	jc := ""
	switch align {
	case "center":
		jc = `<w:jc w:val="center"/>`
	case "right":
		jc = `<w:jc w:val="right"/>`
	case "justify":
		jc = `<w:jc w:val="both"/>`
	}
	// Пустой pPr не эмитим вообще (валидный, но лишний узел для строгих читалок).
	ppr := ""
	if jc != "" {
		ppr = `<w:pPr>` + jc + `</w:pPr>`
	}
	return fmt.Sprintf(
		`<w:tc><w:tcPr><w:tcW w:w="0" w:type="auto"/></w:tcPr><w:p>%s<w:r><w:rPr>%s<w:sz w:val="%d"/><w:rFonts w:ascii="%s" w:hAnsi="%s"/></w:rPr><w:t xml:space="preserve">%s</w:t></w:r></w:p></w:tc>`,
		ppr, style, int(size*2), fam, fam, xmlEscape(text),
	)
}

// printColumnStyle мержит стиль колонки поверх блочного (пустые поля наследуются).
func printColumnStyle(tpl *printTemplate, b printTextStyle, c printBlockColumn) printTextStyle {
	out := b
	if c.Font != "" {
		out.Font = c.Font
	}
	if c.Bold {
		out.Bold = true
	}
	if c.Italic {
		out.Italic = true
	}
	if c.Underline {
		out.Underline = true
	}
	if c.Color != "" {
		out.Color = c.Color
	}
	if c.Size != 0 {
		out.Size = c.Size
	}
	if c.Align != "" {
		out.Align = c.Align
	}
	_ = tpl
	return out
}

// printDocxContentTwips — ширина текстовой зоны в twips (A4 11906 минус поля;
// 0 в шаблоне = дефолт движка 1440, как в pgMar ниже).
func printDocxContentTwips(tpl *printTemplate) int {
	left, right := 1440, 1440
	if tpl.Margins != nil {
		if tpl.Margins.Left != 0 {
			left = int(tpl.Margins.Left * 56.7)
		}
		if tpl.Margins.Right != 0 {
			right = int(tpl.Margins.Right * 56.7)
		}
	}
	return 11906 - left - right
}

func printDocxTable(headers []string, values []string, styles []printTextStyle, widths []int, header, borders bool, fam string) string {
	var b strings.Builder
	// Фиксированная раскладка: колонки строго по gridCol. При auto (дефолт
	// Word) ширины игнорируются и колонки сжимаются по контенту — ячейки
	// без границ слипаются (баг таблицы подписи: «И. О. Фамилия» без пробела).
	b.WriteString(`<w:tbl><w:tblPr><w:tblW w:w="0" w:type="auto"/><w:tblLayout w:type="fixed"/>`)
	if borders {
		b.WriteString(`<w:tblBorders><w:top w:val="single" w:sz="4"/><w:left w:val="single" w:sz="4"/><w:bottom w:val="single" w:sz="4"/><w:right w:val="single" w:sz="4"/><w:insideH w:val="single" w:sz="4"/><w:insideV w:val="single" w:sz="4"/></w:tblBorders>`)
	}
	b.WriteString(`</w:tblPr><w:tblGrid>`)
	for _, w := range widths {
		b.WriteString(fmt.Sprintf(`<w:gridCol w:w="%d"/>`, w))
	}
	b.WriteString(`</w:tblGrid>`)
	if header {
		b.WriteString(`<w:tr>`)
		for i, h := range headers {
			b.WriteString(printDocxCell(h, styles[i], true, 0, styles[i].Align, fam))
		}
		b.WriteString(`</w:tr>`)
	}
	b.WriteString(`<w:tr>`)
	for i, v := range values {
		b.WriteString(printDocxCell(v, styles[i], false, 0, styles[i].Align, fam))
	}
	b.WriteString(`</w:tr></w:tbl>`)
	return b.String()
}

// printDocxCnvIDOffset разводит id в разных областях видимости:
// docPr/@id и pic:cNvPr/@id оба требуют уникальности в пределах документа
// (сам Word пишет их из разных диапазонов: 6 vs 2001819034). Совпадение
// значений — «не удалось прочитать содержимое» с починкой сносом кусков.
const printDocxCnvIDOffset = 100000

// printDocxDrawing — inline-картинка. Размеры в EMU (1 мм = 36000).
// num — уникальный номер картинки в документе (docPr id; cNvPr id
// получается сдвигом и с docPr не пересекается — см. выше).
func printDocxDrawing(num int, rid string, widthMm float64, img []byte) string {
	cx, cy := printImageExtent(img, widthMm)
	return fmt.Sprintf(
		`<w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0">`+
			`<wp:extent cx="%d" cy="%d"/><wp:docPr id="%d" name="img%d"/>`+
			`<a:graphic xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">`+
			`<a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">`+
			`<pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">`+
			`<pic:nvPicPr><pic:cNvPr id="%d" name="img%d"/><pic:cNvPicPr/></pic:nvPicPr>`+
			`<pic:blipFill><a:blip r:embed="%s"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>`+
			`<pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm>`+
			`<a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr>`+
			`</pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing>`,
		cx, cy, num, num, num+printDocxCnvIDOffset, num, rid, cx, cy,
	)
}

// printDocxFloatingDrawing — картинка поверх текста (wp:anchor): x — мм
// от левого края текстовой колонки, y — мм ниже верха абзаца-якоря.
// Так пишет сам Word (см. templ_example.docx: positionH from column,
// positionV from paragraph). Абсолют от страницы здесь непригоден:
// низ документа гуляет от длины переменных полей (напр. примечания),
// а подпись обязана сидеть над своим блоком всегда. behindDoc=0 —
// поверх текста. wrapNone обязателен: без элемента обтекания Word считает
// пакет битым и предлагает восстановить документ.
func printDocxFloatingDrawing(num int, rid string, widthMm, xMm, yMm float64, img []byte) string {
	cx, cy := printImageExtent(img, widthMm)
	return fmt.Sprintf(
		`<w:drawing><wp:anchor distT="0" distB="0" distL="114300" distR="114300" simplePos="0" relativeHeight="1" behindDoc="0" locked="0" layoutInCell="1" allowOverlap="1">`+
			`<wp:simplePos x="0" y="0"/>`+
			`<wp:positionH relativeFrom="column"><wp:posOffset>%d</wp:posOffset></wp:positionH>`+
			`<wp:positionV relativeFrom="paragraph"><wp:posOffset>%d</wp:posOffset></wp:positionV>`+
			`<wp:extent cx="%d" cy="%d"/><wp:wrapNone/>`+
			`<wp:docPr id="%d" name="img%d"/>`+
			`<a:graphic xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">`+
			`<a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">`+
			`<pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">`+
			`<pic:nvPicPr><pic:cNvPr id="%d" name="img%d"/><pic:cNvPicPr/></pic:nvPicPr>`+
			`<pic:blipFill><a:blip r:embed="%s"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>`+
			`<pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm>`+
			`<a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr>`+
			`</pic:pic></a:graphicData></a:graphic></wp:anchor></w:drawing>`,
		int(xMm*36000), int(yMm*36000), cx, cy, num, num, num+printDocxCnvIDOffset, num, rid, cx, cy,
	)
}

// printImageExtent — размеры картинки в EMU с сохранением пропорций.
func printImageExtent(img []byte, widthMm float64) (int, int) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(img))
	cx := int(widthMm * 36000)
	cy := cx
	if err == nil && cfg.Width > 0 {
		cy = int(widthMm * float64(cfg.Height) / float64(cfg.Width) * 36000)
	}
	return cx, cy
}

// printDocxImagePara оборачивает drawing в абзац с выравниванием.
func printDocxImagePara(drawing, align string) string {
	jc := ""
	switch align {
	case "right":
		jc = `<w:jc w:val="right"/>`
	case "left":
		jc = `<w:jc w:val="left"/>`
	default:
		jc = `<w:jc w:val="center"/>`
	}
	return `<w:p><w:pPr>` + jc + `</w:pPr><w:r>` + drawing + `</w:r></w:p>`
}

// docxPart — файл пакета (путь → тело).
type docxPart struct {
	name string
	body string
}

// printDocxImageExt — расширение media-части по magic байтов, а не по суффиксу
// имени: картинки из полей (ключи "field:<имя>") суффикса не имеют и с
// суффиксной логикой всегда падали в png, ломая jpeg. Доверяем байтам,
// как везде в печати (fileMimeByMagic).
func printDocxImageExt(raw []byte) string {
	if mime, ok := fileMimeByMagic(raw); ok && mime == "image/jpeg" {
		return "jpeg"
	}
	return "png"
}

// renderPrintDOCX строит .docx байты из шаблона и строки документа.
// images: src → байты (уже проверенные loadPrintImage).
func renderPrintDOCX(tpl *printTemplate, record map[string]any, images map[string][]byte) ([]byte, error) {
	// rid картинок — заранее и детерминированно (порядок блоков),
	// иначе тело и rels разъедутся.
	var imgOrder []string
	imgRids := make(map[string]string, len(images))
	for _, b := range tpl.Blocks {
		if b.Type != "image" {
			continue
		}
		// Пропущенный field-логотип (пустое поле) в пакет не попадает —
		// ниже такой блок молча пропускается, как пустой field-блок.
		if _, ok := images[b.imageKey()]; !ok {
			continue
		}
		if _, dup := imgRids[b.imageKey()]; dup {
			continue
		}
		imgOrder = append(imgOrder, b.imageKey())
		imgRids[b.imageKey()] = fmt.Sprintf("rIdImg%d", len(imgOrder))
	}

	var body strings.Builder

	tplFam := printDocxFont(printFontFamily(tpl, printTextStyle{}))
	if strings.TrimSpace(renderPrintText(tpl.Title, record)) != "" {
		body.WriteString(printDocxPara(renderPrintText(tpl.Title, record),
			printTextStyle{Bold: true, Align: "center"}, 16, true, tplFam))
	}
	for _, b := range tpl.Blocks {
		switch b.Type {
		case "heading":
			size := 13.0
			if b.Size != 0 {
				size = b.Size
			}
			body.WriteString(printDocxPara(renderPrintText(b.Text, record), b.printTextStyle, size, false, printDocxFont(printFontFamily(tpl, b.printTextStyle))))
		case "field":
			if strings.TrimSpace(stringifyPrintValue(record[b.Field])) == "" {
				continue
			}
			fam := printDocxFont(printFontFamily(tpl, b.printTextStyle))
			body.WriteString(printDocxPara(
				b.Label+": "+stringifyPrintValue(record[b.Field]), b.printTextStyle, 11, false, fam))
		case "text":
			body.WriteString(printDocxPara(renderPrintText(b.Text, record), b.printTextStyle, 11, false, printDocxFont(printFontFamily(tpl, b.printTextStyle))))
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
				body.WriteString(printDocxPara(prefix+rendered, b.printTextStyle, 11, false, printDocxFont(printFontFamily(tpl, b.printTextStyle))))
			}
		case "image":
			img, ok := images[b.imageKey()]
			if !ok {
				// Поле-логотип пустое — пропускаем блок молча, как пустой
				// field-блок. Битая src-картинка сюда не попадает никогда:
				// её отсутствие — 500 ещё на загрузке (ошибка оператора).
				if b.Src != "" {
					return nil, fmt.Errorf("print template: image %q not loaded", b.Src)
				}
				continue
			}
			num := 0
			for i, src := range imgOrder {
				if src == b.imageKey() {
					num = i + 1
					break
				}
			}
			if num == 0 {
				return nil, fmt.Errorf("print template: image %q not registered", b.imageKey())
			}
			if b.Float {
				// Якорь вставляется сырым XML: через printDocxPara его бы
				// заэскейпило в текст. Выравнивание абзаца на anchor не влияет.
				// Контейнер якоря — нулевой высоты (exact 1 twips + шрифт 1пт):
				// в примере якорь вообще живёт внутри содержательного абзаца
				// и места в потоке не занимает; иначе весь текст ниже уезжает
				// на строку вниз и подпись относительно него выглядит выше
				// (в PDF div-контейнер и так нулевой — там всё сходилось).
				body.WriteString(`<w:p><w:pPr><w:spacing w:after="0" w:before="0" w:line="1" w:lineRule="exact"/><w:rPr><w:sz w:val="2"/></w:rPr></w:pPr><w:r>` + printDocxFloatingDrawing(num, imgRids[b.imageKey()], b.Width, b.X, b.Y, img) + `</w:r></w:p>`)
			} else {
				body.WriteString(printDocxImagePara(printDocxDrawing(num, imgRids[b.imageKey()], b.Width, img), b.Align))
			}
		case "table":
			tblFam := printDocxFont(printFontFamily(tpl, b.printTextStyle))
			size := 11.0
			if b.Size != 0 {
				size = b.Size
			}
			if b.Title != "" {
				tsize := 12.0
				if b.Size != 0 {
					tsize = b.Size
				}
				body.WriteString(printDocxPara(renderPrintText(b.Title, record),
					printTextStyle{Bold: true}, tsize, true, tblFam))
			}
			if len(b.Columns) > 0 {
				headers := make([]string, 0, len(b.Columns))
				values := make([]string, 0, len(b.Columns))
				styles := make([]printTextStyle, 0, len(b.Columns))
				colW := printTwipsWidths(b.Columns, printDocxContentTwips(tpl))
				for _, c := range b.Columns {
					headers = append(headers, c.Label)
					values = append(values, printColumnValue(c, record))
					styles = append(styles, printColumnStyle(tpl, b.printTextStyle, c))
				}
				body.WriteString(printDocxTable(headers, values, styles, colW, printBlockShow(b.Header), printBlockShow(b.Borders), tblFam))
			} else {
				tblFam := printDocxFont(printFontFamily(tpl, b.printTextStyle))
				var rows strings.Builder
				borders := `<w:tblBorders><w:top w:val="single" w:sz="4"/><w:left w:val="single" w:sz="4"/><w:bottom w:val="single" w:sz="4"/><w:right w:val="single" w:sz="4"/><w:insideH w:val="single" w:sz="4"/><w:insideV w:val="single" w:sz="4"/></w:tblBorders>`
				if !printBlockShow(b.Borders) {
					borders = ""
				}
				contentTw := printDocxContentTwips(tpl)
				rows.WriteString(fmt.Sprintf(`<w:tbl><w:tblPr><w:tblW w:w="0" w:type="auto"/><w:tblLayout w:type="fixed"/>`+borders+`</w:tblPr><w:tblGrid><w:gridCol w:w="%d"/><w:gridCol w:w="%d"/></w:tblGrid>`, contentTw/3, contentTw-contentTw/3))
				for _, r := range b.Rows {
					rows.WriteString(`<w:tr>`)
					rows.WriteString(printDocxCell(r.Label, printTextStyle{Bold: true}, true, size, "left", tblFam))
					rows.WriteString(printDocxCell(stringifyPrintValue(record[r.Field]), printTextStyle{}, false, size, "left", tblFam))
					rows.WriteString(`</w:tr>`)
				}
				rows.WriteString(`</w:tbl>`)
				body.WriteString(rows.String())
			}
		}
	}
	if tpl.Footer != "" {
		body.WriteString(printDocxPara(renderPrintText(tpl.Footer, record),
			printTextStyle{Align: "center", Color: "808080"}, 9, false, tplFam))
	}

	// --- сборка пакета ---
	var rels strings.Builder
	rels.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
		`</Relationships>`)
	showHeader := tpl.Header != "" || tpl.printPageNum() == "header"
	showFooter := tpl.Footer != "" || tpl.printPageNum() == "footer"
	var contentTypes strings.Builder
	contentTypes.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
		`<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>`)
	if showHeader {
		contentTypes.WriteString(`<Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>`)
	}
	if showFooter {
		contentTypes.WriteString(`<Override PartName="/word/footer1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>`)
	}
	// Default на расширение — один (дубли одного Extension путают строгие
	// читалки; раньше при двух png картинках писалось дважды).
	// Расширение — по magic байтов, а не по суффиксу имени: картинки из полей
	// (ключи "field:<имя>") суффикса не имеют и всегда падали в png, ломая jpeg.
	seenExt := make(map[string]bool)
	for src := range images {
		ext := printDocxImageExt(images[src])
		ctype := "image/png"
		if ext == "jpeg" {
			ctype = "image/jpeg"
		}
		if seenExt[ext] {
			continue
		}
		seenExt[ext] = true
		contentTypes.WriteString(`<Default Extension="` + ext + `" ContentType="` + ctype + `"/>`)
	}
	contentTypes.WriteString(`</Types>`)

	var docRels strings.Builder
	docRels.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	nextRel := 2
	headerRel := ""
	if showHeader {
		headerRel = fmt.Sprintf("rIdH%d", nextRel)
		nextRel++
		docRels.WriteString(`<Relationship Id="` + headerRel + `" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/>`)
	}
	footerRel := ""
	if showFooter {
		footerRel = fmt.Sprintf("rIdF%d", nextRel)
		nextRel++
		docRels.WriteString(`<Relationship Id="` + footerRel + `" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer1.xml"/>`)
	}
	for _, src := range imgOrder {
		ext := printDocxImageExt(images[src])
		docRels.WriteString(`<Relationship Id="` + imgRids[src] + `" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/` + imgRids[src] + `.` + ext + `"/>`)
	}
	docRels.WriteString(`</Relationships>`)

	// Поля страницы: дефолт 1440 twips (~2.54 см), шаблон может задать свои.
	top, right, bottom, left := 1440, 1440, 1440, 1440
	if tpl.Margins != nil {
		if tpl.Margins.Top != 0 {
			top = int(tpl.Margins.Top * 56.7)
		}
		if tpl.Margins.Right != 0 {
			right = int(tpl.Margins.Right * 56.7)
		}
		if tpl.Margins.Bottom != 0 {
			bottom = int(tpl.Margins.Bottom * 56.7)
		}
		if tpl.Margins.Left != 0 {
			left = int(tpl.Margins.Left * 56.7)
		}
	}
	// xmlns:a/pic объявлены строго обязательно: картинки (inline и anchor)
	// используют эти префиксы, без объявления Word открывает документ
	// с «обнаружено содержимое, которое не удалось прочитать» и чинит
	// его сносом кусков (у нас так склеивалась таблица подписи).
	// Go-парсер в тестах такое пропускает молча — ловить только явным ассертом.
	documentXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
		`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" ` +
		`xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" ` +
		`xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" ` +
		`xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">` +
		`<w:body>` + body.String() +
		fmt.Sprintf(`<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="%d" w:right="%d" w:bottom="%d" w:left="%d"/>`, top, right, bottom, left)
	if headerRel != "" {
		documentXML += `<w:headerReference w:type="default" r:id="` + headerRel + `"/>`
	}
	if footerRel != "" {
		documentXML += `<w:footerReference w:type="default" r:id="` + footerRel + `"/>`
	}
	documentXML += `</w:sectPr></w:body></w:document>`

	parts := []docxPart{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + contentTypes.String()},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + rels.String()},
		{"word/_rels/document.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + docRels.String()},
		{"word/document.xml", documentXML},
		{"word/styles.xml", printDocxStyles(tplFam)},
	}
	if showFooter {
		parts = append(parts, docxPart{"word/footer1.xml", printDocxFooter(tpl.Footer, tpl.printPageNum() == "footer", tplFam)})
	}
	if showHeader {
		parts = append(parts, docxPart{"word/header1.xml", printDocxHeader(tpl.Header, tpl.printPageNum() == "header", tplFam)})
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, f := range parts {
		fw, err := w.Create(f.name)
		if err != nil {
			return nil, fmt.Errorf("print docx: %w", err)
		}
		if _, err := fw.Write([]byte(f.body)); err != nil {
			return nil, fmt.Errorf("print docx: %w", err)
		}
	}
	// Картинки — байтами, не строками. Имя media-файла = rid (детерминировано).
	for _, src := range imgOrder {
		ext := printDocxImageExt(images[src])
		fw, err := w.Create("word/media/" + imgRids[src] + "." + ext)
		if err != nil {
			return nil, fmt.Errorf("print docx: %w", err)
		}
		if _, err := fw.Write(images[src]); err != nil {
			return nil, fmt.Errorf("print docx: %w", err)
		}
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("print docx: %w", err)
	}
	return buf.Bytes(), nil
}

// printDocxStyles — Normal по умолчанию (шрифт шаблона).
func printDocxStyles(fam string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/>` +
		`<w:rPr><w:rFonts w:ascii="` + fam + `" w:hAnsi="` + fam + `"/><w:sz w:val="22"/></w:rPr></w:style>` +
		`</w:styles>`
}

// printDocxHeader — колонтитул верха: текст и/или номер страницы.
// fldSimple содержит кэшированное значение (пустой fldSimple без runs
// Word считает битым и предлагает восстановить документ).
func printDocxHeader(text string, withPage bool, fam string) string {
	inner := ""
	if text != "" {
		inner = `<w:r><w:rPr><w:sz w:val="18"/><w:rFonts w:ascii="` + fam + `" w:hAnsi="` + fam + `"/><w:color w:val="808080"/></w:rPr><w:t xml:space="preserve">` +
			xmlEscape(text) + ` </w:t></w:r>`
	}
	if withPage {
		inner += `<w:fldSimple w:instr=" PAGE "><w:r><w:rPr><w:sz w:val="18"/><w:rFonts w:ascii="` + fam + `" w:hAnsi="` + fam + `"/><w:color w:val="808080"/></w:rPr><w:t xml:space="preserve">1</w:t></w:r></w:fldSimple>`
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:p><w:pPr><w:jc w:val="center"/></w:pPr>` + inner + `</w:p></w:hdr>`
}

// printDocxFooter — колонтитул низа: текст и/или "Стр. N".
func printDocxFooter(text string, withPage bool, fam string) string {
	inner := ""
	if text != "" {
		inner = `<w:r><w:rPr><w:sz w:val="18"/><w:rFonts w:ascii="` + fam + `" w:hAnsi="` + fam + `"/><w:color w:val="808080"/></w:rPr><w:t xml:space="preserve">` +
			xmlEscape(text) + ` </w:t></w:r>`
	}
	if withPage {
		inner += `<w:r><w:rPr><w:sz w:val="18"/><w:rFonts w:ascii="` + fam + `" w:hAnsi="` + fam + `"/><w:color w:val="808080"/></w:rPr>` +
			`<w:t xml:space="preserve">Стр. </w:t></w:r>` +
			`<w:fldSimple w:instr=" PAGE "><w:r><w:rPr><w:sz w:val="18"/><w:rFonts w:ascii="` + fam + `" w:hAnsi="` + fam + `"/><w:color w:val="808080"/></w:rPr><w:t xml:space="preserve">1</w:t></w:r></w:fldSimple>`
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:ftr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:p><w:pPr><w:jc w:val="center"/></w:pPr>` + inner + `</w:p></w:ftr>`
}
