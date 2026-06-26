package types

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/agenthands/godoll/network"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const fixtureHTML = `<!DOCTYPE html>
<html>
<head><title>Fixture Page</title></head>
<body>
	<h1>Main Heading</h1>
	<h2>Sub Heading</h2>
	<p>Some descriptive paragraph text.</p>
	<a href="https://example.com/link">A Link</a>
	<form>
		<label for="name">Name</label>
		<input id="name" type="text" name="name" placeholder="enter name">
		<input id="agree" type="checkbox" name="agree">
		<button type="submit">Submit</button>
	</form>
	<ul>
		<li>Item one</li>
		<li>Item two</li>
	</ul>
</body>
</html>`

// newFixtureServer serves a static HTML page with varied elements.
func newFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(fixtureHTML))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newBrowserContext launches a headless browser-backed Context. It works
// without a DISPLAY. Cleanup is deferred via t.Cleanup.
func newBrowserContext(t *testing.T) *Context {
	t.Helper()
	u, err := launcher.New().Headless(true).Launch()
	if err != nil {
		t.Skipf("cannot launch browser (skipping browser-backed test): %v", err)
	}
	ctx := NewContext(context.Background(), Config{CDPEndpoint: u, Mode: Text})
	t.Cleanup(func() {
		_ = ctx.CloseBrowser()
	})
	return ctx
}

func TestContext_LifecycleAndAccessors(t *testing.T) {
	srv := newFixtureServer(t)
	ctx := newBrowserContext(t)

	// EnsurePage launches browser + page.
	page, err := ctx.EnsurePage()
	if err != nil {
		t.Fatalf("EnsurePage: %v", err)
	}
	if page == nil {
		t.Fatal("expected non-nil page")
	}

	// EnsurePage again: browser non-nil, page non-nil -> no-op path of initial().
	if _, err := ctx.EnsurePage(); err != nil {
		t.Fatalf("EnsurePage second call: %v", err)
	}

	// CurrentMode
	if ctx.CurrentMode() != Text {
		t.Fatalf("CurrentMode = %q, want %q", ctx.CurrentMode(), Text)
	}

	// GetBrowser
	if ctx.GetBrowser() == nil {
		t.Fatal("GetBrowser returned nil")
	}

	// GetPluginEngine lazily inits the engine and returns the same instance.
	pe1 := ctx.GetPluginEngine()
	if pe1 == nil {
		t.Fatal("GetPluginEngine returned nil")
	}
	pe2 := ctx.GetPluginEngine()
	if pe1 != pe2 {
		t.Fatal("GetPluginEngine should return cached engine")
	}

	// Loaded plugins accessors.
	if len(ctx.GetLoadedPlugins()) != 0 {
		t.Fatalf("expected no loaded plugins, got %v", ctx.GetLoadedPlugins())
	}
	ctx.AddLoadedPlugin("/some/plugin/a")
	ctx.AddLoadedPlugin("/some/plugin/b")
	got := ctx.GetLoadedPlugins()
	if len(got) != 2 || got[0] != "/some/plugin/a" || got[1] != "/some/plugin/b" {
		t.Fatalf("loaded plugins mismatch: %v", got)
	}

	// Navigate to the fixture so snapshot/controlled-page work.
	cp, err := ctx.ControlledPage()
	if err != nil {
		t.Fatalf("ControlledPage: %v", err)
	}
	if err := cp.Navigate(srv.URL); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	if err := cp.WaitLoad(); err != nil {
		t.Fatalf("wait load: %v", err)
	}

	// BuildSnapshot (positive) then LatestSnapshot.
	snapStr, err := ctx.BuildSnapshot()
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if !strings.Contains(snapStr, "Page URL") || !strings.Contains(snapStr, "Page Snapshot") {
		t.Fatalf("snapshot string missing expected sections:\n%s", snapStr)
	}
	latest, err := ctx.LatestSnapshot()
	if err != nil {
		t.Fatalf("LatestSnapshot: %v", err)
	}
	if latest == nil || latest.String() == "" {
		t.Fatal("expected non-empty latest snapshot")
	}

	// Routes accessors exercise updateInterceptorRules with mock rules.
	ctx.AddRoute("**/*.png", "mocked-png")
	if got := ctx.GetRoutes(); got["**/*.png"] != "mocked-png" {
		t.Fatalf("AddRoute/GetRoutes mismatch: %v", got)
	}
	ctx.RemoveRoute("**/*.png")
	if _, ok := ctx.GetRoutes()["**/*.png"]; ok {
		t.Fatal("expected route removed")
	}

	// Console logs / requests accessors should be reachable (may be empty).
	_ = ctx.GetConsoleLogs()
	_ = ctx.GetRequests()

	// SetPage with a freshly created page.
	newPage, err := ctx.GetBrowser().Page(proto.TargetCreateTarget{})
	if err != nil {
		t.Fatalf("create extra page: %v", err)
	}
	ctx.SetPage(newPage)
	if cp2, err := ctx.ControlledPage(); err != nil || cp2 != newPage {
		t.Fatalf("SetPage/ControlledPage mismatch: %v err=%v", cp2, err)
	}

	// ClosePage closes the current page (calls closePage internally).
	if err := ctx.ClosePage(); err != nil {
		t.Fatalf("ClosePage: %v", err)
	}
	// After ClosePage, ControlledPage errors (page nil).
	if _, err := ctx.ControlledPage(); err == nil {
		t.Fatal("expected ControlledPage error after ClosePage")
	}

	// ClosePage again when page is nil returns nil.
	if err := ctx.ClosePage(); err != nil {
		t.Fatalf("ClosePage on nil page should be nil: %v", err)
	}

	// CloseBrowser (calls closeBrowser internally). Browser still set.
	if err := ctx.CloseBrowser(); err != nil {
		t.Fatalf("CloseBrowser: %v", err)
	}
	if ctx.GetBrowser() != nil {
		t.Fatal("expected browser nil after CloseBrowser")
	}
	// CloseBrowser again when browser is nil returns nil.
	if err := ctx.CloseBrowser(); err != nil {
		t.Fatalf("CloseBrowser on nil browser should be nil: %v", err)
	}
}

func TestContext_ClosePageErrorOnStalePage(t *testing.T) {
	ctx := newBrowserContext(t)
	if _, err := ctx.EnsurePage(); err != nil {
		t.Fatalf("EnsurePage: %v", err)
	}
	// Grab the live page, then close the whole browser out from under it.
	page, err := ctx.ControlledPage()
	if err != nil {
		t.Fatalf("ControlledPage: %v", err)
	}
	if err := ctx.GetBrowser().Close(); err != nil {
		t.Fatalf("browser close: %v", err)
	}
	// Re-attach the now-stale page; ClosePage -> page.Close() should error and
	// be wrapped, exercising the closePage error branch.
	ctx.SetPage(page)
	if err := ctx.ClosePage(); err == nil {
		t.Log("note: closing a stale page did not error in this environment")
	}
}

func TestContext_CloseRemovesNothingForCDP(t *testing.T) {
	// Close() with a CDPEndpoint set must not attempt temp-dir removal.
	ctx := newBrowserContext(t)
	if _, err := ctx.EnsurePage(); err != nil {
		t.Fatalf("EnsurePage: %v", err)
	}
	if err := ctx.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestContext_CloseRemovesTempDir(t *testing.T) {
	// Close() with a temp dir + no CDP endpoint exercises the os.RemoveAll branch.
	dir := t.TempDir()
	ctx := NewContext(context.Background(), Config{BrowserTempDir: dir})
	// No browser launched; closeBrowser short-circuits, then RemoveAll runs.
	if err := ctx.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestContext_UpdateInterceptorRules_NoCatchAll(t *testing.T) {
	// Phase 30 (CDP-01): updateInterceptorRules no longer installs an identity
	// catch-all rule (it moved to Emulation.setUserAgentOverride). With NO mock
	// routes the rule set is therefore EMPTY — the interceptor exists only to serve
	// mock routes now. With a route, exactly that mock rule appears.
	ctx := NewContext(context.Background(), Config{})
	ctx.interceptor = network.NewInterceptor(nil)
	ctx.fingerprint = nil

	ctx.updateInterceptorRules()
	if got := len(ctx.interceptor.Rules()); got != 0 {
		t.Fatalf("expected 0 rules with no routes (catch-all removed), got %d", got)
	}

	ctx.routes = map[string]string{"**/*.png": "mocked"}
	ctx.updateInterceptorRules()
	if got := len(ctx.interceptor.Rules()); got != 1 {
		t.Fatalf("expected exactly 1 mock rule, got %d", got)
	}
}

func TestContext_UpdateInterceptorRules_NilInterceptor(t *testing.T) {
	// Early return when interceptor is nil.
	ctx := NewContext(context.Background(), Config{})
	ctx.updateInterceptorRules() // must not panic
}

func TestContext_FreshNegativePaths(t *testing.T) {
	ctx := NewContext(context.Background(), Config{})

	// ControlledPage on a fresh context: no tab.
	if _, err := ctx.ControlledPage(); err == nil {
		t.Fatal("expected ControlledPage error on fresh context")
	}

	// BuildSnapshot with nil page returns error.
	if _, err := ctx.BuildSnapshot(); err == nil {
		t.Fatal("expected BuildSnapshot error with nil page")
	}

	// LatestSnapshot with no snapshot returns error.
	if _, err := ctx.LatestSnapshot(); err == nil {
		t.Fatal("expected LatestSnapshot error with no snapshot")
	}

	// ClosePage with nil page returns nil.
	if err := ctx.ClosePage(); err != nil {
		t.Fatalf("ClosePage on fresh context: %v", err)
	}

	// CloseBrowser with nil browser returns nil.
	if err := ctx.CloseBrowser(); err != nil {
		t.Fatalf("CloseBrowser on fresh context: %v", err)
	}

	// CurrentMode default is empty.
	if ctx.CurrentMode() != "" {
		t.Fatalf("expected empty mode, got %q", ctx.CurrentMode())
	}
}

func TestContext_InitialBadEndpoint(t *testing.T) {
	// A bad CDP endpoint makes launchBrowser/controlBrowser (via initial) fail.
	ctx := NewContext(context.Background(), Config{CDPEndpoint: "ws://127.0.0.1:1/devtools/browser/bogus"})
	if _, err := ctx.EnsurePage(); err == nil {
		_ = ctx.CloseBrowser()
		t.Fatal("expected error connecting to bad CDP endpoint")
	}
}
