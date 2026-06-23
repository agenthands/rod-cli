package actions

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/agenthands/godoll/humanize"
	"github.com/agenthands/rod-cli/types"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

// errBoom is the canonical injected error used across these error-path tests.
var errBoom = fmt.Errorf("boom")

// withLiveFixture spins up a live headless browser navigated to the fixture and
// returns the ctx plus a combined cleanup.
func withLiveFixture(t *testing.T) (*types.Context, func()) {
	t.Helper()
	ctx, cleanup := newLiveCtx(t)
	srv := newFixtureServer()
	navFixture(t, ctx, srv)
	return ctx, func() {
		srv.Close()
		cleanup()
	}
}

// ---------------------------------------------------------------------------
// Retry-wrapped navigation (GoBack / GoForward / Reload)
// ---------------------------------------------------------------------------

func TestErrPaths_NavRetryWraps(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	origBack := pageNavigateBack
	origFwd := pageNavigateForward
	origReload := pageReload
	pageNavigateBack = func(p *rod.Page) error { return errBoom }
	pageNavigateForward = func(p *rod.Page) error { return errBoom }
	pageReload = func(p *rod.Page) error { return errBoom }
	defer func() {
		pageNavigateBack = origBack
		pageNavigateForward = origFwd
		pageReload = origReload
	}()

	if _, err := GoBack(ctx); err == nil {
		t.Error("GoBack expected wrapped error")
	}
	if _, err := GoForward(ctx); err == nil {
		t.Error("GoForward expected wrapped error")
	}
	if _, err := Reload(ctx); err == nil {
		t.Error("Reload expected wrapped error")
	}
}

// ---------------------------------------------------------------------------
// Keyboard
// ---------------------------------------------------------------------------

func TestErrPaths_PressKeyAndPress(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	origType := keyboardType
	origPress := keyboardPress
	keyboardType = func(kb *rod.Keyboard, keys ...input.Key) error { return errBoom }
	keyboardPress = func(kb *rod.Keyboard, key input.Key) error { return errBoom }
	defer func() { keyboardType = origType; keyboardPress = origPress }()

	if _, err := PressKey(ctx, 'a'); err == nil {
		t.Error("PressKey expected wrapped error")
	}
	if _, err := Press(ctx, "Enter"); err == nil {
		t.Error("Press expected wrapped error")
	}
}

// ---------------------------------------------------------------------------
// CloseBrowser
// ---------------------------------------------------------------------------

func TestErrPaths_CloseBrowser(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	orig := ctxCloseBrowser
	ctxCloseBrowser = func(c *types.Context) error { return errBoom }
	defer func() { ctxCloseBrowser = orig }()

	if _, err := CloseBrowser(ctx); err == nil {
		t.Error("CloseBrowser expected wrapped error")
	}
}

// ---------------------------------------------------------------------------
// Evaluate: element.Eval, page.Eval (func branch), proto RuntimeEvaluate fail
// ---------------------------------------------------------------------------

func TestErrPaths_EvaluateWraps(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	ref := refFor(t, ctx, "Click Me")

	// element.Eval failure
	origElem := elementEval
	elementEval = func(el *rod.Element, js string, params ...interface{}) (*proto.RuntimeRemoteObject, error) {
		return nil, errBoom
	}
	if _, err := Evaluate(ctx, "() => 1", ref); err == nil {
		t.Error("Evaluate element expected wrapped error")
	}
	elementEval = origElem

	// page.Eval (function) failure
	origPage := pageEval
	pageEval = func(p *rod.Page, js string, args ...interface{}) (*proto.RuntimeRemoteObject, error) {
		return nil, errBoom
	}
	if _, err := Evaluate(ctx, "() => 1", ""); err == nil {
		t.Error("Evaluate page func expected wrapped error")
	}
	pageEval = origPage

	// proto RuntimeEvaluate (expression branch) failure
	origRT := runtimeEvaluateCall
	runtimeEvaluateCall = func(req proto.RuntimeEvaluate, page *rod.Page) (*proto.RuntimeEvaluateResult, error) {
		return nil, errBoom
	}
	if _, err := Evaluate(ctx, "1 + 1", ""); err == nil {
		t.Error("Evaluate expression expected wrapped error")
	}
	runtimeEvaluateCall = origRT
}

// ---------------------------------------------------------------------------
// Screenshot: op fail, mkdir fail, write fail (real triggers + seam for op)
// ---------------------------------------------------------------------------

func TestErrPaths_Screenshot(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	// op (page.Screenshot) failure via seam
	origShot := pageScreenshot
	pageScreenshot = func(p *rod.Page, fullPage bool, req *proto.PageCaptureScreenshot) ([]byte, error) {
		return nil, errBoom
	}
	if _, err := Screenshot(ctx, "x", "", 0, 0); err == nil {
		t.Error("Screenshot op expected wrapped error")
	}
	pageScreenshot = origShot

	// mkdir failure (real trigger): chdir into tempdir, make "tmp" a FILE so
	// MkdirAll on tmp/screenshots fails.
	withTempCwd(t, func(dir string) {
		if err := os.WriteFile(filepath.Join(dir, "tmp"), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := Screenshot(ctx, "shot", "", 0, 0); err == nil {
			t.Error("Screenshot mkdir expected error")
		}
	})

	// write failure (real trigger): pre-create the target file path as a DIRECTORY.
	withTempCwd(t, func(dir string) {
		target := filepath.Join(dir, "tmp", "screenshots", "shot.png")
		if err := os.MkdirAll(target, 0755); err != nil {
			t.Fatal(err)
		}
		if _, err := Screenshot(ctx, "shot", "", 0, 0); err == nil {
			t.Error("Screenshot write expected error")
		}
	})
}

// ---------------------------------------------------------------------------
// Pdf: op fail, ReadAll fail, mkdir fail, write fail
// ---------------------------------------------------------------------------

func TestErrPaths_Pdf(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	// op (page.PDF) failure
	origPDF := pagePDF
	pagePDF = func(p *rod.Page, req *proto.PagePrintToPDF) (*rod.StreamReader, error) {
		return nil, errBoom
	}
	if _, err := Pdf(ctx, "x"); err == nil {
		t.Error("Pdf op expected wrapped error")
	}
	pagePDF = origPDF

	// ReadAll failure
	origRead := ioReadAll
	ioReadAll = func(r io.Reader) ([]byte, error) { return nil, errBoom }
	if _, err := Pdf(ctx, "x"); err == nil {
		t.Error("Pdf ReadAll expected error")
	}
	ioReadAll = origRead

	// mkdir failure (real trigger)
	withTempCwd(t, func(dir string) {
		if err := os.WriteFile(filepath.Join(dir, "tmp"), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := Pdf(ctx, "p"); err == nil {
			t.Error("Pdf mkdir expected error")
		}
	})

	// write failure (real trigger): target file path is a directory
	withTempCwd(t, func(dir string) {
		target := filepath.Join(dir, "tmp", "pdfs", "p.pdf")
		if err := os.MkdirAll(target, 0755); err != nil {
			t.Fatal(err)
		}
		if _, err := Pdf(ctx, "p"); err == nil {
			t.Error("Pdf write expected error")
		}
	})
}

// ---------------------------------------------------------------------------
// Element op-fail wraps: Click, DblClick, Hover, Check, Uncheck, Type, Fill,
// Upload, Select, Highlight
// ---------------------------------------------------------------------------

func TestErrPaths_ElementOps(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	// Click via clickWithMouse
	t.Run("Click", func(t *testing.T) {
		ref := refFor(t, ctx, "Click Me")
		orig := clickWithMouse
		clickWithMouse = func(p *rod.Page, e *rod.Element, opts ...humanize.MouseOption) error { return errBoom }
		defer func() { clickWithMouse = orig }()
		if _, err := Click(ctx, ref); err == nil {
			t.Error("Click expected wrapped error")
		}
	})

	// DblClick via elementClick
	t.Run("DblClick", func(t *testing.T) {
		ref := refFor(t, ctx, "Double Click")
		orig := elementClick
		elementClick = func(el *rod.Element, button proto.InputMouseButton, n int) error { return errBoom }
		defer func() { elementClick = orig }()
		if _, err := DblClick(ctx, ref); err == nil {
			t.Error("DblClick expected wrapped error")
		}
	})

	// Hover via humanizeHover
	t.Run("Hover", func(t *testing.T) {
		ref := refFor(t, ctx, "Click Me")
		orig := humanizeHover
		humanizeHover = func(el *rod.Element) error { return errBoom }
		defer func() { humanizeHover = orig }()
		if _, err := Hover(ctx, ref); err == nil {
			t.Error("Hover expected wrapped error")
		}
	})

	// Check / Uncheck via elementEval
	t.Run("CheckUncheck", func(t *testing.T) {
		orig := elementEval
		elementEval = func(el *rod.Element, js string, params ...interface{}) (*proto.RuntimeRemoteObject, error) {
			return nil, errBoom
		}
		defer func() { elementEval = orig }()
		ref := refFor(t, ctx, "Check Box")
		if _, err := Check(ctx, ref); err == nil {
			t.Error("Check expected wrapped error")
		}
		if _, err := Uncheck(ctx, ref); err == nil {
			t.Error("Uncheck expected wrapped error")
		}
	})

	// Type / Fill via typeWithHumanize
	t.Run("TypeFill", func(t *testing.T) {
		orig := typeWithHumanize
		typeWithHumanize = func(el *rod.Element, text string, opts ...humanize.TypingOption) error { return errBoom }
		defer func() { typeWithHumanize = orig }()
		ref := refFor(t, ctx, "Text Input")
		if _, err := Type(ctx, ref, "x"); err == nil {
			t.Error("Type expected wrapped error")
		}
		ref2 := refFor(t, ctx, "Text Input")
		if _, err := Fill(ctx, ref2, "x", false); err == nil {
			t.Error("Fill expected wrapped error")
		}
	})

	// Fill submit branch via keyboardPress
	t.Run("FillSubmit", func(t *testing.T) {
		orig := keyboardPress
		keyboardPress = func(kb *rod.Keyboard, key input.Key) error { return errBoom }
		defer func() { keyboardPress = orig }()
		ref := refFor(t, ctx, "Text Input")
		if _, err := Fill(ctx, ref, "x", true); err == nil {
			t.Error("Fill submit expected wrapped error")
		}
	})

	// Upload via elementSetFiles
	t.Run("Upload", func(t *testing.T) {
		orig := elementSetFiles
		elementSetFiles = func(el *rod.Element, paths []string) error { return errBoom }
		defer func() { elementSetFiles = orig }()
		dir := t.TempDir()
		fp := filepath.Join(dir, "u.txt")
		_ = writeFile(fp, "x")
		ref := refFor(t, ctx, "File Input")
		if _, err := Upload(ctx, ref, []string{fp}); err == nil {
			t.Error("Upload expected wrapped error")
		}
	})

	// Drop via elementSetFiles
	t.Run("Drop", func(t *testing.T) {
		orig := elementSetFiles
		elementSetFiles = func(el *rod.Element, paths []string) error { return errBoom }
		defer func() { elementSetFiles = orig }()
		dir := t.TempDir()
		fp := filepath.Join(dir, "d.txt")
		_ = writeFile(fp, "x")
		ref := refFor(t, ctx, "File Input")
		if _, err := Drop(ctx, ref, fp); err == nil {
			t.Error("Drop expected wrapped error")
		}
	})

	// Select via elementSelect
	t.Run("Select", func(t *testing.T) {
		orig := elementSelect
		elementSelect = func(el *rod.Element, selectors []string, selected bool, ty rod.SelectorType) error {
			return errBoom
		}
		defer func() { elementSelect = orig }()
		ref := refFor(t, ctx, "Select Box")
		if _, err := Select(ctx, ref, []string{"Apple"}); err == nil {
			t.Error("Select expected wrapped error")
		}
	})

	// Highlight via elementEval
	t.Run("Highlight", func(t *testing.T) {
		orig := elementEval
		elementEval = func(el *rod.Element, js string, params ...interface{}) (*proto.RuntimeRemoteObject, error) {
			return nil, errBoom
		}
		defer func() { elementEval = orig }()
		ref := refFor(t, ctx, "Click Me")
		if _, err := Highlight(ctx, ref); err == nil {
			t.Error("Highlight expected wrapped error")
		}
	})

	// ClearHighlights via pageEval
	t.Run("ClearHighlights", func(t *testing.T) {
		orig := pageEval
		pageEval = func(p *rod.Page, js string, args ...interface{}) (*proto.RuntimeRemoteObject, error) {
			return nil, errBoom
		}
		defer func() { pageEval = orig }()
		if _, err := ClearHighlights(ctx); err == nil {
			t.Error("ClearHighlights expected wrapped error")
		}
	})

	// Drag via humanizeDragAndDrop
	t.Run("Drag", func(t *testing.T) {
		orig := humanizeDragAndDrop
		humanizeDragAndDrop = func(page *rod.Page, source, target *rod.Element) error { return errBoom }
		defer func() { humanizeDragAndDrop = orig }()
		src := refFor(t, ctx, "Drag Source")
		dst := refFor(t, ctx, "Drag Dest")
		if _, err := Drag(ctx, src, dst); err == nil {
			t.Error("Drag expected wrapped error")
		}
	})
}

// ---------------------------------------------------------------------------
// MouseWheel (all four directions), Resize, MouseMove/Down/Up
// ---------------------------------------------------------------------------

func TestErrPaths_MouseAndResize(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	origScroll := humanizeScrollBy
	humanizeScrollBy = func(page *rod.Page, direction humanize.ScrollDirection, amount int, opts ...humanize.ScrollOption) error {
		return errBoom
	}
	if _, err := MouseWheel(ctx, 0, 100); err == nil {
		t.Error("MouseWheel down expected wrapped error")
	}
	if _, err := MouseWheel(ctx, 0, -100); err == nil {
		t.Error("MouseWheel up expected wrapped error")
	}
	if _, err := MouseWheel(ctx, 100, 0); err == nil {
		t.Error("MouseWheel right expected wrapped error")
	}
	if _, err := MouseWheel(ctx, -100, 0); err == nil {
		t.Error("MouseWheel left expected wrapped error")
	}
	humanizeScrollBy = origScroll

	origVP := pageSetViewport
	pageSetViewport = func(p *rod.Page, params *proto.EmulationSetDeviceMetricsOverride) error { return errBoom }
	if _, err := Resize(ctx, 100, 100); err == nil {
		t.Error("Resize expected wrapped error")
	}
	pageSetViewport = origVP

	origMove := pageMouseMoveTo
	pageMouseMoveTo = func(p *rod.Page, pt proto.Point) error { return errBoom }
	if _, err := MouseMove(ctx, 1, 1); err == nil {
		t.Error("MouseMove expected wrapped error")
	}
	pageMouseMoveTo = origMove

	origDown := pageMouseDown
	pageMouseDown = func(p *rod.Page, btn proto.InputMouseButton, clicks int) error { return errBoom }
	if _, err := MouseDown(ctx, "left"); err == nil {
		t.Error("MouseDown expected wrapped error")
	}
	pageMouseDown = origDown

	origUp := pageMouseUp
	pageMouseUp = func(p *rod.Page, btn proto.InputMouseButton, clicks int) error { return errBoom }
	if _, err := MouseUp(ctx, "left"); err == nil {
		t.Error("MouseUp expected wrapped error")
	}
	pageMouseUp = origUp
}

// ---------------------------------------------------------------------------
// Tabs: TabNew, TabList Pages-fail, TabClose Pages-fail + Close-fail,
// TabSelect Pages-fail + Activate-fail
// ---------------------------------------------------------------------------

func TestErrPaths_Tabs(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	// TabNew via browserPage
	origNew := browserPage
	browserPage = func(b *rod.Browser, opts proto.TargetCreateTarget) (*rod.Page, error) { return nil, errBoom }
	if _, err := TabNew(ctx, ""); err == nil {
		t.Error("TabNew expected wrapped error")
	}
	browserPage = origNew

	// browserPages failure affects TabList / TabClose / TabSelect
	origPages := browserPages
	browserPages = func(b *rod.Browser) (rod.Pages, error) { return nil, errBoom }
	if _, err := TabList(ctx); err == nil {
		t.Error("TabList expected wrapped error")
	}
	if _, err := TabClose(ctx, 0); err == nil {
		t.Error("TabClose Pages expected error")
	}
	if _, err := TabSelect(ctx, 0); err == nil {
		t.Error("TabSelect Pages expected error")
	}
	browserPages = origPages

	// TabClose: Close failure (Pages succeeds, index valid)
	origClose := pageClose
	pageClose = func(p *rod.Page) error { return errBoom }
	if _, err := TabClose(ctx, 0); err == nil {
		t.Error("TabClose Close expected wrapped error")
	}
	pageClose = origClose

	// TabSelect: Activate failure
	origAct := pageActivate
	pageActivate = func(p *rod.Page) (*rod.Page, error) { return nil, errBoom }
	if _, err := TabSelect(ctx, 0); err == nil {
		t.Error("TabSelect Activate expected wrapped error")
	}
	pageActivate = origAct
}

// ---------------------------------------------------------------------------
// Cookies: GetCookies, ClearCookies, SetCookie, DeleteCookie
// ---------------------------------------------------------------------------

func TestErrPaths_Cookies(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	origGet := browserGetCookies
	browserGetCookies = func(b *rod.Browser) ([]*proto.NetworkCookie, error) { return nil, errBoom }
	if _, err := GetCookies(ctx); err == nil {
		t.Error("GetCookies expected wrapped error")
	}
	browserGetCookies = origGet

	origClear := browserSetCookies
	browserSetCookies = func(b *rod.Browser, cookies []*proto.NetworkCookieParam) error { return errBoom }
	if _, err := ClearCookies(ctx); err == nil {
		t.Error("ClearCookies expected wrapped error")
	}
	browserSetCookies = origClear

	origSet := pageSetCookies
	pageSetCookies = func(p *rod.Page, cookies []*proto.NetworkCookieParam) error { return errBoom }
	if _, err := SetCookie(ctx, "n", "v"); err == nil {
		t.Error("SetCookie expected wrapped error")
	}
	pageSetCookies = origSet

	origDel := networkDeleteCookies
	networkDeleteCookies = func(req proto.NetworkDeleteCookies, page *rod.Page) error { return errBoom }
	if _, err := DeleteCookie(ctx, "n"); err == nil {
		t.Error("DeleteCookie expected wrapped error")
	}
	networkDeleteCookies = origDel
}

// ---------------------------------------------------------------------------
// EvalStorage failure path (delegates to Evaluate; inject pageEval failure)
// ---------------------------------------------------------------------------

func TestErrPaths_EvalStorage(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	orig := runtimeEvaluateCall
	runtimeEvaluateCall = func(req proto.RuntimeEvaluate, page *rod.Page) (*proto.RuntimeEvaluateResult, error) {
		return nil, errBoom
	}
	defer func() { runtimeEvaluateCall = orig }()
	if _, err := EvalStorage(ctx, "localStorage", "get", "k", ""); err == nil {
		t.Error("EvalStorage expected wrapped error")
	}
}

// ---------------------------------------------------------------------------
// StateSave / StateLoad: op-level error injection
// ---------------------------------------------------------------------------

func TestErrPaths_State(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	dir := t.TempDir()
	statePath := filepath.Join(dir, "s.json")

	// StateSave: GetCookies failure
	origGet := browserGetCookies
	browserGetCookies = func(b *rod.Browser) ([]*proto.NetworkCookie, error) { return nil, errBoom }
	if _, err := StateSave(ctx, statePath); err == nil {
		t.Error("StateSave GetCookies expected error")
	}
	browserGetCookies = origGet

	// StateSave: WriteFile failure
	origWrite := osWriteFile
	osWriteFile = func(name string, data []byte, perm os.FileMode) error { return errBoom }
	if _, err := StateSave(ctx, statePath); err == nil {
		t.Error("StateSave WriteFile expected error")
	}
	osWriteFile = origWrite

	// Create a valid state file to load.
	if _, err := StateSave(ctx, statePath); err != nil {
		t.Fatalf("StateSave: %v", err)
	}

	// StateLoad: SetCookies failure
	origSet := pageSetCookies
	pageSetCookies = func(p *rod.Page, cookies []*proto.NetworkCookieParam) error { return errBoom }
	if _, err := StateLoad(ctx, statePath); err == nil {
		t.Error("StateLoad SetCookies expected error")
	}
	pageSetCookies = origSet
}

// ---------------------------------------------------------------------------
// Show annotate: page.Eval failure
// ---------------------------------------------------------------------------

func TestErrPaths_ShowAnnotate(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	orig := pageEval
	pageEval = func(p *rod.Page, js string, args ...interface{}) (*proto.RuntimeRemoteObject, error) {
		return nil, errBoom
	}
	defer func() { pageEval = orig }()
	if _, err := Show(ctx, true); err == nil {
		t.Error("Show annotate expected wrapped error")
	}
}

// ---------------------------------------------------------------------------
// PluginList: json.Marshal failure
// ---------------------------------------------------------------------------

func TestErrPaths_PluginListMarshal(t *testing.T) {
	ctx := freshCtx()
	dir := t.TempDir()
	pluginPath := filepath.Join(dir, "p.js")
	_ = writeFile(pluginPath, "function f(){return '1'}")
	if _, err := PluginLoad(ctx, pluginPath); err != nil {
		t.Fatal(err)
	}

	orig := jsonMarshal
	jsonMarshal = func(v interface{}) ([]byte, error) { return nil, errBoom }
	defer func() { jsonMarshal = orig }()
	if _, err := PluginList(ctx); err == nil {
		t.Error("PluginList expected marshal error")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// withTempCwd chdirs into a fresh temp dir for the duration of fn, restoring
// the original working directory afterward.
func withTempCwd(t *testing.T, fn func(dir string)) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()
	fn(dir)
}
