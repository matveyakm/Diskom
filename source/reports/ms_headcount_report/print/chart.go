package print

import (
	"bytes"
	"fmt"
	"time"

	chart "github.com/wcharczuk/go-chart/v2"
	"shared/report"
)

// chartSlot — график численности по таблице staff: время (ось X, дни) —
// накопленный итог (ось Y). Отдельный filler, не таблица: картинка рисуется
// один раз и кладётся в image-слот (см. print.json). Пустая история валидна
// (как пустая таблица): рисуется плоская нулевая линия за период.
// go-chart детерминирован (проверено тестом: два рендера побайтово равны).
func chartSlot(ctx report.Ctx, _ report.Params) (any, error) {
	rec := ctx.Record()
	from, err := reportDay(rec["date_from"])
	if err != nil {
		return nil, fmt.Errorf("chart: date_from: %w", err)
	}
	to, err := reportDay(rec["date_to"])
	if err != nil {
		return nil, fmt.Errorf("chart: date_to: %w", err)
	}
	if from.After(to) {
		return nil, fmt.Errorf("chart: date_from %s позже date_to %s",
			from.Format("02.01.2006"), to.Format("02.01.2006"))
	}
	rows, err := ctx.Fetch("employee_data_ledger", map[string]string{
		"ddate__lte": to.Format(time.RFC3339),
		"sort":       "ddate",
		"order":      "asc",
	})
	if err != nil {
		return nil, fmt.Errorf("chart: ledger: %w", err)
	}
	hist, err := staffHistory(rows)
	if err != nil {
		return nil, err
	}
	px, py := chartPoints(from, to, hist)
	xs, ys := stepPoints(px, py, to)
	ts := chart.TimeSeries{
		XValues: xs,
		YValues: ys,
		Style: chart.Style{
			StrokeColor: chart.ColorBlack,
			StrokeWidth: 2,
			DotColor:    chart.ColorBlack,
			DotWidth:    3,
		},
	}
	maxY := 1.0
	for _, y := range ys {
		if y+1 > maxY {
			maxY = y + 1
		}
	}
	g := chart.Chart{
		Width:  1024,
		Height: 400,
		XAxis: chart.XAxis{
			ValueFormatter: chart.TimeDateValueFormatter,
		},
		YAxis: chart.YAxis{
			ValueFormatter: chart.IntValueFormatter,
			Range:          &chart.ContinuousRange{Min: 0, Max: maxY},
		},
		Series: []chart.Series{ts},
	}
	var buf bytes.Buffer
	if err := g.Render(chart.PNG, &buf); err != nil {
		return nil, fmt.Errorf("chart: render: %w", err)
	}
	return buf.Bytes(), nil
}

// chartPoints режет историю на период и якорит линию: baseline (итог на
// начало периода) + change-дни внутри. Одинаковые X схлопываются (последний
// побеждает — change-день важнее якоря). Меньше двух разных X go-chart
// роняет паникой — добиваем плоским хвостом (+24ч тем же итогом).
// Это честно: линия горизонтальна там, где событий нет.
func chartPoints(from, to time.Time, hist []staffPoint) ([]time.Time, []float64) {
	type xy struct {
		x time.Time
		y int64
	}
	var pts []xy
	prev := int64(0)
	for _, p := range hist {
		if p.day.After(to) {
			break
		}
		if p.day.Before(from) {
			prev = p.total
			continue
		}
		if len(pts) == 0 || !p.day.Equal(pts[len(pts)-1].x) {
			pts = append(pts, xy{p.day, p.total})
		} else {
			pts[len(pts)-1].y = p.total
		}
		prev = p.total
	}
	if len(pts) == 0 || !pts[0].x.Equal(from) {
		pts = append([]xy{{from, prev}}, pts...)
	}
	for len(pts) < 2 || pts[len(pts)-1].x.Equal(pts[0].x) {
		last := pts[len(pts)-1]
		pts = append(pts, xy{last.x.Add(24 * time.Hour), last.y})
	}
	xs := make([]time.Time, 0, len(pts))
	ys := make([]float64, 0, len(pts))
	for _, p := range pts {
		xs = append(xs, p.x)
		ys = append(ys, float64(p.y))
	}
	return xs, ys
}

// stepPoints разворачивает ломаную в ступеньки: численность постоянна
// между событиями, диагонали врали бы про плавное изменение. Перед каждым
// change-днём добавляется точка (день, прошлый итог) — вертикальный скачок
// в день события. X строго растут (история уникальна по дням).
func stepPoints(xs []time.Time, ys []float64, to time.Time) ([]time.Time, []float64) {
	if len(xs) == 0 {
		return xs, ys
	}
	outX := []time.Time{xs[0]}
	outY := []float64{ys[0]}
	for i := 1; i < len(xs); i++ {
		if xs[i].After(outX[len(outX)-1]) {
			outX = append(outX, xs[i])
			outY = append(outY, outY[len(outY)-1])
		}
		outX = append(outX, xs[i])
		outY = append(outY, ys[i])
	}
	// Плоский хвост до конца периода: после последнего события
	// численность не менялась (выборка идёт по ddate <= to).
	if last := outX[len(outX)-1]; last.Before(to) {
		outX = append(outX, to)
		outY = append(outY, outY[len(outY)-1])
	}
	return outX, outY
}
