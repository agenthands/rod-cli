package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ===================== PathExists (file.go) =====================

func TestPathExists_Existing(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(f, []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}

	ok, err := PathExists(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected PathExists to be true for existing file")
	}
}

func TestPathExists_ExistingDir(t *testing.T) {
	dir := t.TempDir()
	ok, err := PathExists(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected PathExists to be true for existing dir")
	}
}

func TestPathExists_Missing(t *testing.T) {
	dir := t.TempDir()
	ok, err := PathExists(filepath.Join(dir, "nope-does-not-exist"))
	if err != nil {
		t.Fatalf("expected nil error for missing path, got %v", err)
	}
	if ok {
		t.Fatal("expected PathExists to be false for missing path")
	}
}

func TestPathExists_StatError(t *testing.T) {
	// A path whose parent is a file (not a dir) yields a non-IsNotExist error
	// (ENOTDIR) on os.Stat, exercising the trailing error return.
	dir := t.TempDir()
	f := filepath.Join(dir, "afile")
	if err := os.WriteFile(f, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	// Treat the file as a directory component.
	_, err := PathExists(filepath.Join(f, "child"))
	if err == nil {
		t.Fatal("expected an error when traversing through a file as a directory")
	}
}

// ===================== FileName (file.go) =====================

func TestFileName_WithFile(t *testing.T) {
	if got := FileName("/a/b/c/file.txt"); got != "file.txt" {
		t.Fatalf("expected file.txt, got %q", got)
	}
}

func TestFileName_BareName(t *testing.T) {
	if got := FileName("solo.go"); got != "solo.go" {
		t.Fatalf("expected solo.go, got %q", got)
	}
}

func TestFileName_TrailingSlash(t *testing.T) {
	// A trailing slash means filepath.Split returns an empty file component.
	if got := FileName("/a/b/c/"); got != "" {
		t.Fatalf("expected empty string for dir path, got %q", got)
	}
}

// ===================== RandomString (str.go) =====================

func TestRandomString_Length(t *testing.T) {
	for _, n := range []int{0, 1, 5, 32} {
		s := RandomString(n)
		if len(s) != n {
			t.Fatalf("RandomString(%d) length = %d", n, len(s))
		}
	}
}

func TestRandomString_AllLetters(t *testing.T) {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	s := RandomString(200)
	for _, r := range s {
		if !strings.ContainsRune(letters, r) {
			t.Fatalf("RandomString produced unexpected rune %q", r)
		}
	}
}

// ===================== ExecuteTemple (str.go) =====================

func TestExecuteTemple_Success(t *testing.T) {
	out, err := ExecuteTemple("Hello {{.Name}}!", map[string]string{"Name": "World"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "Hello World!" {
		t.Fatalf("expected 'Hello World!', got %q", out)
	}
}

func TestExecuteTemple_ParseError(t *testing.T) {
	_, err := ExecuteTemple("{{.Name", nil)
	if err == nil {
		t.Fatal("expected parse error for malformed template")
	}
}

func TestExecuteTemple_ExecuteError(t *testing.T) {
	// Accessing a field on a non-struct/non-map value triggers an
	// execution-time error (can't evaluate field on type int).
	_, err := ExecuteTemple("{{.Field}}", 42)
	if err == nil {
		t.Fatal("expected execution error for field access on int")
	}
}

// ===================== IsHttp (net.go) =====================

func TestIsHttp(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"  https://trimmed.com", true},
		{"\thttp://tab.com", true},
		{"ftp://example.com", false},
		{"example.com", false},
		{"", false},
		{"httpsfoo", false},
	}
	for _, c := range cases {
		if got := IsHttp(c.in); got != c.want {
			t.Fatalf("IsHttp(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// ===================== GetChinaZoneTime (time.go) =====================

func TestGetChinaZoneTime_ValidRFC3339(t *testing.T) {
	// 2023-01-01T00:00:00Z in UTC -> CST (+8) is 2023-01-01 08:00:00
	got := GetChinaZoneTime("2023-01-01T00:00:00Z")
	if got != "2023-01-01 08:00:00" {
		t.Fatalf("expected 2023-01-01 08:00:00, got %q", got)
	}
}

func TestGetChinaZoneTime_InvalidString(t *testing.T) {
	// Parse error -> zero time, formatted in CST. Must not panic and returns
	// a non-empty, formatted string.
	got := GetChinaZoneTime("not-a-time")
	if got == "" {
		t.Fatal("expected non-empty formatted string for invalid input")
	}
	// Zero time year is 0001.
	if !strings.HasPrefix(got, "0001-01-01") {
		t.Fatalf("expected zero-time formatting, got %q", got)
	}
}

func TestGetChinaZoneTime_Empty(t *testing.T) {
	got := GetChinaZoneTime("")
	if !strings.HasPrefix(got, "0001-01-01") {
		t.Fatalf("expected zero-time formatting for empty input, got %q", got)
	}
}

// ===================== Time format constants =====================

func TestTimeFormatConstants(t *testing.T) {
	if DefaultTimeFormat != "2006/01/02 15:04:05" {
		t.Fatalf("unexpected DefaultTimeFormat %q", DefaultTimeFormat)
	}
	if DefaultDateFormat != "2006-01-02" {
		t.Fatalf("unexpected DefaultDateFormat %q", DefaultDateFormat)
	}
}
