package actions

import (
	"fmt"
	"strings"
	"testing"

	"github.com/agenthands/godoll/humanize"
	"github.com/agenthands/rod-cli/types"
	"github.com/go-rod/rod"
)

func TestMouseMoveTracksPosition(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()
	if _, err := MouseMove(ctx, 120, 340); err != nil {
		t.Fatalf("MouseMove: %v", err)
	}
	page, _ := ctx.ControlledPage()
	if got := humanize.GetMousePos(page); got != (humanize.Point{X: 120, Y: 340}) {
		t.Errorf("GetMousePos = %v, want {120 340}", got)
	}
	if got, err := Evaluate(ctx, "window.__mx+','+window.__my", ""); err != nil || got != "120,340" {
		t.Errorf("page saw %q (err %v), want 120,340", got, err)
	}
}

func TestHoverDoesNotTeleport(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()
	if _, err := Hover(ctx, refFor(t, ctx, "Click Me")); err != nil {
		t.Fatalf("Hover: %v", err)
	}
	page, _ := ctx.ControlledPage()
	p := humanize.GetMousePos(page)
	// Chrome truncates fractional dispatch coords to integer clientX/Y.
	want := fmt.Sprintf("%d,%d", int(p.X), int(p.Y))
	if got, err := Evaluate(ctx, "window.__mx+','+window.__my", ""); err != nil || got != want {
		t.Errorf("page saw %q (err %v), tracked pointer %s", got, err, want)
	}
}

func TestDragHonoursTuning(t *testing.T) {
	steps := 7
	ctx, cleanup := newLiveCtxCfg(t, func(c *types.Config) { c.Stealth.MouseSteps = &steps })
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	var got []int
	orig := moveToElement
	moveToElement = func(p *rod.Page, el *rod.Element, opts ...humanize.MouseOption) error {
		got = append(got, len(opts))
		return nil
	}
	defer func() { moveToElement = orig }()

	// Both refs must come from one snapshot: a second snapshot retires the first's refs.
	snap, err := Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	refs := map[string]string{}
	for _, line := range strings.Split(snap, "\n") {
		for _, name := range []string{"Drag Source", "Drag Dest"} {
			if m := refRe.FindStringSubmatch(line); m != nil && strings.Contains(line, name) {
				refs[name] = m[1]
			}
		}
	}
	if _, err := Drag(ctx, refs["Drag Source"], refs["Drag Dest"]); err != nil {
		t.Fatalf("Drag: %v", err)
	}
	want := len(mouseOpts(ctx.HumanizeTuning()))
	if want == 0 || len(got) != 2 || got[0] != want || got[1] != want {
		t.Errorf("moveToElement option counts = %v, want two moves with %d (non-zero)", got, want)
	}
}
