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

// printDocxRun — текстовый ран с оформлением блока.
func printDocxRun(text string, s printTextStyle, defSize float64, defBold bool) string {
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
	return fmt.Sprintf(
		`<w:r><w:rPr>%s<w:sz w:val="%d"/><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/></w:rPr><w:t xml:space="preserve">%s</w:t></w:r>`,
		style, int(size*2), xmlEscape(text),
	)
}

// printDocxPara — абзац. center/right/justify через jc; indent/firstLine
// в twips (1 мм = 56.7 twips, firstLine — точный, в отличие от PDF).
func printDocxPara(text string, s printTextStyle, defSize float64, defBold bool) string {
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
	if s.Indent != 0 || s.FirstLine != 0 {
		ind = fmt.Sprintf(`<w:ind w:left="%d" w:firstLine="%d"/>`,
			int(s.Indent*56.7), int(s.FirstLine*56.7))
	}
	return fmt.Sprintf(
		`<w:p><w:pPr>%s%s<w:spacing w:after="120"/></w:pPr>%s</w:p>`,
		jc, ind, printDocxRun(text, s, defSize, defBold),
	)
}

func printDocxCell(text string, bold bool, size float64, align string) string {
	style := ""
	if bold {
		style = `<w:b/>`
	}
	jc := ""
	switch align {
	case "center":
		jc = `<w:jc w:val="center"/>`
	case "right":
		jc = `<w:jc w:val="right"/>`
	}
	return fmt.Sprintf(
		`<w:tc><w:tcPr><w:tcW w:w="0" w:type="auto"/></w:tcPr><w:p><w:pPr>%s</w:pPr><w:r><w:rPr>%s<w:sz w:val="%d"/><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/></w:rPr><w:t xml:space="preserve">%s</w:t></w:r></w:p></w:tc>`,
		jc, style, int(size*2), xmlEscape(text),
	)
}

func printDocxTable(headers []string, values []string, aligns []string, size float64) string {
	var b strings.Builder
	b.WriteString(`<w:tbl><w:tblPr><w:tblW w:w="0" w:type="auto"/><w:tblBorders><w:top w:val="single" w:sz="4"/><w:left w:val="single" w:sz="4"/><w:bottom w:val="single" w:sz="4"/><w:right w:val="single" w:sz="4"/><w:insideH w:val="single" w:sz="4"/><w:insideV w:val="single" w:sz="4"/></w:tblBorders></w:tblPr><w:tblGrid>`)
	for range headers {
		b.WriteString(`<w:gridCol w:w="3000"/>`)
	}
	b.WriteString(`</w:tblGrid><w:tr>`)
	for i, h := range headers {
		b.WriteString(printDocxCell(h, true, size, aligns[i]))
	}
	b.WriteString(`</w:tr><w:tr>`)
	for i, v := range values {
		b.WriteString(printDocxCell(v, false, size, aligns[i]))
	}
	b.WriteString(`</w:tr></w:tbl>`)
	return b.String()
}

// printDocxDrawing — inline-картинка. Размеры в EMU (1 мм = 36000).
func printDocxDrawing(rid string, widthMm float64, img []byte) string {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(img))
	cx := int(widthMm * 36000)
	cy := cx
	if err == nil && cfg.Width > 0 {
		cy = int(widthMm * float64(cfg.Height) / float64(cfg.Width) * 36000)
	}
	return fmt.Sprintf(
		`<w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0">`+
			`<wp:extent cx="%d" cy="%d"/><wp:docPr id="1" name="img"/>`+
			`<a:graphic xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">`+
			`<a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">`+
			`<pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">`+
			`<pic:nvPicPr><pic:cNvPr id="1" name="img"/><pic:cNvPicPr/></pic:nvPicPr>`+
			`<pic:blipFill><a:blip r:embed="%s"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>`+
			`<pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm>`+
			`<a:prstGeom prst="rect"><a:avLst/></a:prstGeom></a:spPr>`+
			`</pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing>`,
		cx, cy, rid, cx, cy,
	)
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
		if _, dup := imgRids[b.Src]; dup {
			continue
		}
		imgOrder = append(imgOrder, b.Src)
		imgRids[b.Src] = fmt.Sprintf("rIdImg%d", len(imgOrder))
	}

	var body strings.Builder

	body.WriteString(printDocxPara(renderPrintText(tpl.Title, record),
		printTextStyle{Bold: true, Align: "center"}, 16, true))
	for _, b := range tpl.Blocks {
		switch b.Type {
		case "heading":
			size := 13.0
			if b.Size != 0 {
				size = b.Size
			}
			body.WriteString(printDocxPara(renderPrintText(b.Text, record), b.printTextStyle, size, true))
		case "field":
			if strings.TrimSpace(stringifyPrintValue(record[b.Field])) == "" {
				continue
			}
			body.WriteString(printDocxPara(
				b.Label+": "+stringifyPrintValue(record[b.Field]), b.printTextStyle, 11, false))
		case "text":
			body.WriteString(printDocxPara(renderPrintText(b.Text, record), b.printTextStyle, 11, false))
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
				body.WriteString(printDocxPara(prefix+rendered, b.printTextStyle, 11, false))
			}
		case "image":
			img := images[b.Src]
			body.WriteString(printDocxImagePara(printDocxDrawing(imgRids[b.Src], b.Width, img), b.Align))
		case "table":
			if b.Title != "" {
				size := 12.0
				if b.Size != 0 {
					size = b.Size
				}
				body.WriteString(printDocxPara(renderPrintText(b.Title, record),
					printTextStyle{Bold: true}, size, true))
			}
			size := 11.0
			if b.Size != 0 {
				size = b.Size
			}
			if len(b.Columns) > 0 {
				headers := make([]string, 0, len(b.Columns))
				values := make([]string, 0, len(b.Columns))
				aligns := make([]string, 0, len(b.Columns))
				for _, c := range b.Columns {
					headers = append(headers, c.Label)
					values = append(values, stringifyPrintValue(record[c.Field]))
					aligns = append(aligns, c.Align)
				}
				body.WriteString(printDocxTable(headers, values, aligns, size))
			} else {
				var rows strings.Builder
				rows.WriteString(`<w:tbl><w:tblPr><w:tblW w:w="0" w:type="auto"/><w:tblBorders><w:top w:val="single" w:sz="4"/><w:left w:val="single" w:sz="4"/><w:bottom w:val="single" w:sz="4"/><w:right w:val="single" w:sz="4"/><w:insideH w:val="single" w:sz="4"/><w:insideV w:val="single" w:sz="4"/></w:tblBorders></w:tblPr><w:tblGrid><w:gridCol w:w="3000"/><w:gridCol w:w="6000"/></w:tblGrid>`)
				for _, r := range b.Rows {
					rows.WriteString(`<w:tr>`)
					rows.WriteString(printDocxCell(r.Label, true, size, ""))
					rows.WriteString(printDocxCell(stringifyPrintValue(record[r.Field]), false, size, ""))
					rows.WriteString(`</w:tr>`)
				}
				rows.WriteString(`</w:tbl>`)
				body.WriteString(rows.String())
			}
		}
	}
	if tpl.Footer != "" {
		body.WriteString(printDocxPara(renderPrintText(tpl.Footer, record),
			printTextStyle{Align: "center", Color: "808080"}, 9, false))
	}

	// --- сборка пакета ---
	var rels strings.Builder
	rels.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
		`</Relationships>`)
	var contentTypes strings.Builder
	contentTypes.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
		`<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>`)
	if tpl.Header != "" {
		contentTypes.WriteString(`<Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>`)
	}
	contentTypes.WriteString(`<Override PartName="/word/footer1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>`)
	for src := range images {
		ext := "png"
		ctype := "image/png"
		if strings.HasSuffix(strings.ToLower(src), ".jpg") || strings.HasSuffix(strings.ToLower(src), ".jpeg") {
			ext, ctype = "jpeg", "image/jpeg"
		}
		contentTypes.WriteString(`<Default Extension="` + ext + `" ContentType="` + ctype + `"/>`)
	}
	contentTypes.WriteString(`</Types>`)

	var docRels strings.Builder
	docRels.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	nextRel := 2
	headerRel := ""
	if tpl.Header != "" {
		headerRel = fmt.Sprintf("rIdH%d", nextRel)
		nextRel++
		docRels.WriteString(`<Relationship Id="` + headerRel + `" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/>`)
	}
	footerRel := fmt.Sprintf("rIdF%d", nextRel)
	nextRel++
	docRels.WriteString(`<Relationship Id="` + footerRel + `" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer1.xml"/>`)
	for _, src := range imgOrder {
		ext := "png"
		if strings.HasSuffix(strings.ToLower(src), ".jpg") || strings.HasSuffix(strings.ToLower(src), ".jpeg") {
			ext = "jpeg"
		}
		docRels.WriteString(`<Relationship Id="` + imgRids[src] + `" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/` + imgRids[src] + `.` + ext + `"/>`)
	}
	docRels.WriteString(`</Relationships>`)

	documentXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
		`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" ` +
		`xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing">` +
		`<w:body>` + body.String() +
		`<w:sectPr><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/>`
	if headerRel != "" {
		documentXML += `<w:headerReference w:type="default" r:id="` + headerRel + `"/>`
	}
	documentXML += `<w:footerReference w:type="default" r:id="` + footerRel + `"/>` +
		`</w:sectPr></w:body></w:document>`

	parts := []docxPart{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + contentTypes.String()},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + rels.String()},
		{"word/_rels/document.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + docRels.String()},
		{"word/document.xml", documentXML},
		{"word/styles.xml", printDocxStyles()},
		{"word/footer1.xml", printDocxFooter(tpl.Footer)},
	}
	if tpl.Header != "" {
		parts = append(parts, docxPart{"word/header1.xml", printDocxHeader(tpl.Header, nil)})
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
		ext := "png"
		if strings.HasSuffix(strings.ToLower(src), ".jpg") || strings.HasSuffix(strings.ToLower(src), ".jpeg") {
			ext = "jpeg"
		}
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

// printDocxStyles — минимальный Normal (Arial).
func printDocxStyles() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/>` +
		`<w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:sz w:val="22"/></w:rPr></w:style>` +
		`</w:styles>`
}

// printDocxHeader — колонтитул верха (статический текст).
func printDocxHeader(text string, _ map[string]any) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:sz w:val="18"/><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:color w:val="808080"/></w:rPr><w:t xml:space="preserve">` +
		xmlEscape(text) +
		`</w:t></w:r></w:p></w:hdr>`
}

// printDocxFooter — колонтитул низа: текст + "Стр. N".
func printDocxFooter(text string) string {
	inner := ""
	if text != "" {
		inner = `<w:r><w:rPr><w:sz w:val="18"/><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:color w:val="808080"/></w:rPr><w:t xml:space="preserve">` +
			xmlEscape(text) + ` </w:t></w:r>`
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:ftr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:p><w:pPr><w:jc w:val="center"/></w:pPr>` + inner +
		`<w:r><w:rPr><w:sz w:val="18"/><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:color w:val="808080"/></w:rPr>` +
		`<w:t xml:space="preserve">Стр. </w:t></w:r>` +
		`<w:r><w:rPr><w:sz w:val="18"/><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:color w:val="808080"/></w:rPr>` +
		`<w:fldSimple w:instr="PAGE"/></w:r>` +
		`</w:p></w:ftr>`
}
