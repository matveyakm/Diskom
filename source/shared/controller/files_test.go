package restController

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFileMimeByMagic(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  []byte
		mime string
		ok   bool
	}{
		{"png", []byte{0x89, 'P', 'N', 'G', 1, 2, 3}, "image/png", true},
		{"jpg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, "image/jpeg", true},
		{"pdf", []byte("%PDF-1.7..."), "application/pdf", true},
		{"empty", nil, "", false},
		{"short", []byte{1, 2, 3}, "", false},
		{"zip", []byte{'P', 'K', 3, 4}, "", false},
		{"gif", []byte("GIF89a..."), "", false},
	} {
		mime, ok := fileMimeByMagic(tc.raw)
		if mime != tc.mime || ok != tc.ok {
			t.Fatalf("%s = %q,%v, want %q,%v", tc.name, mime, ok, tc.mime, tc.ok)
		}
	}
}

func TestSanitizeFileName(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"logo.png", "logo.png"},
		{"  scan 1.pdf  ", "scan 1.pdf"},
		{"/tmp/../x.png", "x.png"},
		{`..\\..\\e.exe`, "e.exe"},
		{"", ""},
	} {
		if got := sanitizeFileName(tc.in); got != tc.want {
			t.Fatalf("sanitize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFileLimitsEnv(t *testing.T) {
	// Дефолты без env.
	t.Setenv("FILE_MAX_MB", "")
	t.Setenv("FILE_ORPHAN_TTL_HOURS", "")
	t.Setenv("FILE_GC_INTERVAL_HOURS", "")
	if got := fileMaxBytes(); got != defaultFileMaxMB<<20 {
		t.Fatalf("default max = %d", got)
	}
	if got := fileOrphanTTL(); got != defaultFileOrphanTTL {
		t.Fatalf("default ttl = %v", got)
	}
	if got := fileGCInterval(); got != defaultFileGCInterval {
		t.Fatalf("default interval = %v", got)
	}
	// Валидный env применяется.
	t.Setenv("FILE_MAX_MB", "10")
	t.Setenv("FILE_ORPHAN_TTL_HOURS", "24")
	t.Setenv("FILE_GC_INTERVAL_HOURS", "1")
	if got := fileMaxBytes(); got != 10<<20 {
		t.Fatalf("env max = %d", got)
	}
	if got := fileOrphanTTL(); got != 24*time.Hour {
		t.Fatalf("env ttl = %v", got)
	}
	if got := fileGCInterval(); got != time.Hour {
		t.Fatalf("env interval = %v", got)
	}
	// Мусор в env — молча дефолт (fail-open на своих настройках).
	t.Setenv("FILE_MAX_MB", "много")
	t.Setenv("FILE_ORPHAN_TTL_HOURS", "0")
	t.Setenv("FILE_GC_INTERVAL_HOURS", "-5")
	if got := fileMaxBytes(); got != defaultFileMaxMB<<20 {
		t.Fatalf("bad env max = %d", got)
	}
	if got := fileOrphanTTL(); got != defaultFileOrphanTTL {
		t.Fatalf("bad env ttl = %v", got)
	}
	if got := fileGCInterval(); got != defaultFileGCInterval {
		t.Fatalf("bad env interval = %v", got)
	}
	// Таблица files — одно имя везде (контракт "везде одинаково").
	if got := (FileObject{}).TableName(); got != "files" {
		t.Fatalf("files table = %q", got)
	}
}

func TestHasDeletePredicate(t *testing.T) {
	for _, tc := range []struct {
		name   string
		params map[string][]string
		want   bool
	}{
		{"exact", map[string][]string{"id_document": {"94"}}, true},
		{"suffix", map[string][]string{"id__in": {"1,2"}}, true},
		{"empty query", map[string][]string{}, false},
		{"only reserved", map[string][]string{"sort": {"id"}, "order": {"asc"}}, false},
		{"only id", map[string][]string{"id": {"5"}}, false}, // id идёт отдельным путём
		{"garbage ignored", map[string][]string{"a-b": {"x"}}, false},
	} {
		if got := hasDeletePredicate(tc.params); got != tc.want {
			t.Fatalf("%s = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestFileMetaNoBytes(t *testing.T) {
	// Метаданные сериализуются без байтов (байты — только ?content=).
	raw, err := json.Marshal(fileMeta{Id: 1, FileName: "a.png", Mime: "image/png", Size: 3})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "data") || strings.Contains(string(raw), "Data") {
		t.Fatalf("fileMeta leaks bytes: %s", raw)
	}
}
