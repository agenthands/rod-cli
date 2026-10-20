package actions

import (
	"testing"

	"github.com/agenthands/rod-cli/types"
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
