package restController

// PDF-рендер печатной формы на maroto v2 (тот же движок и шрифты,
// что у ms_job_report: Arial + кириллица из коробки).

import (
	_ "embed"
	"strings"
	"sync"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
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

// Шрифты грузятся один раз на процесс (per-request грузить — расточительно,
// maroto-инстанс при этом всё равно новый на каждый документ).
var printFontsOnce = sync.OnceValue(func() []entity.CustomFont {
	customFonts, err := fontrepository.New().
		AddUTF8FontFromBytes("Arial", fontstyle.Normal, printArialRegular).
		AddUTF8FontFromBytes("Arial", fontstyle.Bold, printArialBold).
		Load()
	if err != nil {
		panic("print: load fonts: " + err.Error())
	}
	return customFonts
})

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
func renderPrintPDF(tpl *printTemplate, record map[string]any) ([]byte, error) {
	m := newPrintMaroto()

	m.AddRow(20, col.New(12).Add(
		text.New(renderPrintText(tpl.Title, record), props.Text{
			Top:   5,
			Size:  16,
			Style: fontstyle.Bold,
			Align: align.Center,
		}),
	))
	m.AddRow(8)

	for _, b := range tpl.Blocks {
		switch b.Type {
		case "heading":
			m.AddRow(12, col.New(12).Add(
				text.New(renderPrintText(b.Text, record), props.Text{
					Size:  13,
					Style: fontstyle.Bold,
					Align: printAlign(b.Align),
				}),
			))
			m.AddRow(4)
		case "field":
			// Пустое поле — пропускаем: приказы не печатают "Примечание: ".
			if strings.TrimSpace(stringifyPrintValue(record[b.Field])) == "" {
				continue
			}
			m.AddRow(8, col.New(12).Add(
				text.New(b.Label+": "+stringifyPrintValue(record[b.Field]), props.Text{Size: 10}),
			))
		case "text":
			m.AddRow(8, col.New(12).Add(
				text.New(renderPrintText(b.Text, record), props.Text{
					Size:  10,
					Align: printAlign(b.Align),
				}),
			))
		case "table":
			n := len(b.Columns)
			widths := printColumnWidths(n)
			if b.Title != "" {
				m.AddRow(10, col.New(12).Add(
					text.New(renderPrintText(b.Title, record), props.Text{Size: 11, Style: fontstyle.Bold}),
				))
			}
			header := row.New(10)
			for i, c := range b.Columns {
				header.Add(col.New(widths[i]).Add(
					text.New(c.Label, props.Text{Size: 9, Style: fontstyle.Bold}),
				))
			}
			m.AddRows(header)
			data := row.New(10)
			for i, c := range b.Columns {
				data.Add(col.New(widths[i]).Add(
					text.New(stringifyPrintValue(record[c.Field]), props.Text{Size: 9}),
				))
			}
			m.AddRows(data)
			m.AddRow(4)
		}
	}

	document, err := m.Generate()
	if err != nil {
		return nil, err
	}
	return document.GetBytes(), nil
}

// printAlign маппит DSL-выравнивание в maroto (дефолт — влево).
func printAlign(a string) align.Type {
	if a == "center" {
		return align.Center
	}
	return align.Left
}

// printColumnWidths делит сетку 12 на n колонок целыми ширинами.
func printColumnWidths(n int) []int {
	widths := make([]int, n)
	base := 12 / n
	rest := 12 % n
	for i := range widths {
		widths[i] = base
		if i < rest {
			widths[i]++
		}
	}
	return widths
}
