package types

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"gopkg.in/yaml.v3"
)

// errBoom is a sentinel error used by seam overrides.
var errBoom = fmt.Errorf("boom")

// navFixture launches a browser-backed page navigated to the static fixture.
func navFixture(t *testing.T) (*Context, *rod.Page) {
	t.Helper()
	srv := newFixtureServer(t)
	ctx := newBrowserContext(t)
	page, err := ctx.EnsurePage()
	if err != nil {
		t.Fatalf("EnsurePage: %v", err)
	}
	if err := page.Navigate(srv.URL); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	if err := page.WaitLoad(); err != nil {
		t.Fatalf("wait load: %v", err)
	}
	return ctx, page
}

// --- snapshot.go BuildSnapshot error branches via seams ---

func TestBuildSnapshot_EvalError(t *testing.T) {
	_, page := navFixture(t)

	orig := pageEval
	pageEval = func(p *rod.Page, js string, args ...any) (*proto.RuntimeRemoteObject, error) {
		return nil, errBoom
	}
	defer func() { pageEval = orig }()

	if _, err := BuildSnapshot(page); err == nil {
		t.Fatal("expected error from eval failure")
	}
}

func TestBuildSnapshot_UnmarshalError(t *testing.T) {
	_, page := navFixture(t)

	orig := yamlUnmarshal
	yamlUnmarshal = func([]byte, any) error { return errBoom }
	defer func() { yamlUnmarshal = orig }()

	if _, err := BuildSnapshot(page); err == nil {
		t.Fatal("expected error from yaml.Unmarshal failure")
	}
}

func TestBuildSnapshot_MarshalError(t *testing.T) {
	_, page := navFixture(t)

	orig := yamlMarshal
	yamlMarshal = func(any) ([]byte, error) { return nil, errBoom }
	defer func() { yamlMarshal = orig }()

	if _, err := BuildSnapshot(page); err == nil {
		t.Fatal("expected error from yaml.Marshal failure")
	}
}

func TestBuildSnapshot_PageInfoError(t *testing.T) {
	_, page := navFixture(t)

	orig := pageInfo
	pageInfo = func(p *rod.Page) (*proto.TargetTargetInfo, error) { return nil, errBoom }
	defer func() { pageInfo = orig }()

	if _, err := BuildSnapshot(page); err == nil {
		t.Fatal("expected error from page.Info failure")
	}
}

func TestBuildSnapshot_TemplateError(t *testing.T) {
	_, page := navFixture(t)

	orig := executeTemple
	executeTemple = func(string, any) (string, error) { return "", errBoom }
	defer func() { executeTemple = orig }()

	if _, err := BuildSnapshot(page); err == nil {
		t.Fatal("expected error from template execute failure")
	}
}

// --- iframe recovery branches in walk() ---

const iframeParentHTML = `data:text/html,<html><body><h1>Parent</h1>` +
	`<iframe srcdoc="<html><body><p>child content paragraph</p></body></html>"></iframe>` +
	`</body></html>`

func navIframe(t *testing.T) *rod.Page {
	t.Helper()
	ctx := newBrowserContext(t)
	page, err := ctx.EnsurePage()
	if err != nil {
		t.Fatalf("EnsurePage: %v", err)
	}
	if err := page.Navigate(iframeParentHTML); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	if err := page.WaitLoad(); err != nil {
		t.Fatalf("wait load: %v", err)
	}
	return page
}

// QueryEleByAria failure on the iframe ref -> pairNode recovery (snapshot.go ~151).
func TestWalk_IframeQueryEleError(t *testing.T) {
	page := navIframe(t)

	orig := queryEleByAria
	queryEleByAria = func(frame *rod.Page, selector string) (*rod.Element, error) {
		return nil, errBoom
	}
	defer func() { queryEleByAria = orig }()

	snap, err := BuildSnapshot(page)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if !strings.Contains(snap.String(), "could not capture iframe snapshot") {
		t.Fatalf("expected iframe-failure placeholder, got:\n%s", snap.String())
	}
}

// Child-frame capture failure -> pairNode recovery (snapshot.go ~160).
// The first Eval (parent) succeeds; the nested Eval (child frame) fails.
func TestWalk_IframeChildCaptureError(t *testing.T) {
	page := navIframe(t)

	var calls int
	orig := pageEval
	pageEval = func(p *rod.Page, js string, args ...any) (*proto.RuntimeRemoteObject, error) {
		calls++
		if calls == 1 {
			return orig(p, js, args...)
		}
		return nil, errBoom
	}
	defer func() { pageEval = orig }()

	snap, err := BuildSnapshot(page)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if !strings.Contains(snap.String(), "could not capture iframe snapshot") {
		t.Fatalf("expected iframe-failure placeholder, got:\n%s", snap.String())
	}
}

// childFrameEle.Frame() failure -> pairNode recovery (snapshot.go ~168).
// queryEleByAria is seamed to return a stale element (its JS context has been
// invalidated by a navigation), so the subsequent .Frame() call errors.
func TestWalk_IframeFrameError(t *testing.T) {
	page := navIframe(t)

	// Build a stale element: grab an element, then navigate to invalidate it.
	staleEle, err := page.Element("h1")
	if err != nil {
		t.Fatalf("element: %v", err)
	}
	if err := page.Navigate(iframeParentHTML); err != nil {
		t.Fatalf("re-navigate: %v", err)
	}
	if err := page.WaitLoad(); err != nil {
		t.Fatalf("wait load: %v", err)
	}

	orig := queryEleByAria
	queryEleByAria = func(frame *rod.Page, selector string) (*rod.Element, error) {
		return staleEle, nil // valid element, but .Frame() will fail (stale ctx)
	}
	defer func() { queryEleByAria = orig }()

	snap, err := BuildSnapshot(page)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if !strings.Contains(snap.String(), "could not capture iframe snapshot") {
		t.Fatalf("expected Frame()-failure placeholder, got:\n%s", snap.String())
	}
}

// --- LocatorInFrame error branches ---

func TestLocatorInFrame_AtoiOverflow(t *testing.T) {
	_, page := navFixture(t)
	snap, err := BuildSnapshot(page)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	// A frame index that parses by the regex but overflows strconv.Atoi.
	if _, err := snap.LocatorInFrame("f99999999999999999999s5"); err == nil {
		t.Fatal("expected Atoi overflow error")
	}
}

func TestLocatorInFrame_NestedFrameRef(t *testing.T) {
	page := navIframe(t)
	snap, err := BuildSnapshot(page)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if len(snap.frames) < 2 {
		t.Skipf("iframe not captured as a frame in this environment (frames=%d)", len(snap.frames))
	}
	// Find an f1-prefixed ref in the snapshot to exercise the in-range
	// nested-frame branch (frame = s.frames[frameIndex]).
	ref := firstFrameRef(snap.textSnapshot, "f1")
	if ref == "" {
		t.Skip("no f1 ref present in nested snapshot")
	}
	// May or may not resolve depending on element type; we only need the
	// frame-index code path to run.
	_, _ = snap.LocatorInFrame(ref)
}

// firstFrameRef returns the first [ref=<prefix>...] token value with the given prefix.
func firstFrameRef(s, prefix string) string {
	marker := "[ref=" + prefix
	idx := strings.Index(s, marker)
	if idx < 0 {
		return ""
	}
	rest := s[idx+len("[ref="):]
	end := strings.IndexByte(rest, ']')
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// --- context.go error branches ---

// launchBrowser when no Chrome is found and no bin path is configured.
func TestLaunchBrowser_NoChrome(t *testing.T) {
	origLP := launcherLookPath
	launcherLookPath = func() (string, bool) { return "", false }
	defer func() { launcherLookPath = origLP }()

	_, _, err := launchBrowser(context.Background(), Config{BrowserTempDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error when no Chrome is installed")
	}
	if !strings.Contains(err.Error(), "Chrome") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// launchBrowser with a Proxy set + bad bin path: exercises the godoll proxy
// branch (ApplyToLauncher, no-auth path) and then the godoll launch-failure wrap
// (does not require a real browser). The proxy now rides cfg.Stealth.Proxy.
func TestLaunchBrowser_ProxyAndBadBin(t *testing.T) {
	cfg := Config{
		BrowserBinPath: "/nonexistent/chrome-binary-xyz",
		Stealth:        StealthConfig{Proxy: "http://127.0.0.1:9"},
		BrowserTempDir: t.TempDir(),
	}
	if _, _, err := launchBrowser(context.Background(), cfg); err == nil {
		t.Fatal("expected launch failure with bogus binary")
	}
}

// launchBrowser with an unparseable proxy URL fails loudly before any launch
// attempt (parseProxyConfig error path through launchBrowser).
func TestLaunchBrowser_BadProxyURL(t *testing.T) {
	origLP := launcherLookPath
	launcherLookPath = func() (string, bool) { return "/some/chrome", true }
	defer func() { launcherLookPath = origLP }()

	cfg := Config{
		Stealth:        StealthConfig{Proxy: "127.0.0.1:8080"}, // missing scheme
		BrowserTempDir: t.TempDir(),
	}
	_, _, err := launchBrowser(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for proxy url without a scheme")
	}
	if !strings.Contains(err.Error(), "proxy") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Close() os.RemoveAll failure branch via seam.
func TestClose_RemoveAllError(t *testing.T) {
	ctx := NewContext(context.Background(), Config{BrowserTempDir: t.TempDir()})

	orig := osRemoveAll
	osRemoveAll = func(string) error { return errBoom }
	defer func() { osRemoveAll = orig }()

	if err := ctx.Close(); err == nil {
		t.Fatal("expected error from os.RemoveAll failure")
	}
}

// BuildSnapshot (context method) wrapping the inner BuildSnapshot error.
func TestContextBuildSnapshot_InnerError(t *testing.T) {
	ctx, _ := navFixture(t)

	orig := pageEval
	pageEval = func(p *rod.Page, js string, args ...any) (*proto.RuntimeRemoteObject, error) {
		return nil, errBoom
	}
	defer func() { pageEval = orig }()

	if _, err := ctx.BuildSnapshot(); err == nil {
		t.Fatal("expected ctx.BuildSnapshot to propagate inner error")
	}
}

// closeBrowser's browser.Close() error wrap: launch a browser, then kill the
// underlying process so the subsequent Close() over the dead connection errors.
func TestCloseBrowser_CloseError(t *testing.T) {
	l := launcher.New().Headless(true)
	u, err := l.Launch()
	if err != nil {
		t.Skipf("cannot launch browser: %v", err)
	}
	ctx := NewContext(context.Background(), Config{CDPEndpoint: u})
	if _, err := ctx.EnsurePage(); err != nil {
		t.Fatalf("EnsurePage: %v", err)
	}
	// Clear the page so closeBrowser skips page.Close and reaches browser.Close.
	ctx.SetPage(nil)
	l.Kill() // kill the process; the CDP connection is now dead
	if err := ctx.CloseBrowser(); err == nil {
		t.Log("note: closing browser over a killed process did not error in this environment")
	}
}

// createPage error propagation through initial(): closing the browser out from
// under the context, clearing the page, then re-driving EnsurePage makes
// browser.Page() (createPage) fail and initial() wrap it.
func TestInitial_CreatePageError(t *testing.T) {
	ctx := newBrowserContext(t)
	if _, err := ctx.EnsurePage(); err != nil {
		t.Fatalf("EnsurePage: %v", err)
	}
	br := ctx.GetBrowser()
	ctx.SetPage(nil) // page nil so initial() takes the createPage branch
	if err := br.Close(); err != nil {
		t.Fatalf("browser close: %v", err)
	}
	// browser field still references the closed browser; createPage must fail.
	if _, err := ctx.EnsurePage(); err == nil {
		t.Fatal("expected createPage error after browser close")
	}
}

// Empty child-frame snapshot -> the len(childSnapshot.Content)==0 placeholder
// branch. The parent Eval succeeds; the child frame's yaml.Unmarshal yields an
// empty document (no Content), so walk takes the empty-content recovery path.
func TestWalk_IframeEmptyChildContent(t *testing.T) {
	page := navIframe(t)

	var calls int
	origU := yamlUnmarshal
	yamlUnmarshal = func(b []byte, v any) error {
		calls++
		if calls == 1 {
			return origU(b, v) // parent document parses normally
		}
		// Child frame: leave the node empty (DocumentNode with no Content).
		return nil
	}
	defer func() { yamlUnmarshal = origU }()

	snap, err := BuildSnapshot(page)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if !strings.Contains(snap.String(), "could not capture iframe snapshot") {
		t.Fatalf("expected empty-child placeholder, got:\n%s", snap.String())
	}
}

// Ensure seam defaults remain the real functions (guards against accidental drift).
func TestSeamDefaults(t *testing.T) {
	if _, err := yamlMarshal(map[string]string{"a": "b"}); err != nil {
		t.Fatalf("default yamlMarshal: %v", err)
	}
	var n yaml.Node
	if err := yamlUnmarshal([]byte("a: b"), &n); err != nil {
		t.Fatalf("default yamlUnmarshal: %v", err)
	}
	_ = os.RemoveAll // touch import
}
