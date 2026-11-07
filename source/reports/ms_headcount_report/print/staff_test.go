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
	one     map[string]any
	oneErr  error
	files   []report.FileMeta
	filesEr error
	data    []byte
	mime    string
	dataErr error
}

func (s *stubCtx) Record() map[string]any { return s.record }
func (s *stubCtx) Fetch(ref string, q map[string]string) ([]map[string]any, error) {
	if s.rowsErr != nil {
		return nil, s.rowsErr
	}
	if ref != "employee_data_ledger" {
		return nil, errors.New("unexpected ref " + ref)
	}
	if _, ok := q["id_committee"]; ok {
		return nil, errors.New("committee filter must not be sent")
	}
	return s.rows, nil
}
func (s *stubCtx) FetchOne(ref string, q map[string]string) (map[string]any, error) {
	return s.one, s.oneErr
}
func (s *stubCtx) FilesOf(ref string, docID int64) ([]report.FileMeta, error) {
	return s.files, s.filesEr
}
func (s *stubCtx) FileBytes(ref string, fileID, docID int64) ([]byte, string, error) {
	return s.data, s.mime, s.dataErr
}

// accumulatedRow — строка регистра: balance уже накопленный (считает сервер).
func accumulatedRow(id int64, ddate string, balance float64) map[string]any {
	return map[string]any{"id": float64(id), "ddate": ddate, "balance": balance}
}

func TestStaffSlot(t *testing.T) {
	at := func(s string) time.Time {
		tm, _ := time.Parse(time.RFC3339, s)
		return tm
	}
	rec := map[string]any{
		"date_from": at("2026-01-01T00:00:00Z"),
		"date_to":   at("2026-03-31T00:00:00Z"),
	}
	rows := []map[string]any{
		accumulatedRow(1, "2025-12-20T00:00:00Z", 1), // до периода: базовый итог
		accumulatedRow(2, "2026-01-10T00:00:00Z", 2),
		accumulatedRow(3, "2026-01-10T12:00:00Z", 3), // тот же день: последний итог дня
		accumulatedRow(4, "2026-02-01T00:00:00Z", 2),
		accumulatedRow(5, "2026-02-05T00:00:00Z", 3),
		accumulatedRow(6, "2026-03-01T00:00:00Z", 2), // день без чистого изменения — пропуск
		accumulatedRow(7, "2026-03-01T12:00:00Z", 3),
		accumulatedRow(8, "2026-04-01T00:00:00Z", 4), // после периода — мимо
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
		{"10.01.2026", 3},
		{"01.02.2026", 2},
		{"05.02.2026", 3},
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
	// from > to, нет дат, ошибка fetch — громко.
	badFrom := map[string]any{"date_from": at("2026-05-01T00:00:00Z"), "date_to": at("2026-01-01T00:00:00Z")}
	for _, tc := range []map[string]any{
		{},
		{"date_from": at("2026-01-01T00:00:00Z")},
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
