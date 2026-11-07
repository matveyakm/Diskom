package print

import (
	"bytes"
	"image/png"
	"testing"
	"time"
)

func chartRecord(from, to string) map[string]any {
	f, _ := time.Parse(time.RFC3339, from)
	t, _ := time.Parse(time.RFC3339, to)
	return map[string]any{"date_from": f, "date_to": t}
}

func chartRows() []map[string]any {
	return []map[string]any{
		{"id": float64(1), "ddate": "2026-01-05T00:00:00Z", "balance": float64(1)},
		{"id": float64(2), "ddate": "2026-02-01T12:00:00Z", "balance": float64(3)},
		{"id": float64(3), "ddate": "2026-02-01T18:00:00Z", "balance": float64(2)},
	}
}

func mustChart(t *testing.T, rec map[string]any, rows []map[string]any) []byte {
	t.Helper()
	got, err := chartSlot(&stubCtx{record: rec, rows: rows}, nil)
	if err != nil {
		t.Fatalf("chart: %v", err)
	}
	raw, ok := got.([]byte)
	if !ok || len(raw) == 0 {
		t.Fatalf("chart: want PNG bytes, got %T len %d", got, len(raw))
	}
	return raw
}

func TestChartSlot(t *testing.T) {
	raw := mustChart(t, chartRecord("2026-01-01T00:00:00Z", "2026-03-31T00:00:00Z"), chartRows())
	cfg, err := png.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("not a PNG: %v", err)
	}
	if cfg.Width != 1024 || cfg.Height != 400 {
		t.Fatalf("want 1024x400, got %dx%d", cfg.Width, cfg.Height)
	}
	// Детерминированность: два рендера побайтово равны.
	again := mustChart(t, chartRecord("2026-01-01T00:00:00Z", "2026-03-31T00:00:00Z"), chartRows())
	if !bytes.Equal(raw, again) {
		t.Fatal("chart render is not deterministic")
	}
	// Пустая история валидна (как пустая таблица): плоская нулевая линия.
	empty := mustChart(t, chartRecord("2026-01-01T00:00:00Z", "2026-03-31T00:00:00Z"), nil)
	if _, err := png.Decode(bytes.NewReader(empty)); err != nil {
		t.Fatalf("empty history: not a PNG: %v", err)
	}
	// Один день — без паники.
	one := mustChart(t, chartRecord("2026-02-01T00:00:00Z", "2026-02-01T00:00:00Z"), chartRows())
	if _, err := png.Decode(bytes.NewReader(one)); err != nil {
		t.Fatalf("single day: not a PNG: %v", err)
	}
	// from > to — громко.
	if _, err := chartSlot(&stubCtx{record: chartRecord("2026-05-01T00:00:00Z", "2026-01-01T00:00:00Z")}, nil); err == nil {
		t.Fatal("want error for from > to")
	}
}
