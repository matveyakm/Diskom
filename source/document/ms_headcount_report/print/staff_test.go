package print

import (
	"errors"
	"shared/report"
	"testing"
	"time"
)

type stubCtx struct {
	record  map[string]any
	rows    []map[string]any
	rowsErr error
}

func (s *stubCtx) Record() map[string]any { return s.record }
func (s *stubCtx) Fetch(ref string, q map[string]string) ([]map[string]any, error) {
	if s.rowsErr != nil {
		return nil, s.rowsErr
	}
	return s.rows, nil
}
func (s *stubCtx) FetchOne(ref string, q map[string]string) (map[string]any, error) {
	return nil, errors.New("not needed")
}
func (s *stubCtx) FilesOf(ref string, docID int64) ([]report.FileMeta, error) {
	return nil, errors.New("not needed")
}
func (s *stubCtx) FileBytes(ref string, fileID, docID int64) ([]byte, string, error) {
	return nil, "", errors.New("not needed")
}

func ledgerRow(ddate string, balance float64) map[string]any {
	return map[string]any{"ddate": ddate, "balance": balance}
}

func TestStaffSlot(t *testing.T) {
	at := func(s string) time.Time {
		tm, _ := time.Parse(time.RFC3339, s)
		return tm
	}
	rec := map[string]any{
		"id_committee": float64(1),
		"date_from":    at("2026-01-01T00:00:00Z"),
		"date_to":      at("2026-03-31T00:00:00Z"),
	}
	rows := []map[string]any{
		ledgerRow("2025-12-20T00:00:00Z", 1), // до периода: входит в стартовый итог
		ledgerRow("2026-01-10T00:00:00Z", 1),
		ledgerRow("2026-01-10T12:00:00Z", 1), // тот же день: суммируется
		ledgerRow("2026-02-01T00:00:00Z", -1),
		ledgerRow("2026-02-05T00:00:00Z", 1),
		ledgerRow("2026-03-01T00:00:00Z", -1), // нулевое нетто за день — пропуск
		ledgerRow("2026-03-01T12:00:00Z", 1),
		ledgerRow("2026-04-01T00:00:00Z", 1), // после периода — мимо
	}
	got, err := staffSlot(&stubCtx{record: rec, rows: rows}, nil)
	if err != nil {
		t.Fatalf("staff: %v", err)
	}
	list, ok := got.([]map[string]any)
	if !ok {
		t.Fatalf("want rows, got %T", got)
	}
	want := []struct {
		date  string
		count float64
	}{
		{"10.01.2026", 3}, // 1 + 2 найма
		{"01.02.2026", 2}, // -1
		{"05.02.2026", 3}, // +1
	}
	if len(list) != len(want) {
		t.Fatalf("rows = %+v, want %+v", list, want)
	}
	for i := range want {
		if list[i]["date"] != want[i].date || list[i]["count"] != want[i].count {
			t.Fatalf("row %d = %+v, want %+v", i, list[i], want[i])
		}
	}
	// Пусто — валидно (только шапка).
	got, err = staffSlot(&stubCtx{record: rec, rows: nil}, nil)
	if err != nil {
		t.Fatalf("empty: %v", err)
	}
	if list, _ := got.([]map[string]any); len(list) != 0 {
		t.Fatalf("empty must stay empty, got %+v", list)
	}
	// from > to, нет комитета, ошибка fetch — громко.
	badFrom := map[string]any{"id_committee": float64(1), "date_from": at("2026-05-01T00:00:00Z"), "date_to": at("2026-01-01T00:00:00Z")}
	for _, tc := range []map[string]any{
		{},
		{"id_committee": float64(1)},
		badFrom,
		rec,
	} {
		s := &stubCtx{record: tc}
		if tc["date_from"] != nil {
			s.rowsErr = errors.New("boom")
		}
		if _, err := staffSlot(s, nil); err == nil {
			t.Fatalf("want error for %+v", tc)
		}
	}
}
