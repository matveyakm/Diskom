package print

import (
	"errors"
	"shared/report"
	"strings"
	"testing"
	"time"
)

type stubCtx struct {
	record  map[string]any
	placed  []map[string]any
	emp     []map[string]any
	jobs    []map[string]any
	depts   []map[string]any
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
	switch ref {
	case "staff_placement_ledger":
		return s.placed, nil
	case "employee_data_ledger":
		return s.emp, nil
	case "job_title_data_ledger":
		return s.jobs, nil
	case "department_data_ledger":
		return s.depts, nil
	}
	return nil, errors.New("unexpected ref " + ref)
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

// evRow — событие журнала (только тройка id; имён здесь больше нет).
func evRow(id int64, event, ddate string, emp, dep, job int64) map[string]any {
	return map[string]any{
		"id": float64(id), "event": event, "ddate": ddate,
		"id_employee": float64(emp), "id_department": float64(dep),
		"id_job_title": float64(job),
	}
}

func careerRec() map[string]any {
	return map[string]any{
		"ddate":     time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC),
		"date_from": time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		"date_to":   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
	}
}

func careerStub() *stubCtx {
	return &stubCtx{
		record: careerRec(),
		emp: []map[string]any{
			{"id": float64(1), "id_employee": float64(1), "ddate": "2026-01-05T00:00:00Z", "fio": "Иванов Иван Иванович"},
		},
		jobs: []map[string]any{
			{"id": float64(1), "id_job_title": float64(1), "ddate": "2026-01-01T00:00:00Z", "name_instrumental": "Активистом"},
			{"id": float64(2), "id_job_title": float64(2), "ddate": "2026-01-01T00:00:00Z", "name_instrumental": "Руководителем"},
		},
		depts: []map[string]any{
			{"id": float64(1), "id_department": float64(1), "ddate": "2026-01-01T00:00:00Z", "name_genitive": "Отдела"},
			{"id": float64(2), "id_department": float64(2), "ddate": "2026-01-01T00:00:00Z", "name_genitive": "Отдела"},
		},
	}
}

func careerText(t *testing.T, s *stubCtx) string {
	t.Helper()
	got, err := careerSlot(s, nil)
	if err != nil {
		t.Fatalf("career = %v", err)
	}
	str, ok := got.(string)
	if !ok {
		t.Fatalf("slot must return string, got %T", got)
	}
	return str
}

func TestCareerText(t *testing.T) {
	s := careerStub()
	s.placed = []map[string]any{
		evRow(1, "open", "2026-03-02T00:00:00Z", 1, 1, 1),
		evRow(2, "open", "2026-03-02T00:00:00Z", 1, 2, 1), // параллельная тройка
		evRow(3, "close", "2026-06-12T00:00:00Z", 1, 2, 1),
		evRow(4, "open", "2026-05-10T00:00:00Z", 1, 1, 2),
		evRow(5, "close", "2026-07-12T00:00:00Z", 1, 1, 2),
		evRow(6, "close", "2026-05-10T00:00:00Z", 1, 1, 1),
	}
	want := "Иванов Иван Иванович\n" +
		"- В период с 02.03.2026 по 10.05.2026 являлся Активистом Отдела\n" +
		"- В период с 02.03.2026 по 12.06.2026 являлся Активистом Отдела\n" +
		"- В период с 10.05.2026 по 12.07.2026 являлся Руководителем Отдела\n"
	if s := careerText(t, s); s != want {
		t.Fatalf("got:\n%q\nwant:\n%q", s, want)
	}
}

func TestCareerRenameTimeline(t *testing.T) {
	s := careerStub()
	s.placed = []map[string]any{
		evRow(1, "open", "2026-03-02T00:00:00Z", 1, 1, 1),
	}
	// Переименование 01.06: до отчёта (ddate 10.11) — новое имя везде.
	s.emp = append(s.emp, map[string]any{
		"id": float64(2), "id_employee": float64(1),
		"ddate": "2026-06-01T00:00:00Z", "fio": "Петрова Анна Ивановна",
	})
	if got := careerText(t, s); !strings.HasPrefix(got, "Петрова Анна Ивановна\n") {
		t.Fatalf("rename must win: %q", got)
	}
	// Переименование после формирования — невидимо (замороженный взгляд).
	s2 := careerStub()
	s2.record["ddate"] = time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	s2.placed = s.placed
	s2.emp = s.emp
	if got := careerText(t, s2); !strings.HasPrefix(got, "Иванов Иван Иванович\n") {
		t.Fatalf("future rename must not leak: %q", got)
	}
}

func TestCareerOpenAndClip(t *testing.T) {
	s := careerStub()
	s.placed = []map[string]any{
		evRow(1, "open", "2026-01-05T00:00:00Z", 1, 1, 1), // открыт до периода
		evRow(2, "open", "2027-02-01T00:00:00Z", 1, 1, 1), // открыт после периода
	}
	s.record = map[string]any{
		"ddate":     time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		"date_from": time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		"date_to":   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
	}
	want := "Иванов Иван Иванович\n" +
		"- В период с 01.03.2026 по настоящее время является Активистом Отдела\n"
	if got := careerText(t, s); got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestCareerLoud(t *testing.T) {
	// Сирота-close, неизвестное событие, нулевая тройка.
	bad := [][]map[string]any{
		{evRow(1, "close", "2026-03-02T00:00:00Z", 1, 1, 1)},
		{evRow(1, "moved", "2026-03-02T00:00:00Z", 1, 1, 1)},
		{evRow(1, "open", "2026-03-02T00:00:00Z", 0, 1, 1)},
	}
	for i, placed := range bad {
		s := careerStub()
		s.placed = placed
		if _, err := careerSlot(s, nil); err == nil {
			t.Fatalf("case %d: want error", i)
		}
	}
	// Нет ФИО на дату формирования — громко.
	s := careerStub()
	s.placed = []map[string]any{evRow(1, "open", "2026-03-02T00:00:00Z", 9, 1, 1)}
	if _, err := careerSlot(s, nil); err == nil {
		t.Fatal("missing fio: want error")
	}
	// Нет названия должности — громко.
	s = careerStub()
	s.placed = []map[string]any{evRow(1, "open", "2026-03-02T00:00:00Z", 1, 1, 9)}
	if _, err := careerSlot(s, nil); err == nil {
		t.Fatal("missing job name: want error")
	}
	if _, err := careerSlot(&stubCtx{record: map[string]any{
		"ddate":     time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC),
		"date_from": time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		"date_to":   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}, placed: nil}, nil); err == nil {
		t.Fatal("from>to: want error")
	}
	// Пустой регистр — валидная пустота (только шапка), не ошибка.
	got, err := careerSlot(careerStub(), nil)
	if err != nil || !strings.EqualFold(got.(string), "") {
		t.Fatalf("empty must stay empty: %q %v", got, err)
	}
}
