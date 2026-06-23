package types

import (
	"context"
	"testing"

	"github.com/go-rod/rod/lib/launcher"
)

// TestLaunchBrowser_LocalHeadless exercises the non-CDP launch path of
// launchBrowser (godoll stealth preset, temp dir, LookPath). Requires a local
// Chrome; skips if none is available.
func TestLaunchBrowser_LocalHeadless(t *testing.T) {
	if _, has := launcher.LookPath(); !has {
		t.Skip("no local Chrome found; skipping local-launch path")
	}

	dir := t.TempDir()
	ctx := NewContext(context.Background(), Config{
		Headless:       true,
		NoSandbox:      true,
		BrowserTempDir: dir,
	})
	t.Cleanup(func() { _ = ctx.Close() })

	page, err := ctx.EnsurePage()
	if err != nil {
		t.Fatalf("EnsurePage (local launch): %v", err)
	}
	if page == nil {
		t.Fatal("expected non-nil page from local launch")
	}
	if ctx.GetBrowser() == nil {
		t.Fatal("expected browser after local launch")
	}
}

// TestLaunchBrowser_DefaultTempDirAndReinit covers the empty-BrowserTempDir
// default branch and the initial() path that re-creates a page when the
// browser already exists but the page was closed.
func TestLaunchBrowser_DefaultTempDirAndReinit(t *testing.T) {
	if _, has := launcher.LookPath(); !has {
		t.Skip("no local Chrome found; skipping local-launch path")
	}

	// Empty BrowserTempDir -> launchBrowser applies DefaultBrowserTempDir.
	// Use a chdir into a temp dir so the default "./rod/browser" lands there.
	chdirTemp(t)
	ctx := NewContext(context.Background(), Config{
		Headless:  true,
		NoSandbox: true,
	})
	t.Cleanup(func() { _ = ctx.Close() })

	if _, err := ctx.EnsurePage(); err != nil {
		t.Fatalf("EnsurePage: %v", err)
	}

	// Close just the page, keeping the browser alive.
	if err := ctx.ClosePage(); err != nil {
		t.Fatalf("ClosePage: %v", err)
	}

	// EnsurePage again: browser != nil, page == nil -> re-create page branch.
	if _, err := ctx.EnsurePage(); err != nil {
		t.Fatalf("EnsurePage re-init: %v", err)
	}
	if p, err := ctx.ControlledPage(); err != nil || p == nil {
		t.Fatalf("expected re-created page: %v", err)
	}
}

// TestLaunchBrowser_BadBinPath drives the failure path where an explicit but
// invalid browser binary is configured.
func TestLaunchBrowser_BadBinPath(t *testing.T) {
	dir := t.TempDir()
	ctx := NewContext(context.Background(), Config{
		Headless:       true,
		NoSandbox:      true,
		BrowserBinPath: "/nonexistent/path/to/chrome-binary",
		BrowserTempDir: dir,
	})
	t.Cleanup(func() { _ = ctx.Close() })

	if _, err := ctx.EnsurePage(); err == nil {
		t.Fatal("expected launch error with bad browser bin path")
	}
}
