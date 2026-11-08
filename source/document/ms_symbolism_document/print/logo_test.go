package print

import (
	"errors"
	"shared/report"
	"testing"
)

type stubCtx struct {
	record  map[string]any
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
	return nil, errors.New("not needed")
}
func (s *stubCtx) FilesOf(ref string, docID int64) ([]report.FileMeta, error) {
	return s.files, s.filesEr
}
func (s *stubCtx) FileBytes(ref string, fileID, docID int64) ([]byte, string, error) {
	return s.data, s.mime, s.dataErr
}

func TestLogoSlot(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 1}
	got, err := logoSlot(&stubCtx{
		record: map[string]any{"id": float64(7)},
		files:  []report.FileMeta{{ID: 4}},
		data:   png, mime: "image/png",
	}, nil)
	raw, ok := got.([]byte)
	if err != nil || !ok || string(raw) != string(png) {
		t.Fatalf("logo = %v,%v", got, err)
	}
	// Нет id / ошибка списка / 0 файлов / 2 файла / не-картинка — громко.
	for i, tc := range []stubCtx{
		{record: map[string]any{}},
		{record: map[string]any{"id": float64(7)}, filesEr: errors.New("ref: 500")},
		{record: map[string]any{"id": float64(7)}},
		{record: map[string]any{"id": float64(7)}, files: []report.FileMeta{{ID: 1}, {ID: 2}}},
		{record: map[string]any{"id": float64(7)}, files: []report.FileMeta{{ID: 1}}, data: []byte("%PDF"), mime: "application/pdf"},
	} {
		if _, err := logoSlot(&tc, nil); err == nil {
			t.Fatalf("case %d: want error", i)
		}
	}
}
