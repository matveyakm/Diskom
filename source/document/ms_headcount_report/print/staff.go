package print

// Package print — индивидуальный код печати отчёта: filler-ы слотов
// шаблона print/print.json + их реестр. Главный движок (shared) документ
// не смотрит: связь дырки и функции — совпадением имён, смысл — только
// в позиции в шаблоне.

import (
	"fmt"
	"shared/report"
	"sort"
	"strings"
	"time"
)

// Slots — реестр дырок печати отчёта.
var Slots = report.Slots{"staff": staffSlot}

// staffSlot — таблица «Дата | Численность»: только дни периода, в которые
// численность менялась. Источник — employee_data_ledger (+1 приём, -1
// увольнение) с фильтром по комитету. Пустой итог —
// валиден: рисуется только шапка.
func staffSlot(ctx report.Ctx, _ report.Params) (any, error) {
	rec := ctx.Record()
	org, ok := report.Int(rec["id_committee"])
	if !ok {
		return nil, fmt.Errorf("staff: нет id_committee в строке")
	}
	from, err := reportDay(rec["date_from"])
	if err != nil {
		return nil, fmt.Errorf("staff: date_from: %w", err)
	}
	to, err := reportDay(rec["date_to"])
	if err != nil {
		return nil, fmt.Errorf("staff: date_to: %w", err)
	}
	if from.After(to) {
		return nil, fmt.Errorf("staff: date_from %s позже date_to %s",
			from.Format("02.01.2006"), to.Format("02.01.2006"))
	}
	rows, err := ctx.Fetch("employee_data_ledger", map[string]string{
		"id_committee": report.Str(org),
		"ddate__lte":   to.Format(time.RFC3339),
		"sort":         "ddate",
		"order":        "asc",
	})
	if err != nil {
		return nil, fmt.Errorf("staff: ledger: %w", err)
	}
	// Нарастающий итог с начала истории; эмитим только дни внутри
	// периода, где итог изменился. День без чистого изменения
	// (приём+увольнение в один день) не эмитится — численность та же.
	byDay := make(map[string]int64)
	var days []time.Time
	seen := make(map[string]bool)
	var total int64
	for _, r := range rows {
		d, err := reportDay(r["ddate"])
		if err != nil {
			return nil, fmt.Errorf("staff: bad row date: %w", err)
		}
		delta, err := reportDelta(r["balance"])
		if err != nil {
			return nil, fmt.Errorf("staff: %w", err)
		}
		total += delta
		key := d.Format("2006-01-02")
		byDay[key] = total
		if !seen[key] {
			seen[key] = true
			days = append(days, time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC))
		}
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	out := make([]map[string]any, 0, len(days))
	var last int64
	started := false
	for _, d := range days {
		if d.Before(from) || d.After(to) {
			continue
		}
		t := byDay[d.Format("2006-01-02")]
		if started && t == last {
			continue
		}
		started = true
		last = t
		out = append(out, map[string]any{
			"date":  d.Format("02.01.2006"),
			"count": float64(t),
		})
	}
	return out, nil
}

// reportDelta приводит balance проводки к знаковому целому
// (report.Int — только положительные id, для проводок не годится).

// reportDelta приводит balance проводки к знаковому целому
// (report.Int — только положительные id, для проводок не годится).
func reportDelta(v any) (int64, error) {
	switch t := v.(type) {
	case float64:
		if t == float64(int64(t)) {
			return int64(t), nil
		}
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	case string:
		var n int64
		if _, err := fmt.Sscanf(strings.TrimSpace(t), "%d", &n); err == nil {
			return n, nil
		}
	}
	return 0, fmt.Errorf("bad balance %v", v)
}

// reportDay приводит значение даты строки к дню (UTC, без времени):
// time.Time из драйвера, RFC3339 или YYYY-MM-DD строкой.
func reportDay(v any) (time.Time, error) {
	if t, ok := v.(time.Time); ok {
		if t.IsZero() {
			return time.Time{}, fmt.Errorf("empty date")
		}
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC), nil
	}
	if s, ok := v.(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			y, m, d := t.Date()
			return time.Date(y, m, d, 0, 0, 0, 0, time.UTC), nil
		}
		if t, err := time.Parse("2006-01-02", s); err == nil {
			return t, nil
		}
		if t, err := time.Parse("02.01.2006", s); err == nil {
			return t, nil
		}
		return time.Time{}, fmt.Errorf("bad date %q", s)
	}
	return time.Time{}, fmt.Errorf("bad date %v", v)
}
