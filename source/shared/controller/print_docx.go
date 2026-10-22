package restController

// DOCX-рендер печатной формы на stdlib (archive/zip + ручной XML).
//
// Новых зависимостей ноль осознанно: документ простой (абзацы + одна
// таблица на блок), а тащить unioffice ради этого — лишний вес и вопросы
// лицензирования. Минимальный набор частей, который открывают Word и
// LibreOffice: [Content_Types], _rels, word/document, word/styles.

import (
	"archive/zip"
	"bytes"
	"fmt"
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

func printDocxPara(text string, bold bool, sizeHalfPoints int, center bool) string {
	style := ""
	if bold {
		style = `<w:b/>`
	}
	just := ""
	if center {
		just = `<w:jc w:val="center"/>`
	}
	return fmt.Sprintf(
		`<w:p><w:pPr>%s<w:spacing w:after="120"/></w:pPr><w:r><w:rPr>%s<w:sz w:val="%d"/><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/></w:rPr><w:t xml:space="preserve">%s</w:t></w:r></w:p>`,
		just, style, sizeHalfPoints, xmlEscape(text),
	)
}

func printDocxCell(text string, bold bool) string {
	style := ""
	if bold {
		style = `<w:b/>`
	}
	return fmt.Sprintf(
		`<w:tc><w:tcPr><w:tcW w:w="0" w:type="auto"/></w:tcPr><w:p><w:r><w:rPr>%s<w:sz w:val="22"/><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/></w:rPr><w:t xml:space="preserve">%s</w:t></w:r></w:p></w:tc>`,
		style, xmlEscape(text),
	)
}

func printDocxTable(headers []string, values []string) string {
	var b strings.Builder
	b.WriteString(`<w:tbl><w:tblPr><w:tblW w:w="0" w:type="auto"/><w:tblBorders><w:top w:val="single" w:sz="4"/><w:left w:val="single" w:sz="4"/><w:bottom w:val="single" w:sz="4"/><w:right w:val="single" w:sz="4"/><w:insideH w:val="single" w:sz="4"/><w:insideV w:val="single" w:sz="4"/></w:tblBorders></w:tblPr><w:tblGrid>`)
	for range headers {
		b.WriteString(`<w:gridCol w:w="3000"/>`)
	}
	b.WriteString(`</w:tblGrid><w:tr>`)
	for _, h := range headers {
		b.WriteString(printDocxCell(h, true))
	}
	b.WriteString(`</w:tr><w:tr>`)
	for _, v := range values {
		b.WriteString(printDocxCell(v, false))
	}
	b.WriteString(`</w:tr></w:tbl>`)
	return b.String()
}

// renderPrintDOCX строит .docx байты из шаблона и строки документа.
func renderPrintDOCX(tpl *printTemplate, record map[string]any) ([]byte, error) {
	var body strings.Builder
	body.WriteString(printDocxPara(renderPrintText(tpl.Title, record), true, 32, true))
	for _, b := range tpl.Blocks {
		center := b.Align == "center"
		switch b.Type {
		case "heading":
			body.WriteString(printDocxPara(renderPrintText(b.Text, record), true, 26, center))
		case "field":
			if strings.TrimSpace(stringifyPrintValue(record[b.Field])) == "" {
				continue
			}
			body.WriteString(printDocxPara(b.Label+": "+stringifyPrintValue(record[b.Field]), false, 22, false))
		case "text":
			body.WriteString(printDocxPara(renderPrintText(b.Text, record), false, 22, center))
		case "table":
			if b.Title != "" {
				body.WriteString(printDocxPara(renderPrintText(b.Title, record), true, 24, false))
			}
			headers := make([]string, 0, len(b.Columns))
			values := make([]string, 0, len(b.Columns))
			for _, c := range b.Columns {
				headers = append(headers, c.Label)
				values = append(values, stringifyPrintValue(record[c.Field]))
			}
			body.WriteString(printDocxTable(headers, values))
		}
	}

	documentXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:body>` + body.String() +
		`<w:sectPr><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr>` +
		`</w:body></w:document>`

	const contentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
		`<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>` +
		`</Types>`

	const rels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
		`</Relationships>`

	const styles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/>` +
		`<w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:sz w:val="22"/></w:rPr></w:style>` +
		`</w:styles>`

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, f := range []struct{ name, body string }{
		{"[Content_Types].xml", contentTypes},
		{"_rels/.rels", rels},
		{"word/document.xml", documentXML},
		{"word/styles.xml", styles},
	} {
		fw, err := w.Create(f.name)
		if err != nil {
			return nil, fmt.Errorf("print docx: %w", err)
		}
		if _, err := fw.Write([]byte(f.body)); err != nil {
			return nil, fmt.Errorf("print docx: %w", err)
		}
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("print docx: %w", err)
	}
	return buf.Bytes(), nil
}
