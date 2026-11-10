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
	if ref != "staff_placement_ledger" {
		return nil, errors.New("unexpected ref " + ref)
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

// evRow — строка журнала событий: снапшоты обязательны (иначе громко).
func evRow(id int64, event, ddate string, emp, dep, job int64) map[string]any {
	return map[string]any{
		"id": float64(id), "event": event, "ddate": ddate,
		"id_employee": float64(emp), "employee_name": "Иванов Иван Иванович",
		"id_department": float64(dep), "department_name": "Отдел",
		"department_genitive": "Отдела",
		"id_job_title":        float64(job), "job_title_name": "Активист",
		"job_title_instrumental": "Активистом",
	}
}

func careerRec() map[string]any {
	return map[string]any{
		"date_from": time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		"date_to":   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
	}
}

func TestCareerPairing(t *testing.T) {
	rows := []map[string]any{
		evRow(1, "open", "2026-03-02T00:00:00Z", 1, 1, 1),
		evRow(2, "open", "2026-03-02T00:00:00Z", 1, 2, 1), // параллельная тройка
		evRow(3, "close", "2026-06-12T00:00:00Z", 1, 2, 1),
		evRow(4, "open", "2026-05-10T00:00:00Z", 1, 1, 2),
		evRow(5, "close", "2026-07-12T00:00:00Z", 1, 1, 2),
		evRow(6, "close", "2026-05-10T00:00:00Z", 1, 1, 1),
	}
	got, err := careerSlot(&stubCtx{record: careerRec(), rows: rows}, nil)
	if err != nil {
		t.Fatalf("career = %v", err)
	}
	out := got.([]map[string]any)
	if len(out) != 3 {
		t.Fatalf("want 3 intervals, got %d: %v", len(out), out)
	}
	// 1|1|1: 02.03–10.05; 1|2|1: 02.03–12.06; 1|1|2: 10.05–12.07.
	if out[0]["from"] != "02.03.2026" || out[0]["to"] != "10.05.2026" {
		t.Fatalf("first interval = %v", out[0])
	}
	if out[1]["from"] != "02.03.2026" || out[1]["to"] != "12.06.2026" {
		t.Fatalf("second interval = %v", out[1])
	}
	if out[2]["from"] != "10.05.2026" || out[2]["to"] != "12.07.2026" {
		t.Fatalf("third interval = %v", out[2])
	}
}

func TestCareerOpenAndClip(t *testing.T) {
	rows := []map[string]any{
		evRow(1, "open", "2026-01-05T00:00:00Z", 1, 1, 1), // открыт до периода
		evRow(2, "open", "2027-02-01T00:00:00Z", 1, 1, 1), // открыт после периода
	}
	rec := map[string]any{
		"date_from": time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		"date_to":   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
	}
	got, err := careerSlot(&stubCtx{record: rec, rows: rows}, nil)
	if err != nil {
		t.Fatalf("career = %v", err)
	}
	out := got.([]map[string]any)
	if len(out) != 1 {
		t.Fatalf("want 1 interval (second opens after period), got %v", out)
	}
	if out[0]["from"] != "01.03.2026" || out[0]["to"] != "по настоящее время" {
		t.Fatalf("clipped = %v", out[0])
	}
}

func TestCareerLoud(t *testing.T) {
	rec := careerRec()
	// Сирота-close, неизвестное событие, пустой снапшот, нулевой id, from>to.
	bad := [][]map[string]any{
		{evRow(1, "close", "2026-03-02T00:00:00Z", 1, 1, 1)},
		{evRow(1, "moved", "2026-03-02T00:00:00Z", 1, 1, 1)},
		{func() map[string]any {
			r := evRow(1, "open", "2026-03-02T00:00:00Z", 1, 1, 1)
			r["job_title_instrumental"] = ""
			return r
		}()},
		{func() map[string]any {
			r := evRow(1, "open", "2026-03-02T00:00:00Z", 1, 1, 1)
			r["id_department"] = float64(0)
			return r
		}()},
	}
	for i, rows := range bad {
		if _, err := careerSlot(&stubCtx{record: rec, rows: rows}, nil); err == nil {
			t.Fatalf("case %d: want error", i)
		}
	}
	if _, err := careerSlot(&stubCtx{record: map[string]any{
		"date_from": time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		"date_to":   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}, rows: nil}, nil); err == nil {
		t.Fatal("from>to: want error")
	}
}
