package actions

import (
	"strings"
	"testing"

	"github.com/agenthands/godoll/humanize"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
)

// TestPressSendsNamedKeyNeverFirstByte pins actions:A2: a named key is sent as
// that key, never as the key for its first byte; an unknown key is an error.
func TestPressSendsNamedKeyNeverFirstByte(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	var sent []input.Key
	orig := keyboardPress
	keyboardPress = func(kb *rod.Keyboard, key input.Key) error { sent = append(sent, key); return nil }
	t.Cleanup(func() { keyboardPress = orig })

	for name, want := range map[string]input.Key{
		"Delete": input.Delete, "Home": input.Home, "End": input.End,
		"PageDown": input.PageDown, "Space": input.Space, "F5": input.F5,
		"Enter": input.Enter, "a": input.KeyA, "A": input.Key('A'), "1": input.Digit1,
	} {
		sent = nil
		if _, err := Press(ctx, name); err != nil || len(sent) != 1 || sent[0] != want {
			t.Errorf("Press(%q): sent %v, err %v; want [%v]", name, sent, err, want)
		}
	}
	for _, name := range []string{"é", "Bogus"} {
		sent = nil
		_, err := Press(ctx, name)
		if err == nil || !strings.Contains(err.Error(), `unknown key "`+name+`"`) || len(sent) != 0 {
			t.Errorf("Press(%q): sent %v, err %v; want unknown key error and nothing sent", name, sent, err)
		}
	}
}

// TestMouseWheelEmptyMeasureIsError pins actions:A5: no scroll boxes is an
// error, never ("", nil) from errors.Wrap(nil).
func TestMouseWheelEmptyMeasureIsError(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()
	origMeasure, origMove := measureScroll, moveToPoint
	measureScroll = func(el *rod.Element) ([]scrollBox, error) { return []scrollBox{}, nil }
	moveToPoint = func(p *rod.Page, pt humanize.Point, opts ...humanize.MouseOption) error { return nil }
	t.Cleanup(func() { measureScroll, moveToPoint = origMeasure, origMove })

	if got, err := MouseWheel(ctx, 0, 0, "", 0); err == nil || !strings.Contains(err.Error(), "Failed to measure scroll") {
		t.Fatalf("MouseWheel = %q, %v; want a Failed to measure scroll error", got, err)
	}
}
