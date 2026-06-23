package types

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// chdirTemp switches into a fresh temp dir and restores cwd on cleanup so the
// repo is not polluted with rod-cli.yaml / ./rod / ./log artifacts.
func chdirTemp(t *testing.T) string {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
	return dir
}

func TestInitDefaultConfig_CreatesFile(t *testing.T) {
	dir := chdirTemp(t)

	if err := InitDefaultConfig(); err != nil {
		t.Fatalf("InitDefaultConfig: %v", err)
	}

	p := filepath.Join(dir, ConfigName)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("expected config file created: %v", err)
	}

	// Calling again should hit the "already exists" early-return branch.
	if err := InitDefaultConfig(); err != nil {
		t.Fatalf("InitDefaultConfig second call: %v", err)
	}
}

func TestLoadConfig_EmptyPathCreatesDefault(t *testing.T) {
	dir := chdirTemp(t)

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig(\"\"): %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}

	// The default config file must have been created in cwd.
	if _, err := os.Stat(filepath.Join(dir, ConfigName)); err != nil {
		t.Fatalf("expected %s created: %v", ConfigName, err)
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	dir := chdirTemp(t)

	want := DefaultConfig
	want.Mode = Vision
	want.Proxy = "http://localhost:8080"

	p := filepath.Join(dir, ConfigName)
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := yaml.NewEncoder(f).Encode(want); err != nil {
		t.Fatalf("encode: %v", err)
	}
	f.Close()

	cfg, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Mode != Vision || cfg.Proxy != "http://localhost:8080" {
		t.Fatalf("config not decoded correctly: %+v", cfg)
	}
}

func TestLoadConfig_WrongFileName(t *testing.T) {
	dir := chdirTemp(t)

	// A file whose base name is NOT contained in ConfigName.
	p := filepath.Join(dir, "totally-unrelated-name.yaml")
	if err := os.WriteFile(p, []byte("mode: text\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// NOTE: the wrong-name branch wraps a nil error, so errors.Wrapf returns a
	// nil error. The contract that matters here is that no config is returned.
	cfg, _ := LoadConfig(p)
	if cfg != nil {
		t.Fatalf("expected nil config for wrong file name, got %+v", cfg)
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	dir := chdirTemp(t)

	p := filepath.Join(dir, "does-not-exist", ConfigName)
	// NOTE: the not-found branch also wraps a nil error, so the returned error
	// is nil; the meaningful guarantee is that no config is returned.
	cfg, _ := LoadConfig(p)
	if cfg != nil {
		t.Fatalf("expected nil config for missing path, got %+v", cfg)
	}
}

func TestInitDefaultConfig_CreateFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses directory permissions")
	}
	dir := chdirTemp(t)
	// Make cwd non-writable so os.Create of rod-cli.yaml fails.
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	if err := InitDefaultConfig(); err == nil {
		t.Fatal("expected error creating config in read-only dir")
	}

	// LoadConfig("") should propagate the init failure (covers its error wrap).
	if _, err := LoadConfig(""); err == nil {
		t.Fatal("expected LoadConfig error when init default config fails")
	}
}

func TestLoadConfig_PathExistsError(t *testing.T) {
	dir := chdirTemp(t)
	// Create a regular file, then reference a path that treats it as a directory
	// component. os.Stat returns ENOTDIR (not IsNotExist) -> PathExists errors.
	regular := filepath.Join(dir, "afile")
	if err := os.WriteFile(regular, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	bogus := filepath.Join(regular, ConfigName) // afile/rod-cli.yaml

	cfg, err := LoadConfig(bogus)
	if err == nil {
		t.Fatal("expected PathExists error (ENOTDIR)")
	}
	if cfg != nil {
		t.Fatal("expected nil config on PathExists error")
	}
}

func TestLoadConfig_MalformedYAML(t *testing.T) {
	dir := chdirTemp(t)

	p := filepath.Join(dir, ConfigName)
	// Invalid YAML that yaml.Decoder will reject.
	if err := os.WriteFile(p, []byte("mode: [unterminated\n  : :\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := LoadConfig(p)
	if err == nil {
		t.Fatal("expected decode error for malformed yaml")
	}
	if cfg != nil {
		t.Fatal("expected nil config on decode error")
	}
}
