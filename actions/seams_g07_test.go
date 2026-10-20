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
