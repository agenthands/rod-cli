package types

import (
	"path/filepath"
	"testing"
)

func TestDefaultProfileRootIsUserCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/tmp/rod-cli-xdg-test")
	got := DefaultProfileRoot()
	want := filepath.Join("/tmp/rod-cli-xdg-test", "rod-cli", "profiles")
	if !filepath.IsAbs(got) || got != want {
		t.Fatalf("DefaultProfileRoot() = %q, want %q", got, want)
	}
}
