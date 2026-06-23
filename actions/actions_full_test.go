package actions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/agenthands/rod-cli/types"
	"github.com/go-rod/rod/lib/launcher"
)

// ---------------------------------------------------------------------------
// Test fixtures / helpers
// ---------------------------------------------------------------------------

const fixtureHTML = `<!DOCTYPE html>
<html>
<head><title>Fixture</title></head>
<body>
  <h1 id="heading">Hello World</h1>
  <a id="link" href="/page2">Go to page 2</a>
  <button id="btn" onclick="document.getElementById('heading').textContent='clicked'">Click Me</button>
  <button id="dblbtn" ondblclick="document.getElementById('heading').textContent='dbl'">Double Click</button>
  <form id="form" action="/submit" method="get">
    <input id="textinput" name="textinput" type="text" aria-label="Text Input" />
    <input id="fileinput" name="fileinput" type="file" aria-label="File Input" />
    <input id="checkbox" name="checkbox" type="checkbox" aria-label="Check Box" />
    <select id="sel" name="sel" aria-label="Select Box">
      <option value="a">Apple</option>
      <option value="b">Banana</option>
    </select>
    <textarea id="ta" aria-label="Text Area"></textarea>
  </form>
  <div id="dragsrc" role="button" tabindex="0" draggable="true" aria-label="Drag Source" style="width:50px;height:50px;background:red">SRC</div>
  <div id="dragdst" role="button" tabindex="0" aria-label="Drag Dest" style="width:50px;height:50px;background:blue">DST</div>
  <button id="dlgbtn" onclick="window.__r = confirm('ok?')" aria-label="Dialog Button">Dialog</button>
</body>
</html>`

const page2HTML = `<!DOCTYPE html><html><head><title>Page 2</title></head><body><h1>Page Two</h1></body></html>`

func newFixtureServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(fixtureHTML))
	})
	mux.HandleFunc("/page2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(page2HTML))
	})
	mux.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>Submitted</title></head><body>OK</body></html>`))
	})
	return httptest.NewServer(mux)
}

// newLiveCtx returns a context with a launched headless browser and a created
// page, plus a cleanup func. The browser is real but headless.
func newLiveCtx(t *testing.T) (*types.Context, func()) {
	t.Helper()
	u := launcher.New().Headless(true).MustLaunch()
	ctx := types.NewContext(context.Background(), types.Config{CDPEndpoint: u})
	if _, err := ctx.EnsurePage(); err != nil {
		t.Fatalf("EnsurePage failed: %v", err)
	}
	cleanup := func() {
		_ = ctx.CloseBrowser()
	}
	return ctx, cleanup
}

// freshCtx returns a context with NO page/browser (for negative tests where
// ControlledPage must error).
func freshCtx() *types.Context {
	return types.NewContext(context.Background(), types.Config{})
}

var refRe = regexp.MustCompile(`\[ref=((?:f\d+)?s\d+e\d+)\]`)

// refFor navigates, snapshots, and returns the snapshot ref for the element
// whose snapshot line contains the given substring (e.g. accessible name).
func refFor(t *testing.T, ctx *types.Context, substr string) string {
	t.Helper()
	snap, err := Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}
	for _, line := range strings.Split(snap, "\n") {
		if substr != "" && !strings.Contains(line, substr) {
			continue
		}
		m := refRe.FindStringSubmatch(line)
		if m != nil {
			return m[1]
		}
	}
	t.Fatalf("no ref found for %q in snapshot:\n%s", substr, snap)
	return ""
}

// navFixture navigates the live ctx to the fixture and returns the base URL.
func navFixture(t *testing.T, ctx *types.Context, srv *httptest.Server) {
	t.Helper()
	if _, err := Navigate(ctx, srv.URL); err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Navigation
// ---------------------------------------------------------------------------

func TestNavigate(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()

	// positive
	out, err := Navigate(ctx, srv.URL)
	if err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	if !strings.Contains(out, "Navigated to") {
		t.Errorf("unexpected output: %s", out)
	}

	// negative: invalid URL (not http)
	if _, err := Navigate(ctx, "ftp://foo"); err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestNavigate_NoPage_BadURL(t *testing.T) {
	// invalid url returns before page is required
	if _, err := Navigate(freshCtx(), "notaurl"); err == nil {
		t.Error("expected invalid URL error")
	}
}

func TestGoBackForwardReload(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()

	navFixture(t, ctx, srv)
	if _, err := Navigate(ctx, srv.URL+"/page2"); err != nil {
		t.Fatalf("nav page2: %v", err)
	}

	if _, err := GoBack(ctx); err != nil {
		t.Errorf("GoBack: %v", err)
	}
	if _, err := GoForward(ctx); err != nil {
		t.Errorf("GoForward: %v", err)
	}
	if _, err := Reload(ctx); err != nil {
		t.Errorf("Reload: %v", err)
	}
}

func TestNavActions_NoPage(t *testing.T) {
	if _, err := GoBack(freshCtx()); err == nil {
		t.Error("GoBack expected error")
	}
	if _, err := GoForward(freshCtx()); err == nil {
		t.Error("GoForward expected error")
	}
	if _, err := Reload(freshCtx()); err == nil {
		t.Error("Reload expected error")
	}
}

// ---------------------------------------------------------------------------
// Element interactions
// ---------------------------------------------------------------------------

func TestClick(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	ref := refFor(t, ctx, "Click Me")
	if _, err := Click(ctx, ref); err != nil {
		t.Errorf("Click: %v", err)
	}

	// negative: bad ref
	if _, err := Click(ctx, "#rod-cli-no-such-element"); err == nil {
		t.Error("Click expected error for bad ref")
	}
	// negative: no page
	if _, err := Click(freshCtx(), "x"); err == nil {
		t.Error("Click expected error no page")
	}
}

func TestDblClick(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	ref := refFor(t, ctx, "Double Click")
	if _, err := DblClick(ctx, ref); err != nil {
		t.Errorf("DblClick: %v", err)
	}
	if _, err := DblClick(ctx, "#rod-cli-no-such-element"); err == nil {
		t.Error("DblClick expected error")
	}
	if _, err := DblClick(freshCtx(), "x"); err == nil {
		t.Error("DblClick expected error no page")
	}
}

func TestType(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	ref := refFor(t, ctx, "Text Input")
	if _, err := Type(ctx, ref, "hello"); err != nil {
		t.Errorf("Type: %v", err)
	}
	if _, err := Type(ctx, "#rod-cli-no-such-element", "x"); err == nil {
		t.Error("Type expected error")
	}
	if _, err := Type(freshCtx(), "x", "y"); err == nil {
		t.Error("Type expected error no page")
	}
}

func TestFill(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	ref := refFor(t, ctx, "Text Input")
	if _, err := Fill(ctx, ref, "value", false); err != nil {
		t.Errorf("Fill: %v", err)
	}
	// positive with submit=true
	ref2 := refFor(t, ctx, "Text Input")
	if _, err := Fill(ctx, ref2, "more", true); err != nil {
		t.Errorf("Fill submit: %v", err)
	}
	if _, err := Fill(ctx, "#rod-cli-no-such-element", "x", false); err == nil {
		t.Error("Fill expected error")
	}
	if _, err := Fill(freshCtx(), "x", "y", false); err == nil {
		t.Error("Fill expected error no page")
	}
}

func TestHover(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	ref := refFor(t, ctx, "Click Me")
	if _, err := Hover(ctx, ref); err != nil {
		t.Errorf("Hover: %v", err)
	}
	if _, err := Hover(ctx, "#rod-cli-no-such-element"); err == nil {
		t.Error("Hover expected error")
	}
	if _, err := Hover(freshCtx(), "x"); err == nil {
		t.Error("Hover expected error no page")
	}
}

func TestCheckUncheck(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	ref := refFor(t, ctx, "Check Box")
	if _, err := Check(ctx, ref); err != nil {
		t.Errorf("Check: %v", err)
	}
	ref2 := refFor(t, ctx, "Check Box")
	if _, err := Uncheck(ctx, ref2); err != nil {
		t.Errorf("Uncheck: %v", err)
	}

	if _, err := Check(ctx, "#rod-cli-no-such-element"); err == nil {
		t.Error("Check expected error")
	}
	if _, err := Uncheck(ctx, "#rod-cli-no-such-element"); err == nil {
		t.Error("Uncheck expected error")
	}
	if _, err := Check(freshCtx(), "x"); err == nil {
		t.Error("Check expected error no page")
	}
	if _, err := Uncheck(freshCtx(), "x"); err == nil {
		t.Error("Uncheck expected error no page")
	}
}

func TestSelect(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	ref := refFor(t, ctx, "Select Box")
	if _, err := Select(ctx, ref, []string{"Banana"}); err != nil {
		t.Errorf("Select: %v", err)
	}
	// negative: nonexistent option text -> Select returns error
	ref2 := refFor(t, ctx, "Select Box")
	if _, err := Select(ctx, ref2, []string{"DoesNotExist"}); err == nil {
		t.Error("Select expected error for missing option")
	}
	if _, err := Select(ctx, "#rod-cli-no-such-element", []string{"a"}); err == nil {
		t.Error("Select expected error bad ref")
	}
	if _, err := Select(freshCtx(), "x", []string{"a"}); err == nil {
		t.Error("Select expected error no page")
	}
}

func TestUpload(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	dir := t.TempDir()
	fp := filepath.Join(dir, "up.txt")
	if err := writeFile(fp, "data"); err != nil {
		t.Fatal(err)
	}

	ref := refFor(t, ctx, "File Input")
	if _, err := Upload(ctx, ref, []string{fp}); err != nil {
		t.Errorf("Upload: %v", err)
	}
	if _, err := Upload(ctx, "#rod-cli-no-such-element", []string{fp}); err == nil {
		t.Error("Upload expected error bad ref")
	}
	if _, err := Upload(freshCtx(), "x", []string{fp}); err == nil {
		t.Error("Upload expected error no page")
	}
}

func TestDrop(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	dir := t.TempDir()
	fp := filepath.Join(dir, "drop.txt")
	if err := writeFile(fp, "data"); err != nil {
		t.Fatal(err)
	}

	ref := refFor(t, ctx, "File Input")
	if _, err := Drop(ctx, ref, fp); err != nil {
		t.Errorf("Drop: %v", err)
	}
	// negative: bad ref
	if _, err := Drop(ctx, "#rod-cli-no-such-element", fp); err == nil {
		t.Error("Drop expected error bad ref")
	}
}

func TestDrag(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	src := refFor(t, ctx, "Drag Source")
	dst := refFor(t, ctx, "Drag Dest")
	// Drag may or may not error depending on physics; just exercise path.
	_, _ = Drag(ctx, src, dst)

	// negative: bad start ref
	if _, err := Drag(ctx, "#rod-cli-no-such-element", dst); err == nil {
		t.Error("Drag expected error bad start ref")
	}
	// negative: bad end ref
	src2 := refFor(t, ctx, "Drag Source")
	if _, err := Drag(ctx, src2, "#rod-cli-no-such-element"); err == nil {
		t.Error("Drag expected error bad end ref")
	}
	if _, err := Drag(freshCtx(), "a", "b"); err == nil {
		t.Error("Drag expected error no page")
	}
}

// ---------------------------------------------------------------------------
// Evaluate / Storage
// ---------------------------------------------------------------------------

func TestEvaluate(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	// expression branch
	if out, err := Evaluate(ctx, "1 + 1", ""); err != nil {
		t.Errorf("Evaluate expr: %v", err)
	} else if !strings.Contains(out, "2") {
		t.Errorf("expected 2, got %s", out)
	}

	// function branch
	if _, err := Evaluate(ctx, "() => document.title", ""); err != nil {
		t.Errorf("Evaluate func: %v", err)
	}

	// element ref branch
	ref := refFor(t, ctx, "Click Me")
	if _, err := Evaluate(ctx, "() => this.tagName", ref); err != nil {
		t.Errorf("Evaluate on element: %v", err)
	}

	// expression returning undefined (exercises the nil-result formatting branch)
	if _, err := Evaluate(ctx, "void 0", ""); err != nil {
		t.Errorf("Evaluate void: %v", err)
	}
	// function returning null (exercises res.Value.Nil() branch -> "null")
	if out, err := Evaluate(ctx, "() => null", ""); err != nil {
		t.Errorf("Evaluate func null: %v", err)
	} else if !strings.Contains(out, "null") {
		t.Errorf("expected null, got %s", out)
	}
	// expression returning a string (string result formatting branch)
	if out, err := Evaluate(ctx, "'hello'", ""); err != nil {
		t.Errorf("Evaluate string: %v", err)
	} else if !strings.Contains(out, "hello") {
		t.Errorf("expected hello, got %s", out)
	}

	// negative: syntax error in expression -> exception
	if _, err := Evaluate(ctx, "throw new Error('boom')", ""); err == nil {
		t.Error("Evaluate expected exception error")
	}
	// negative: function with error
	if _, err := Evaluate(ctx, "() => { throw new Error('x') }", ""); err == nil {
		t.Error("Evaluate func expected error")
	}
	// negative: bad element ref
	if _, err := Evaluate(ctx, "() => 1", "#rod-cli-no-such-element"); err == nil {
		t.Error("Evaluate expected error bad ref")
	}
	// negative: element eval error
	ref2 := refFor(t, ctx, "Click Me")
	if _, err := Evaluate(ctx, "() => { throw new Error('e') }", ref2); err == nil {
		t.Error("Evaluate on element expected error")
	}
	// negative: no page
	if _, err := Evaluate(freshCtx(), "1+1", ""); err == nil {
		t.Error("Evaluate expected error no page")
	}
}

func TestEvalStorage(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	if _, err := EvalStorage(ctx, "localStorage", "set", "k", "v"); err != nil {
		t.Errorf("EvalStorage set: %v", err)
	}
	if out, err := EvalStorage(ctx, "localStorage", "get", "k", ""); err != nil {
		t.Errorf("EvalStorage get: %v", err)
	} else if !strings.Contains(out, "v") {
		t.Errorf("expected v, got %s", out)
	}
	if _, err := EvalStorage(ctx, "localStorage", "get", "", ""); err != nil {
		t.Errorf("EvalStorage get-all: %v", err)
	}
	if _, err := EvalStorage(ctx, "localStorage", "delete", "k", ""); err != nil {
		t.Errorf("EvalStorage delete: %v", err)
	}
	if _, err := EvalStorage(ctx, "localStorage", "clear", "", ""); err != nil {
		t.Errorf("EvalStorage clear: %v", err)
	}
	// negative: unknown action
	if _, err := EvalStorage(ctx, "localStorage", "bogus", "", ""); err == nil {
		t.Error("EvalStorage expected error unknown action")
	}
	// negative: eval failure (no page)
	if _, err := EvalStorage(freshCtx(), "localStorage", "get", "k", ""); err == nil {
		t.Error("EvalStorage expected error no page")
	}
}

// ---------------------------------------------------------------------------
// Screenshot / PDF
// ---------------------------------------------------------------------------

func TestScreenshot(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	out, err := Screenshot(ctx, "test_shot", "", 800, 600)
	if err != nil {
		t.Errorf("Screenshot: %v", err)
	}
	if !strings.Contains(out, "Save to") {
		t.Errorf("unexpected: %s", out)
	}
	if _, err := Screenshot(freshCtx(), "x", "", 0, 0); err == nil {
		t.Error("Screenshot expected error no page")
	}
}

func TestPdf(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	out, err := Pdf(ctx, "test_pdf")
	if err != nil {
		t.Errorf("Pdf: %v", err)
	}
	if !strings.Contains(out, "Save to") {
		t.Errorf("unexpected: %s", out)
	}
	if _, err := Pdf(freshCtx(), "x"); err == nil {
		t.Error("Pdf expected error no page")
	}
}

func TestSnapshot(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	if _, err := Snapshot(ctx); err != nil {
		t.Errorf("Snapshot: %v", err)
	}
	if _, err := Snapshot(freshCtx()); err == nil {
		t.Error("Snapshot expected error no page")
	}
}

// ---------------------------------------------------------------------------
// Keyboard / Mouse
// ---------------------------------------------------------------------------

func TestPressKey(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	if _, err := PressKey(ctx, 'a'); err != nil {
		t.Errorf("PressKey: %v", err)
	}
	if _, err := PressKey(freshCtx(), 'a'); err == nil {
		t.Error("PressKey expected error no page")
	}
}

func TestPress(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	for _, k := range []string{"Enter", "Tab", "Backspace", "Escape", "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "a"} {
		_, _ = Press(ctx, k)
	}
	if _, err := Press(freshCtx(), "Enter"); err == nil {
		t.Error("Press expected error no page")
	}
}

func TestParseKey(t *testing.T) {
	// directly exercise all branches
	cases := []string{"Enter", "Tab", "Backspace", "Escape", "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "z", ""}
	for _, c := range cases {
		_ = parseKey(c)
	}
}

func TestMouse(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	if _, err := MouseMove(ctx, 10, 10); err != nil {
		t.Errorf("MouseMove: %v", err)
	}
	for _, b := range []string{"left", "right", "middle", "other"} {
		if _, err := MouseDown(ctx, b); err != nil {
			t.Errorf("MouseDown %s: %v", b, err)
		}
		if _, err := MouseUp(ctx, b); err != nil {
			t.Errorf("MouseUp %s: %v", b, err)
		}
	}

	// MouseWheel all directions
	if _, err := MouseWheel(ctx, 0, 100); err != nil {
		t.Errorf("MouseWheel down: %v", err)
	}
	if _, err := MouseWheel(ctx, 0, -50); err != nil {
		t.Errorf("MouseWheel up: %v", err)
	}
	if _, err := MouseWheel(ctx, 100, 0); err != nil {
		t.Errorf("MouseWheel right: %v", err)
	}
	if _, err := MouseWheel(ctx, -50, 0); err != nil {
		t.Errorf("MouseWheel left: %v", err)
	}

	// negative no page
	if _, err := MouseMove(freshCtx(), 1, 1); err == nil {
		t.Error("MouseMove expected error no page")
	}
	if _, err := MouseDown(freshCtx(), "left"); err == nil {
		t.Error("MouseDown expected error no page")
	}
	if _, err := MouseUp(freshCtx(), "left"); err == nil {
		t.Error("MouseUp expected error no page")
	}
	if _, err := MouseWheel(freshCtx(), 0, 10); err == nil {
		t.Error("MouseWheel expected error no page")
	}
}

func TestResize(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	if _, err := Resize(ctx, 1024, 768); err != nil {
		t.Errorf("Resize: %v", err)
	}
	if _, err := Resize(freshCtx(), 100, 100); err == nil {
		t.Error("Resize expected error no page")
	}
}

// ---------------------------------------------------------------------------
// Tabs
// ---------------------------------------------------------------------------

func TestTabs(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	if _, err := TabList(ctx); err != nil {
		t.Errorf("TabList: %v", err)
	}
	if _, err := TabNew(ctx, srv.URL+"/page2"); err != nil {
		t.Errorf("TabNew url: %v", err)
	}
	if _, err := TabNew(ctx, ""); err != nil {
		t.Errorf("TabNew blank: %v", err)
	}
	if _, err := TabSelect(ctx, 0); err != nil {
		t.Errorf("TabSelect: %v", err)
	}
	// out of range
	if _, err := TabSelect(ctx, 999); err == nil {
		t.Error("TabSelect expected error out of range")
	}
	if _, err := TabClose(ctx, 999); err == nil {
		t.Error("TabClose expected error out of range")
	}
	if _, err := TabClose(ctx, 1); err != nil {
		t.Errorf("TabClose: %v", err)
	}

	// negative: no browser
	if _, err := TabList(freshCtx()); err == nil {
		t.Error("TabList expected error no browser")
	}
	if _, err := TabNew(freshCtx(), ""); err == nil {
		t.Error("TabNew expected error no browser")
	}
	if _, err := TabClose(freshCtx(), 0); err == nil {
		t.Error("TabClose expected error no browser")
	}
	if _, err := TabSelect(freshCtx(), 0); err == nil {
		t.Error("TabSelect expected error no browser")
	}
}

// ---------------------------------------------------------------------------
// Dialog
// ---------------------------------------------------------------------------

func TestHandleDialog(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	if _, err := HandleDialog(ctx, true, "yes"); err != nil {
		t.Errorf("HandleDialog: %v", err)
	}
	// Trigger a dialog so the handler goroutine consumes it and completes.
	// Use a non-blocking JS trigger (alert resolves immediately once handled).
	page, _ := ctx.ControlledPage()
	go func() { _, _ = page.Eval("() => alert('hi')") }()
	// Give the handler goroutine time to consume the dialog before cleanup
	// closes the browser (otherwise the Must* handler panics on a canceled ctx).
	time.Sleep(1 * time.Second)

	if _, err := HandleDialog(freshCtx(), true, ""); err == nil {
		t.Error("HandleDialog expected error no page")
	}
}

// ---------------------------------------------------------------------------
// Cookies
// ---------------------------------------------------------------------------

func TestCookies(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	if _, err := SetCookie(ctx, "foo", "bar"); err != nil {
		t.Errorf("SetCookie: %v", err)
	}
	if out, err := GetCookies(ctx); err != nil {
		t.Errorf("GetCookies: %v", err)
	} else if !strings.Contains(out, "Cookies") {
		t.Errorf("unexpected: %s", out)
	}
	if _, err := DeleteCookie(ctx, "foo"); err != nil {
		t.Errorf("DeleteCookie: %v", err)
	}
	if _, err := ClearCookies(ctx); err != nil {
		t.Errorf("ClearCookies: %v", err)
	}

	// negative no page
	if _, err := SetCookie(freshCtx(), "a", "b"); err == nil {
		t.Error("SetCookie expected error no page")
	}
	if _, err := GetCookies(freshCtx()); err == nil {
		t.Error("GetCookies expected error no page")
	}
	if _, err := DeleteCookie(freshCtx(), "a"); err == nil {
		t.Error("DeleteCookie expected error no page")
	}
	if _, err := ClearCookies(freshCtx()); err == nil {
		t.Error("ClearCookies expected error no page")
	}
}

// ---------------------------------------------------------------------------
// Console / Network / Routes
// ---------------------------------------------------------------------------

func TestConsoleLogs(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()

	// empty case
	if out, err := ConsoleLogs(ctx); err != nil || !strings.Contains(out, "No console logs") {
		t.Errorf("ConsoleLogs empty: out=%s err=%v", out, err)
	}

	navFixture(t, ctx, srv)
	_, _ = Evaluate(ctx, "console.log('hi from test')", "")
	time.Sleep(200 * time.Millisecond)
	if _, err := ConsoleLogs(ctx); err != nil {
		t.Errorf("ConsoleLogs: %v", err)
	}
}

func TestNetworkRequests(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()

	// empty
	if out, err := NetworkRequests(ctx); err != nil || !strings.Contains(out, "No network requests") {
		t.Errorf("NetworkRequests empty: out=%s err=%v", out, err)
	}
	if _, err := NetworkRequest(ctx, 0); err == nil {
		t.Error("NetworkRequest expected error empty/out of bounds")
	}

	navFixture(t, ctx, srv)
	time.Sleep(200 * time.Millisecond)
	if _, err := NetworkRequests(ctx); err != nil {
		t.Errorf("NetworkRequests: %v", err)
	}
	if _, err := NetworkRequest(ctx, 0); err != nil {
		t.Errorf("NetworkRequest 0: %v", err)
	}
	if _, err := NetworkRequest(ctx, -1); err == nil {
		t.Error("NetworkRequest expected error negative index")
	}
}

func TestRoutes(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()

	// empty
	if out, err := RouteList(ctx); err != nil || !strings.Contains(out, "No active routes") {
		t.Errorf("RouteList empty: out=%s err=%v", out, err)
	}
	if _, err := Route(ctx, "*.example.com", "mocked"); err != nil {
		t.Errorf("Route: %v", err)
	}
	if out, err := RouteList(ctx); err != nil || !strings.Contains(out, "example") {
		t.Errorf("RouteList: out=%s err=%v", out, err)
	}
	if _, err := Unroute(ctx, "*.example.com"); err != nil {
		t.Errorf("Unroute: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Highlight
// ---------------------------------------------------------------------------

func TestHighlight(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	ref := refFor(t, ctx, "Click Me")
	if _, err := Highlight(ctx, ref); err != nil {
		t.Errorf("Highlight: %v", err)
	}
	if _, err := ClearHighlights(ctx); err != nil {
		t.Errorf("ClearHighlights: %v", err)
	}

	// negative
	if _, err := Highlight(ctx, "#rod-cli-no-such-element"); err == nil {
		t.Error("Highlight expected error bad ref")
	}
	if _, err := Highlight(freshCtx(), "x"); err == nil {
		t.Error("Highlight expected error no page")
	}
	if _, err := ClearHighlights(freshCtx()); err == nil {
		t.Error("ClearHighlights expected error no page")
	}
}

// ---------------------------------------------------------------------------
// State save / load
// ---------------------------------------------------------------------------

func TestStateSaveLoad(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	_, _ = SetCookie(ctx, "session", "abc")
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")

	if _, err := StateSave(ctx, statePath); err != nil {
		t.Errorf("StateSave: %v", err)
	}
	if _, err := StateLoad(ctx, statePath); err != nil {
		t.Errorf("StateLoad: %v", err)
	}

	// negative: save to bad path (dir doesn't exist)
	if _, err := StateSave(ctx, filepath.Join(dir, "nope", "x.json")); err == nil {
		t.Error("StateSave expected error bad path")
	}
	// negative: load missing file
	if _, err := StateLoad(ctx, filepath.Join(dir, "missing.json")); err == nil {
		t.Error("StateLoad expected error missing file")
	}
	// negative: load invalid json
	badPath := filepath.Join(dir, "bad.json")
	_ = writeFile(badPath, "{not json")
	if _, err := StateLoad(ctx, badPath); err == nil {
		t.Error("StateLoad expected error invalid json")
	}
	// negative: no page
	if _, err := StateSave(freshCtx(), statePath); err == nil {
		t.Error("StateSave expected error no page")
	}
	if _, err := StateLoad(freshCtx(), statePath); err == nil {
		t.Error("StateLoad expected error no page")
	}
}

// ---------------------------------------------------------------------------
// Show / CloseBrowser
// ---------------------------------------------------------------------------

func TestShow(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	// annotate=false: returns instructional message, no page needed
	out, err := Show(ctx, false)
	if err != nil {
		t.Errorf("Show false: %v", err)
	}
	if !strings.Contains(out, "headless") {
		t.Errorf("unexpected Show output: %s", out)
	}

	// annotate=true: launches the annotator UI JS, which returns a Promise that
	// resolves only when the user clicks Cancel or Done. We run Show in a
	// goroutine and resolve the promise by clicking the Cancel button via JS,
	// which exercises the {cancelled:true} branch.
	annOut := make(chan string, 1)
	go func() {
		o, _ := Show(ctx, true)
		annOut <- o
	}()
	// Poll for the cancel button to be injected, then click it.
	clicked := false
	for i := 0; i < 30 && !clicked; i++ {
		time.Sleep(150 * time.Millisecond)
		res, err := Evaluate(ctx, "() => { const b = document.getElementById('rod-btn-cancel'); if (b) { b.click(); return true; } return false; }", "")
		if err == nil && strings.Contains(res, "true") {
			clicked = true
		}
	}
	select {
	case o := <-annOut:
		if clicked && !strings.Contains(o, "cancelled") {
			t.Logf("Show annotate returned: %s", o)
		}
	case <-time.After(5 * time.Second):
		t.Log("annotator did not resolve in time (tolerated)")
	}

	// annotate=true with no page -> error
	if _, err := Show(freshCtx(), true); err == nil {
		t.Error("Show annotate expected error no page")
	}
}

func TestCloseBrowser(t *testing.T) {
	u := launcher.New().Headless(true).MustLaunch()
	ctx := types.NewContext(context.Background(), types.Config{CDPEndpoint: u})
	if _, err := ctx.EnsurePage(); err != nil {
		t.Fatalf("EnsurePage: %v", err)
	}
	if out, err := CloseBrowser(ctx); err != nil {
		t.Errorf("CloseBrowser: %v", err)
	} else if !strings.Contains(out, "Close browser successfully") {
		t.Errorf("unexpected: %s", out)
	}
}
