package print

import (
	"errors"
	"shared/report"
	"testing"
	"time"
)

func emblemRecord() map[string]any {
	return map[string]any{
		"ddate": time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC),
	}
}

func emblemParams() report.Params {
	return report.Params{"id_committee": "1"}
}

func TestEmblemSlot(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 1}
	got, err := emblemSlot(&stubCtx{
		record: emblemRecord(),
		one:    map[string]any{"id": float64(9)},
		files:  []report.FileMeta{{ID: 4}},
		data:   png, mime: "image/png",
	}, emblemParams())
	raw, ok := got.([]byte)
	if err != nil || !ok || string(raw) != string(png) {
		t.Fatalf("emblem = %v,%v", got, err)
	}
	// Нет версии / нет файлов / чужая ошибка / нет параметра комитета — громко.
	for _, tc := range []struct {
		ctx stubCtx
		p   report.Params
	}{
		{stubCtx{record: emblemRecord(), oneErr: errors.New("ref: 404")}, emblemParams()},
		{stubCtx{record: emblemRecord(), one: map[string]any{"id": 1}}, emblemParams()},
		{stubCtx{record: emblemRecord(), one: map[string]any{"id": 1}, files: []report.FileMeta{{ID: 1}, {ID: 2}}}, emblemParams()},
		{stubCtx{record: emblemRecord(), one: map[string]any{"id": 1}, files: []report.FileMeta{{ID: 1}}}, nil},
		{stubCtx{record: emblemRecord(), one: map[string]any{"id": 1}, files: []report.FileMeta{{ID: 1}}}, report.Params{}},
	} {
		if _, err := emblemSlot(&tc.ctx, tc.p); err == nil {
			t.Fatalf("want error for %+v / %+v", tc.ctx.one, tc.p)
		}
	}
}
