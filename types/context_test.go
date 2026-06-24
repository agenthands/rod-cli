package types

import (
	"testing"

	"github.com/agenthands/godoll/browser"
)

// TestParseProxyConfig covers the pure-parsing helper that maps the
// cfg.Stealth.Proxy URL (+ cfg.Stealth.ProxyAuth) onto a godoll
// browser.ProxyConfig. The helper must never carry URL-embedded credentials
// into the launcher URL (T-25-05) and must split proxyAuth on the FIRST colon.
func TestParseProxyConfig(t *testing.T) {
	t.Run("http no auth", func(t *testing.T) {
		cfg, err := parseProxyConfig("http://127.0.0.1:8080", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg == nil {
			t.Fatal("expected non-nil config")
		}
		if cfg.Protocol != "http" {
			t.Errorf("Protocol = %q, want http", cfg.Protocol)
		}
		if cfg.Address != "127.0.0.1:8080" {
			t.Errorf("Address = %q, want 127.0.0.1:8080", cfg.Address)
		}
		if cfg.HasAuth() {
			t.Error("HasAuth() = true, want false")
		}
	})

	t.Run("socks5 with auth", func(t *testing.T) {
		cfg, err := parseProxyConfig("socks5://127.0.0.1:1080", "user:pass")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg == nil {
			t.Fatal("expected non-nil config")
		}
		if cfg.Protocol != "socks5" {
			t.Errorf("Protocol = %q, want socks5", cfg.Protocol)
		}
		if cfg.Address != "127.0.0.1:1080" {
			t.Errorf("Address = %q, want 127.0.0.1:1080", cfg.Address)
		}
		if cfg.Username != "user" || cfg.Password != "pass" {
			t.Errorf("creds = %q/%q, want user/pass", cfg.Username, cfg.Password)
		}
		if !cfg.HasAuth() {
			t.Error("HasAuth() = false, want true")
		}
	})

	t.Run("empty url returns nil config", func(t *testing.T) {
		cfg, err := parseProxyConfig("", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg != nil {
			t.Errorf("expected nil config for empty url, got %+v", cfg)
		}
	})

	t.Run("empty url with auth still returns nil config", func(t *testing.T) {
		// No proxy URL means no proxy at all; auth alone must not synthesize one.
		cfg, err := parseProxyConfig("", "user:pass")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg != nil {
			t.Errorf("expected nil config when no proxy url, got %+v", cfg)
		}
	})

	t.Run("https scheme normalized to http", func(t *testing.T) {
		cfg, err := parseProxyConfig("https://127.0.0.1:8080", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Protocol != "http" {
			t.Errorf("Protocol = %q, want http (https normalized)", cfg.Protocol)
		}
	})

	t.Run("embedded creds never reach launcher url", func(t *testing.T) {
		// Embedded user:pass@ must not leak into Chrome's --proxy-server. The
		// helper either rejects (error) or strips them; in both cases the
		// LauncherURL must be credential-free.
		cfg, err := parseProxyConfig("http://user:s3cret@127.0.0.1:8080", "")
		if err != nil {
			// Rejection is an acceptable disposition.
			return
		}
		if cfg == nil {
			t.Fatal("expected non-nil config when creds are stripped")
		}
		lu := cfg.LauncherURL()
		if containsAny(lu, "user", "s3cret") {
			t.Errorf("LauncherURL() = %q leaks embedded credentials", lu)
		}
		if cfg.Address != "127.0.0.1:8080" {
			t.Errorf("Address = %q, want 127.0.0.1:8080 (creds stripped)", cfg.Address)
		}
	})

	t.Run("unparseable url is a loud error", func(t *testing.T) {
		cfg, err := parseProxyConfig("://:::not a url", "")
		if err == nil {
			t.Errorf("expected error for unparseable url, got cfg=%+v", cfg)
		}
	})

	t.Run("missing scheme is a loud error", func(t *testing.T) {
		// No scheme means we cannot map Protocol -> bare host:port is ambiguous.
		_, err := parseProxyConfig("127.0.0.1:8080", "")
		if err == nil {
			t.Error("expected error for url without scheme")
		}
	})

	t.Run("proxyAuth splits on first colon only", func(t *testing.T) {
		cfg, err := parseProxyConfig("http://127.0.0.1:8080", "user:pass:with:colons")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Username != "user" {
			t.Errorf("Username = %q, want user", cfg.Username)
		}
		if cfg.Password != "pass:with:colons" {
			t.Errorf("Password = %q, want pass:with:colons", cfg.Password)
		}
	})

	t.Run("proxyAuth without colon is loud error", func(t *testing.T) {
		// A malformed auth value (no colon) cannot be split into user:pass; fail
		// loudly rather than silently auth with an empty password.
		_, err := parseProxyConfig("http://127.0.0.1:8080", "justauser")
		if err == nil {
			t.Error("expected error for proxyAuth without a colon")
		}
	})
}

// compile-time assertion that the helper returns godoll's ProxyConfig type.
var _ = func() *browser.ProxyConfig { c, _ := parseProxyConfig("", ""); return c }

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && len(s) >= len(sub) {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
