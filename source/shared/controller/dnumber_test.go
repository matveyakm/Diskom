package restController

import "testing"

func TestBuildDnumber(t *testing.T) {
	if got := buildDnumber("НАД-", 42); got != "НАД-42" {
		t.Fatalf("got %q", got)
	}
	if got := buildDnumber("", 7); got != "7" {
		t.Fatalf("got %q", got)
	}
}

func TestDnumberID(t *testing.T) {
	tests := []struct {
		in     any
		want   int64
		wantOK bool
	}{
		{float64(42), 42, true},
		{float64(0), 0, false},
		{float64(-3), 0, false},
		{float64(1.5), 0, false},
		{"42", 42, true},
		{"007", 7, true},
		{"0", 0, false},
		{"", 0, false},
		{"abc", 0, false},
		{"4.2", 0, false},
		{nil, 0, false},
		{true, 0, false},
	}
	for _, tt := range tests {
		got, ok := dnumberID(tt.in)
		if got != tt.want || ok != tt.wantOK {
			t.Fatalf("dnumberID(%v) = (%d, %v), want (%d, %v)", tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestPlanDnumber(t *testing.T) {
	// empty + prefix + id -> generate
	id, val, ok := planDnumber(map[string]any{"id": float64(42), "dnumber": ""}, "НАД-")
	if !ok || id != 42 || val != "НАД-42" {
		t.Fatalf("got (%d, %q, %v)", id, val, ok)
	}
	// null dnumber -> generate
	if _, _, ok := planDnumber(map[string]any{"id": float64(5), "dnumber": nil}, "СН-"); !ok {
		t.Fatalf("null dnumber must be generated")
	}
	// manual number -> keep
	if _, _, ok := planDnumber(map[string]any{"id": float64(5), "dnumber": "РУЧ-1"}, "СН-"); ok {
		t.Fatalf("manual dnumber must be kept")
	}
	// no column (ledgers) -> skip
	if _, _, ok := planDnumber(map[string]any{"id": float64(5)}, "СН-"); ok {
		t.Fatalf("missing column must be skipped")
	}
	// no prefix -> skip
	if _, _, ok := planDnumber(map[string]any{"id": float64(5), "dnumber": ""}, ""); ok {
		t.Fatalf("empty prefix must be skipped")
	}
	// zero id -> skip
	if _, _, ok := planDnumber(map[string]any{"id": float64(0), "dnumber": ""}, "СН-"); ok {
		t.Fatalf("zero id must be skipped")
	}
	// non-string dnumber -> do not touch
	if _, _, ok := planDnumber(map[string]any{"id": float64(5), "dnumber": float64(9)}, "СН-"); ok {
		t.Fatalf("non-string dnumber must be skipped")
	}
}
