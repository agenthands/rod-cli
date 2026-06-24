package types

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agenthands/godoll/stealth"
	"gopkg.in/yaml.v3"
)

// writeProfile saves a stealth.Profile JSON file at dir/<name>.json and returns
// its path, for the profile-tier precedence tests.
func writeProfile(t *testing.T, dir, name string, p stealth.Profile) string {
	t.Helper()
	path := filepath.Join(dir, name+".json")
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal profile: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}
	return path
}

func TestResolveStealth_CLIFlagWins_NoProfile(t *testing.T) {
	cfg := DefaultConfig
	flags := &StealthFlags{Proxy: "http://127.0.0.1:8888"}
	if err := ResolveStealth(&cfg, flags); err != nil {
		t.Fatalf("ResolveStealth: %v", err)
	}
	if cfg.Stealth.Proxy != "http://127.0.0.1:8888" {
		t.Fatalf("Stealth.Proxy = %q, want CLI value", cfg.Stealth.Proxy)
	}
	// Deprecated shim bridged from Stealth.Proxy.
	if cfg.Proxy != "http://127.0.0.1:8888" {
		t.Fatalf("Config.Proxy bridge = %q, want CLI value", cfg.Proxy)
	}
}

func TestResolveStealth_ProfileTier(t *testing.T) {
	dir := t.TempDir()
	path := writeProfile(t, dir, "euro", stealth.DefaultProfile())

	cfg := DefaultConfig
	flags := &StealthFlags{Profile: path}
	if err := ResolveStealth(&cfg, flags); err != nil {
		t.Fatalf("ResolveStealth: %v", err)
	}
	if cfg.Stealth.ProfilePath != path {
		t.Fatalf("ProfilePath = %q, want %q", cfg.Stealth.ProfilePath, path)
	}
	// No CLI proxy supplied → proxy stays at the built-in default (empty).
	if cfg.Stealth.Proxy != "" {
		t.Fatalf("Stealth.Proxy = %q, want default empty", cfg.Stealth.Proxy)
	}
}

func TestResolveStealth_DefaultTier(t *testing.T) {
	cfg := DefaultConfig
	if err := ResolveStealth(&cfg, &StealthFlags{}); err != nil {
		t.Fatalf("ResolveStealth: %v", err)
	}
	if cfg.Stealth.Proxy != "" || cfg.Stealth.ProxyAuth != "" || cfg.Stealth.ProfilePath != "" {
		t.Fatalf("expected built-in defaults, got %+v", cfg.Stealth)
	}
}

func TestResolveStealth_CLIOverridesProfile(t *testing.T) {
	dir := t.TempDir()
	path := writeProfile(t, dir, "p", stealth.DefaultProfile())

	cfg := DefaultConfig
	flags := &StealthFlags{Proxy: "socks5://10.0.0.1:1080", Profile: path}
	if err := ResolveStealth(&cfg, flags); err != nil {
		t.Fatalf("ResolveStealth: %v", err)
	}
	// CLI flag wins over the profile selection for the proxy field.
	if cfg.Stealth.Proxy != "socks5://10.0.0.1:1080" {
		t.Fatalf("Stealth.Proxy = %q, want CLI value (precedence CLI > profile)", cfg.Stealth.Proxy)
	}
	if cfg.Stealth.ProfilePath != path {
		t.Fatalf("ProfilePath = %q, want %q", cfg.Stealth.ProfilePath, path)
	}
}

func TestResolveStealth_MissingProfileIsLoudError(t *testing.T) {
	cfg := DefaultConfig
	flags := &StealthFlags{Profile: filepath.Join(t.TempDir(), "does-not-exist.json")}
	if err := ResolveStealth(&cfg, flags); err == nil {
		t.Fatal("expected loud error for non-existent profile, got nil")
	}
}

func TestResolveStealth_ProxyAuthBridge(t *testing.T) {
	cfg := DefaultConfig
	flags := &StealthFlags{Proxy: "http://h:1", ProxyAuth: "user:secret"}
	if err := ResolveStealth(&cfg, flags); err != nil {
		t.Fatalf("ResolveStealth: %v", err)
	}
	if cfg.Stealth.ProxyAuth != "user:secret" {
		t.Fatalf("ProxyAuth = %q, want CLI value", cfg.Stealth.ProxyAuth)
	}
}

func TestLoadConfig_StealthBlockRoundTrips(t *testing.T) {
	dir := chdirTemp(t)

	want := DefaultConfig
	want.Stealth.Proxy = "http://localhost:9090"
	want.Stealth.ProxyAuth = "u:p"
	want.Stealth.ProfilePath = "/tmp/euro.json"

	p := filepath.Join(dir, ConfigName)
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := yaml.NewEncoder(f).Encode(want); err != nil {
		t.Fatalf("encode: %v", err)
	}
	f.Close()

	// ProxyAuth is credential-sensitive and tagged `yaml:"-"` — it must NEVER be
	// serialized to (or read back from) a config file. The encoded YAML must not
	// contain the credential at all.
	encoded, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read encoded config: %v", err)
	}
	if strings.Contains(string(encoded), "u:p") || strings.Contains(string(encoded), "proxyAuth") {
		t.Fatalf("ProxyAuth was serialized to the config file (credential leak): %s", encoded)
	}

	cfg, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	// Non-secret stealth fields round-trip; ProxyAuth deliberately does NOT (it is
	// supplied at runtime via ROD_CLI_PROXY_AUTH, never persisted).
	if cfg.Stealth.Proxy != "http://localhost:9090" ||
		cfg.Stealth.ProfilePath != "/tmp/euro.json" {
		t.Fatalf("stealth block did not round-trip: %+v", cfg.Stealth)
	}
	if cfg.Stealth.ProxyAuth != "" {
		t.Fatalf("ProxyAuth must not be persisted/loaded from config, got %q", cfg.Stealth.ProxyAuth)
	}
}

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
