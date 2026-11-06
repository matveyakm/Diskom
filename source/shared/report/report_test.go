package report

import (
	"errors"
	"testing"
	"time"
)

func TestStr(t *testing.T) {
	at, _ := time.Parse(time.RFC3339, "2026-11-06T00:00:00Z")
	for _, tc := range []struct {
		in   any
		want string
	}{
		{nil, ""},
		{"  x  ", "x"},
		{float64(2), "2"},
		{float64(2.5), "2.5"},
		{int64(7), "7"},
		{int16(3), "3"},
		{true, "true"},
		{at, "2026-11-06T00:00:00Z"},
	} {
		if got := Str(tc.in); got != tc.want {
			t.Fatalf("Str(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestInt(t *testing.T) {
	for _, tc := range []struct {
		in   any
		want int64
		ok   bool
	}{
		{int64(7), 7, true},
		{float64(7), 7, true},
		{float64(7.5), 0, false},
		{"42", 42, true},
		{" 42 ", 42, true},
		{"x", 0, false},
		{0, 0, false},
		{-3, 0, false},
		{nil, 0, false},
		{int16(5), 5, true},
	} {
		got, ok := Int(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("Int(%v) = (%d,%v), want (%d,%v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

// stubCtx — минимальный Ctx для композитов (без сети).
type stubCtx struct {
	record   map[string]any
	one      map[string]any
	oneErr   error
	files    []FileMeta
	filesErr error
	data     []byte
	mime     string
	dataErr  error
}

func (s *stubCtx) Record() map[string]any { return s.record }
func (s *stubCtx) Fetch(ref string, q map[string]string) ([]map[string]any, error) {
	return nil, errors.New("not needed")
}
func (s *stubCtx) FetchOne(ref string, q map[string]string) (map[string]any, error) {
	return s.one, s.oneErr
}
func (s *stubCtx) FilesOf(ref string, docID int64) ([]FileMeta, error) {
	return s.files, s.filesErr
}
func (s *stubCtx) FileBytes(ref string, fileID, docID int64) ([]byte, string, error) {
	return s.data, s.mime, s.dataErr
}

func TestActualFileBytes(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 1}
	// Счастливый путь.
	ok := &stubCtx{
		one:   map[string]any{"id": float64(9)},
		files: []FileMeta{{ID: 4}},
		data:  png, mime: "image/png",
	}
	got, err := ActualFileBytes(ok, "symbolism_document", "effective_date", "2026-11-06", map[string]string{"id_committee": "1"})
	if err != nil || string(got) != string(png) {
		t.Fatalf("happy = %v,%v", got, err)
	}
	// Нет версии — громко.
	_, err = ActualFileBytes(&stubCtx{oneErr: errors.New("ref: 404")}, "r", "f", "d", nil)
	if err == nil {
		t.Fatal("want error on missing version")
	}
	// Файлов 0 и 2 — громко.
	for _, n := range []int{0, 2} {
		files := make([]FileMeta, n)
		_, err := ActualFileBytes(&stubCtx{one: map[string]any{"id": 1}, files: files}, "r", "f", "d", nil)
		if err == nil {
			t.Fatalf("want error on %d files", n)
		}
	}
	// PDF — громко.
	_, err = ActualFileBytes(&stubCtx{
		one: map[string]any{"id": 1}, files: []FileMeta{{ID: 4}},
		data: []byte("%PDF-"), mime: "application/pdf",
	}, "r", "f", "d", nil)
	if err == nil {
		t.Fatal("want error on PDF")
	}
}
