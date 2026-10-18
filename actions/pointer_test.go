package actions

import (
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
