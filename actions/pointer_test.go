package actions

import (
	"fmt"
	"testing"

	"github.com/agenthands/godoll/humanize"
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
