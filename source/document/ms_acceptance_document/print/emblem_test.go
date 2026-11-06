package print

import (
	"errors"
	"shared/report"
	"testing"
	"time"
)

// stubCtx — минимальный report.Ctx для filler-а (без сети).
type stubCtx struct {
	record  map[string]any
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
	return nil, errors.New("not needed")
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

func emblemRecord() map[string]any {
	return map[string]any{
		"ddate":        time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC),
		"id_committee": int64(1),
	}
}

func TestEmblemSlot(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 1}
	got, err := emblemSlot(&stubCtx{
		record: emblemRecord(),
		one:    map[string]any{"id": float64(9)},
		files:  []report.FileMeta{{ID: 4}},
		data:   png, mime: "image/png",
	}, nil)
	raw, ok := got.([]byte)
	if err != nil || !ok || string(raw) != string(png) {
		t.Fatalf("emblem = %v,%v", got, err)
	}
	// Нет версии / нет файлов / чужая ошибка — громко, без текста filler-а.
	for _, tc := range []stubCtx{
		{record: emblemRecord(), oneErr: errors.New("ref: 404")},
		{record: emblemRecord(), one: map[string]any{"id": 1}},
		{record: emblemRecord(), one: map[string]any{"id": 1}, files: []report.FileMeta{{ID: 1}, {ID: 2}}},
	} {
		if _, err := emblemSlot(&tc, nil); err == nil {
			t.Fatalf("want error for %+v", tc.one)
		}
	}
}
