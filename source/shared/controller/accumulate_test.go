package restController

import (
	"testing"
)

func TestParseAccumulateSpec(t *testing.T) {
	ok, good := parseAccumulateSpec("balance:delta")
	if !good || ok.balance != "balance" || ok.delta != "delta" || ok.order != "id" {
		t.Fatalf("two-part spec: got %+v, %v", ok, good)
	}
	three, good := parseAccumulateSpec("balance:delta:ddate")
	if !good || three.order != "ddate" {
		t.Fatalf("three-part spec: got %+v, %v", three, good)
	}
	for _, bad := range []string{
		"", "balance", "a:b:c:d", "balance:balance",
		"balance:1delta", "bal ance:delta", "balance:delta;DROP",
		":delta", "balance:",
	} {
		if _, good := parseAccumulateSpec(bad); good {
			t.Fatalf("spec %q accepted, want reject", bad)
		}
	}
	// Пробелы по краям — терпим (env пишут руками).
	sp, good := parseAccumulateSpec("  balance : delta  ")
	if !good || sp.balance != "balance" || sp.delta != "delta" {
		t.Fatalf("spaced spec: got %+v, %v", sp, good)
	}
}

func TestExtractAccumulateDelta(t *testing.T) {
	good := map[string]any{
		"int float":    float64(1),
		"neg float":    float64(-1),
		"zero":         float64(0),
		"int":          1,
		"int64":        int64(-2),
		"string":       "3",
		"spaced str":   "  -4 ",
		"big integral": float64(9007199254740992), // 2^53, точно представимо
	}
	for name, v := range good {
		if _, err := extractAccumulateDelta(map[string]any{"delta": v}, "delta"); err != nil {
			t.Fatalf("delta %s (%v) rejected: %v", name, v, err)
		}
	}
	if n, err := extractAccumulateDelta(map[string]any{"delta": float64(-1)}, "delta"); err != nil || n != -1 {
		t.Fatalf("delta -1: got %d, %v", n, err)
	}
	bad := map[string]any{
		"missing":   nil, // ключа нет вообще — ниже отдельным кейсом
		"nil":       nil,
		"fraction":  float64(1.5),
		"bool":      true,
		"text":      "abc",
		"empty str": "",
		"object":    map[string]any{},
	}
	for name, v := range bad {
		row := map[string]any{}
		if name != "missing" {
			row["delta"] = v
		}
		if _, err := extractAccumulateDelta(row, "delta"); err == nil {
			t.Fatalf("delta %s (%v) accepted, want reject", name, v)
		}
	}
	// Чужая колонка мимо: дельта лежит в другом ключе.
	if _, err := extractAccumulateDelta(map[string]any{"balance": float64(5)}, "delta"); err == nil {
		t.Fatal("wrong column accepted, want reject")
	}
}

func TestAccumulateSpecNoHardcode(t *testing.T) {
	// Механизм обязан работать с любыми именами: в коде нет balance/delta.
	spec, good := parseAccumulateSpec("rest_sum:shift:posted_at")
	if !good || spec.balance != "rest_sum" || spec.delta != "shift" || spec.order != "posted_at" {
		t.Fatalf("custom names: got %+v, %v", spec, good)
	}
	if _, good := parseAccumulateSpec("select:delta"); !good {
		t.Fatal("reserved-looking but valid column name rejected")
	}
}
