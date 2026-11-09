package restController

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestParseExpand(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    []expandSpec
		wantErr bool
	}{
		{"empty", "", nil, false},
		{"spaces", "   ", nil, false},
		{
			"single",
			"id_department:department_directory",
			[]expandSpec{{column: "id_department", endpoint: "department_directory"}},
			false,
		},
		{
			"multi with spaces",
			" id_employee:employee_directory , id_job_title:job_title_directory ",
			[]expandSpec{
				{column: "id_employee", endpoint: "employee_directory"},
				{column: "id_job_title", endpoint: "job_title_directory"},
			},
			false,
		},
		{"no colon", "id_employee", nil, true},
		{"empty endpoint", "id_employee:", nil, true},
		{"bad column chars", "id-employee:employee_directory", nil, true},
		{"uppercase endpoint", "id_employee:Employee", nil, true},
		{"path traversal", "id:x/../y", nil, true},
		{"url scheme", "id:http://evil", nil, true},
		{"too many", "a:b,c:d,e:f,g:h,i:j,k:l", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseExpand(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseExpand(%q) err = %v, wantErr = %v", tt.raw, err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseExpand(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestParseIDs(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    []int64
		wantErr bool
	}{
		{"empty", "", nil, false},
		{"single", "5", []int64{5}, false},
		{"multi dedupe", "3, 1,3,2", []int64{3, 1, 2}, false},
		{"zero", "0", nil, true},
		{"negative", "-1", nil, true},
		{"float", "1.5", nil, true},
		{"alpha", "abc", nil, true},
		{"empty token", "1,,2", nil, true},
		{"injection", "1 OR 1=1", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIDs(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseIDs(%q) err = %v, wantErr = %v", tt.raw, err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseIDs(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestRefIDKey(t *testing.T) {
	tests := []struct {
		in     any
		want   string
		wantOK bool
	}{
		{float64(7), "7", true},
		{float64(0), "", false},
		{float64(-2), "", false},
		{float64(1.5), "", false},
		{"42", "42", true},
		{"007", "7", true},
		{"0", "", false},
		{"", "", false},
		{"  9  ", "9", true},
		{nil, "", false},
		{true, "", false},
	}
	for _, tt := range tests {
		got, ok := refIDKey(tt.in)
		if got != tt.want || ok != tt.wantOK {
			t.Fatalf("refIDKey(%v) = (%q, %v), want (%q, %v)", tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestRefDisplayName(t *testing.T) {
	if got, ok := refDisplayName(map[string]any{"id": 1.0, "name": "Кадры"}); !ok || got != "Кадры" {
		t.Fatalf("want Кадры, got %q %v", got, ok)
	}
	for _, row := range []map[string]any{
		{"id": 1.0},
		{"id": 1.0, "name": ""},
		{"id": 1.0, "name": "   "},
		{"id": 1.0, "name": 42},
	} {
		if _, ok := refDisplayName(row); ok {
			t.Fatalf("want no name for %v", row)
		}
	}
}

func TestCollectSpecIDs(t *testing.T) {
	rows := []map[string]any{
		{"id_department": float64(3)},
		{"id_department": float64(3)},
		{"id_department": float64(0)},
		{"id_department": nil},
		{"other": float64(9)},
	}
	if got := collectSpecIDs(rows, "id_department"); !reflect.DeepEqual(got, []int64{3}) {
		t.Fatalf("got %v", got)
	}
}

func TestEnrichRowsConcurrent(t *testing.T) {
	// Два справочника на httptest-серверах; enrichRows ходит в них
	// параллельно — раньше это роняло процесс (concurrent map writes).
	// Гонять с -race.
	emps := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[{"id":1,"name":"Иванов"},{"id":2,"name":"Петров"}]}`)
	}))
	defer emps.Close()
	depts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[{"id":3,"name":"Отдел"},{"id":4,"name":"Цех"}]}`)
	}))
	defer depts.Close()
	t.Setenv("REF_EMPLOYEE_DIRECTORY", emps.URL)
	t.Setenv("REF_DEPARTMENT_DIRECTORY", depts.URL)
	rows := make([]map[string]any, 0, 50)
	for i := 0; i < 50; i++ {
		rows = append(rows, map[string]any{
			"id_employee":   float64(i%2 + 1),
			"id_department": float64(i%2 + 3),
		})
	}
	specs := []expandSpec{
		{column: "id_employee", endpoint: "employee_directory"},
		{column: "id_department", endpoint: "department_directory"},
	}
	enrichRows(&http.Client{Timeout: 5 * time.Second}, rows, specs)
	for i, row := range rows {
		if row["id_employee_name"] != []string{"Иванов", "Петров"}[i%2] {
			t.Fatalf("row %d: %+v", i, row)
		}
		if row["id_department_name"] != []string{"Отдел", "Цех"}[i%2] {
			t.Fatalf("row %d: %+v", i, row)
		}
	}
}
