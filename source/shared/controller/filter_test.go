package restController

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitFilterKey(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		want   filterOp
		wantOK bool
	}{
		{"exact", "status", filterOp{column: "status", op: "eq"}, true},
		{"ne", "status__ne", filterOp{column: "status", op: "ne"}, true},
		{"gt", "id__gt", filterOp{column: "id", op: "gt"}, true},
		{"gte", "ddate__gte", filterOp{column: "ddate", op: "gte"}, true},
		{"lt", "id__lt", filterOp{column: "id", op: "lt"}, true},
		{"lte", "ddate__lte", filterOp{column: "ddate", op: "lte"}, true},
		{"like", "note__like", filterOp{column: "note", op: "like"}, true},
		{"in", "status__in", filterOp{column: "status", op: "in"}, true},
		{"unknown suffix ignored", "status__foo", filterOp{}, false},
		{"reserved skipped", "expand", filterOp{}, false},
		{"reserved skipped 2", "sort", filterOp{}, false},
		{"reserved skipped 3", "q_fields", filterOp{}, false},
		{"bad ident ignored", "sta-tus", filterOp{}, false},
		{"injection ignored", "id; DROP TABLE x", filterOp{}, false},
		{"empty ignored", "", filterOp{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := splitFilterKey(tt.key)
			if ok != tt.wantOK {
				t.Fatalf("splitFilterKey(%q) ok = %v, want %v", tt.key, ok, tt.wantOK)
			}
			if ok && !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("splitFilterKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestEscapeLike(t *testing.T) {
	tests := []struct{ in, want string }{
		{"plain", "plain"},
		{"100%", `100\%`},
		{"a_b", `a\_b`},
		{`a\b`, `a\\b`},
		{"привет %_", `привет \%\_`},
		{"", ""},
	}
	for _, tt := range tests {
		if got := escapeLike(tt.in); got != tt.want {
			t.Errorf("escapeLike(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseInValues(t *testing.T) {
	vals, empty, err := parseInValues("1, 2,3")
	if err != nil || empty || !reflect.DeepEqual(vals, []string{"1", "2", "3"}) {
		t.Fatalf("parseInValues normal = %v,%v,%v", vals, empty, err)
	}
	vals, empty, err = parseInValues(" , ,")
	if err != nil || !empty || len(vals) != 0 {
		t.Fatalf("parseInValues blank = %v,%v,%v", vals, empty, err)
	}
	if _, _, err = parseInValues(strings.Repeat("a,", 201) + "b"); err == nil {
		t.Fatal("parseInValues over cap: want error")
	}
}

func paramsOf(m map[string]string) map[string][]string {
	out := make(map[string][]string, len(m))
	for k, v := range m {
		out[k] = []string{v}
	}
	return out
}

func TestParseSortOrder(t *testing.T) {
	def, err := parseSortOrder(paramsOf(map[string]string{}))
	if err != nil || def != "id DESC" {
		t.Fatalf("default sort = %q,%v", def, err)
	}
	got, err := parseSortOrder(paramsOf(map[string]string{"sort": "ddate", "order": "desc"}))
	if err != nil || got != "ddate DESC, id DESC" {
		t.Fatalf("sort ddate desc = %q,%v", got, err)
	}
	got, err = parseSortOrder(paramsOf(map[string]string{"sort": "note"}))
	if err != nil || got != "note ASC, id ASC" {
		t.Fatalf("sort default order = %q,%v", got, err)
	}
	got, err = parseSortOrder(paramsOf(map[string]string{"sort": "id", "order": "asc"}))
	if err != nil || got != "id ASC" {
		t.Fatalf("sort id needs no tiebreak = %q,%v", got, err)
	}
	for _, bad := range []map[string]string{
		{"sort": "id; DROP TABLE t"},
		{"sort": "id", "order": "sideways"},
	} {
		if _, err := parseSortOrder(paramsOf(bad)); err == nil {
			t.Fatalf("parseSortOrder(%v): want error", bad)
		}
	}
}

func TestParseSearch(t *testing.T) {
	_, _, ok, err := parseSearch(paramsOf(map[string]string{}))
	if err != nil || ok {
		t.Fatalf("no q = %v,%v", ok, err)
	}
	fields, pattern, ok, err := parseSearch(paramsOf(map[string]string{
		"q": "по-12%", "q_fields": "dnumber,note",
	}))
	if err != nil || !ok || pattern != `%по-12\%%` ||
		!reflect.DeepEqual(fields, []string{"dnumber", "note"}) {
		t.Fatalf("search = %v,%q,%v,%v", fields, pattern, ok, err)
	}
	for _, bad := range []map[string]string{
		{"q": "x"},                    // no q_fields
		{"q": "x", "q_fields": "a-b"}, // bad ident
		{"q": strings.Repeat("x", 101), "q_fields": "a"}, // too long
	} {
		if _, _, _, err := parseSearch(paramsOf(bad)); err == nil {
			t.Fatalf("parseSearch(%v): want error", bad)
		}
	}
}

func TestParseDistinct(t *testing.T) {
	_, _, _, ok, err := parseDistinct(paramsOf(map[string]string{}))
	if err != nil || ok {
		t.Fatalf("no distinct = %v,%v", ok, err)
	}
	col, like, limit, ok, err := parseDistinct(paramsOf(map[string]string{
		"distinct": "status", "distinct_like": "сох",
	}))
	if err != nil || !ok || col != "status" || like != "сох" || limit != defaultDistinctLimit {
		t.Fatalf("distinct = %q,%q,%d,%v,%v", col, like, limit, ok, err)
	}
	_, _, limit, _, err = parseDistinct(paramsOf(map[string]string{
		"distinct": "status", "distinct_limit": "50",
	}))
	if err != nil || limit != 50 {
		t.Fatalf("distinct_limit = %d,%v", limit, err)
	}
	for _, bad := range []map[string]string{
		{"distinct": "a-b"},
		{"distinct": "a", "distinct_limit": "0"},
		{"distinct": "a", "distinct_limit": "501"},
		{"distinct": "a", "distinct_limit": "lots"},
	} {
		if _, _, _, _, err := parseDistinct(paramsOf(bad)); err == nil {
			t.Fatalf("parseDistinct(%v): want error", bad)
		}
	}
}

func TestStripOwnColumn(t *testing.T) {
	params := paramsOf(map[string]string{
		"status":     "1",
		"status__in": "1,2",
		"note__like": "x",
		"sort":       "id",
		"expand":     "a:b",
	})
	got := stripOwnColumn(params, "status")
	if _, found := got["status"]; found {
		t.Fatal("stripOwnColumn: own exact key must go")
	}
	if _, found := got["status__in"]; found {
		t.Fatal("stripOwnColumn: own suffixed key must go")
	}
	for _, keep := range []string{"note__like", "sort", "expand"} {
		if _, found := got[keep]; !found {
			t.Fatalf("stripOwnColumn: %q must stay", keep)
		}
	}
}

func TestHasEmptyIn(t *testing.T) {
	yes, err := hasEmptyIn(paramsOf(map[string]string{"status__in": " , "}))
	if err != nil || !yes {
		t.Fatalf("blank in = %v,%v", yes, err)
	}
	yes, err = hasEmptyIn(paramsOf(map[string]string{"status__in": "1,2"}))
	if err != nil || yes {
		t.Fatalf("filled in = %v,%v", yes, err)
	}
	yes, err = hasEmptyIn(paramsOf(map[string]string{"status": "1"}))
	if err != nil || yes {
		t.Fatalf("no in = %v,%v", yes, err)
	}
}
