package actions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/agenthands/rod-cli/types"
	"pgregory.net/rapid"
)

// liveHTML returns a live ctx whose page holds the given body HTML.
func liveHTML(t *testing.T, html string) (*types.Context, func()) {
	t.Helper()
	ctx, cleanup := newLiveCtx(t)
	page, err := ctx.ControlledPage()
	if err != nil {
		cleanup()
		t.Fatalf("ControlledPage: %v", err)
	}
	if err := page.SetDocumentContent(html); err != nil {
		cleanup()
		t.Fatalf("SetDocumentContent: %v", err)
	}
	return ctx, cleanup
}

// TestSelectPrefersExactTextThenValue pins actions:A3 / actions:selectOne:
// exact option text wins, then value, then case-insensitive substring.
func TestSelectPrefersExactTextThenValue(t *testing.T) {
	ctx, cleanup := liveHTML(t, `<select aria-label="Items"><option value="10">Item 10</option><option value="1">Item 1</option><option value="b">Banana</option></select>`)
	defer cleanup()
	ref := refFor(t, ctx, "Items")
	for in, want := range map[string]string{"Item 1": "Item 1", "1": "Item 1", "ban": "Banana", "Item 10": "Item 10"} {
		out, err := Select(ctx, ref, []string{in})
		if exp := `Selected "` + want + `" in ` + ref; err != nil || out != exp {
			t.Errorf("Select(%q) = %q, %v; want %q", in, out, err, exp)
		}
	}
}

// TestSelectByValueEscapesForCSS pins actions:selectByValue: an option value
// holding a tab (or a quote or backslash) is still selectable by value.
func TestSelectByValueEscapesForCSS(t *testing.T) {
	ctx, cleanup := liveHTML(t, `<select aria-label="Odd"><option value="x">Plain</option><option value="a&#9;b">Tabbed</option><option value="q&quot;\z">Quoted</option></select>`)
	defer cleanup()
	ref := refFor(t, ctx, "Odd")
	for in, want := range map[string]string{"a\tb": "Tabbed", `q"\z`: "Quoted"} {
		out, err := Select(ctx, ref, []string{in})
		if exp := `Selected "` + want + `" in ` + ref; err != nil || out != exp {
			t.Errorf("Select(%q) = %q, %v; want %q", in, out, err, exp)
		}
	}
}

// TestEvaluateRunsFunctionsAndStatements pins actions:A6 / actions:evaluate:
// statements run as a script, any function-valued source is called, and a
// result is never printed as "<nil>".
func TestEvaluateRunsFunctionsAndStatements(t *testing.T) {
	ctx, cleanup := liveHTML(t, `<button aria-label="Btn">b</button>`)
	defer cleanup()
	for _, c := range []struct{ src, want string }{
		{"function f(){return 5}; f()", "5"},
		{"async function(){return 7}", "7"},
		{"(a=1) => a+1", "2"},
		{"() => document.title", ""},
		{"1 + 1", "2"},
		{"void 0", "undefined"},
		{"({a:1})", `{"a":1}`},
	} {
		if got, err := Evaluate(ctx, c.src, ""); err != nil || got != c.want {
			t.Errorf("Evaluate(%q) = %q, %v; want %q", c.src, got, err, c.want)
		}
	}
	ref := refFor(t, ctx, "Btn")
	if got, err := Evaluate(ctx, "() => null", ref); err != nil || got != "null" {
		t.Errorf("Evaluate(() => null, ref) = %q, %v; want null", got, err)
	}
}

// TestStateSaveLoadRoundTripsCookiesPrivately pins actions:A1 /
// actions:state-0644: save, clear, load brings back a session and a persistent
// cookie, and the saved jar is readable by the owner only.
func TestStateSaveLoadRoundTripsCookiesPrivately(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()
	if _, err := Evaluate(ctx, `document.cookie = "sess=1"; document.cookie = "pers=2; max-age=3600"`, ""); err != nil {
		t.Fatalf("set cookies: %v", err)
	}
	path := filepath.Join(t.TempDir(), "state.json")
	if _, err := StateSave(ctx, path); err != nil {
		t.Fatalf("StateSave: %v", err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Mode().Perm()&0o077 != 0 {
		t.Fatalf("state file mode = %v, %v; want no group/other bits", fi.Mode().Perm(), err)
	}
	page, _ := ctx.ControlledPage()
	if err := page.Browser().SetCookies(nil); err != nil {
		t.Fatalf("clear cookies: %v", err)
	}
	if got, _ := Evaluate(ctx, "document.cookie", ""); got != "" {
		t.Fatalf("cookies after clear = %q; want empty", got)
	}
	if _, err := StateLoad(ctx, path); err != nil {
		t.Fatalf("StateLoad: %v", err)
	}
	got, _ := Evaluate(ctx, "document.cookie", "")
	if !strings.Contains(got, "sess=1") || !strings.Contains(got, "pers=2") {
		t.Fatalf("cookies after load = %q; want sess=1 and pers=2", got)
	}
}

// TestTabNewBadURLLeavesTabsUnchanged pins actions:tabnew-orphan: a URL that
// Navigate would refuse errors before any tab is created or switched to.
func TestTabNewBadURLLeavesTabsUnchanged(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()
	before, _ := ctx.ControlledPage()
	pages, err := ctx.GetBrowser().Pages()
	if err != nil {
		t.Fatalf("Pages: %v", err)
	}
	if _, err := TabNew(ctx, "file:///x"); err == nil {
		t.Fatal("TabNew(file:///x) succeeded; want invalid URL error")
	}
	after, _ := ctx.GetBrowser().Pages()
	active, _ := ctx.ControlledPage()
	if len(after) != len(pages) || active.TargetID != before.TargetID {
		t.Fatalf("tabs %d→%d, active %s→%s; want both unchanged", len(pages), len(after), before.TargetID, active.TargetID)
	}
}

// TestFillValueEqualsText pins actions:A4: with typos on (rate 0.5) and fast
// typing, Fill replaces any existing value with exactly the given text.
// Run with -rapid.checks=30; one browser serves every draw.
func TestFillValueEqualsText(t *testing.T) {
	ctx, cleanup := newLiveCtxCfg(t, func(c *types.Config) {
		rate, lo, hi := float32(0.5), 1, 2
		c.Stealth.TypoRate, c.Stealth.TypingSpeedMin, c.Stealth.TypingSpeedMax = &rate, &lo, &hi
	})
	defer cleanup()
	page, _ := ctx.ControlledPage()
	if err := page.SetDocumentContent(`<input aria-label="Field">`); err != nil {
		t.Fatalf("SetDocumentContent: %v", err)
	}
	ref := refFor(t, ctx, "Field")
	el, err := getElementByRef(ctx, ref)
	if err != nil {
		t.Fatalf("getElementByRef: %v", err)
	}
	str := rapid.StringOfN(rapid.RuneFrom(nil, unicode.L, unicode.N, unicode.P, unicode.S, unicode.Zs), 0, 20, -1)
	rapid.Check(t, func(rt *rapid.T) {
		old, text := str.Draw(rt, "old"), str.Draw(rt, "text")
		if _, err := el.Eval(`v => { this.value = v }`, old); err != nil {
			rt.Fatalf("set old value: %v", err)
		}
		if _, err := Fill(ctx, ref, text, false); err != nil {
			rt.Fatalf("Fill: %v", err)
		}
		got, err := el.Eval(`() => this.value`)
		if err != nil {
			rt.Fatalf("read value: %v", err)
		}
		if got.Value.Str() != text {
			rt.Fatalf("value = %q; want %q", got.Value.Str(), text)
		}
	})
}
