package actions

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/agenthands/godoll/humanize"
	"github.com/agenthands/rod-cli/types"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/gson"
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

// TestDragReleasesButtonOnlyWhenPressed pins actions:A7: a failed move to the
// target releases the button exactly once; a failed press never releases it.
func TestDragReleasesButtonOnlyWhenPressed(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()
	// Both refs from one snapshot: a ref resolves only against its own snapshot.
	refs := func() (string, string) {
		snap, err := Snapshot(ctx, "", false)
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		var src, dst string
		for _, line := range strings.Split(snap, "\n") {
			if m := refRe.FindStringSubmatch(line); m != nil && strings.Contains(line, "Drag Source") {
				src = m[1]
			} else if m != nil && strings.Contains(line, "Drag Dest") {
				dst = m[1]
			}
		}
		return src, dst
	}
	src, dst := refs()

	moves, ups, downErr := 0, 0, error(nil)
	origMove, origDown, origUp := moveToElement, pageMouseDown, pageMouseUp
	moveToElement = func(p *rod.Page, el *rod.Element, opts ...humanize.MouseOption) error {
		if moves++; moves == 2 {
			return errBoom
		}
		return nil
	}
	pageMouseDown = func(p *rod.Page, btn proto.InputMouseButton, clicks int) error { return downErr }
	pageMouseUp = func(p *rod.Page, btn proto.InputMouseButton, clicks int) error { ups++; return nil }
	t.Cleanup(func() { moveToElement, pageMouseDown, pageMouseUp = origMove, origDown, origUp })

	_, err := Drag(ctx, src, dst)
	if err == nil || !strings.HasPrefix(err.Error(), "Failed to move to drag target: boom") || ups != 1 {
		t.Errorf("2nd move fails: err %v, mouseUp calls %d; want move-to-target error and 1 release", err, ups)
	}

	moves, ups, downErr = 0, 0, errBoom
	src, dst = refs()
	_, err = Drag(ctx, src, dst)
	if err == nil || !strings.HasPrefix(err.Error(), "Failed to press mouse on drag source: boom") || ups != 0 {
		t.Errorf("mouseDown fails: err %v, mouseUp calls %d; want press error and 0 releases", err, ups)
	}
}

// TestMouseWheelHorizontalThenEnd pins actions:A10 on a fake horizontal
// scroller (MaxX 1000): dx=300 moves x only, then --end stops at the limit.
func TestMouseWheelHorizontalThenEnd(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()
	x := 0.0
	origMeasure, origScroll, origMove, origSettle := measureScroll, mouseScroll, moveToPoint, wheelSettle
	measureScroll = func(el *rod.Element) ([]scrollBox, error) {
		return []scrollBox{{X: x, MaxX: 1000, VW: 800, VH: 600}}, nil
	}
	mouseScroll = func(p *rod.Page, dx, dy float64) error { x = math.Min(1000, math.Max(0, x+dx)); return nil }
	moveToPoint = func(p *rod.Page, pt humanize.Point, opts ...humanize.MouseOption) error { return nil }
	wheelSettle = 0
	t.Cleanup(func() {
		measureScroll, mouseScroll, moveToPoint, wheelSettle = origMeasure, origScroll, origMove, origSettle
	})

	if got, err := MouseWheel(ctx, 300, 0, "", 0); err != nil || got != "scrolled dx=300 x=300/1000 dy=0 y=0/0" {
		t.Fatalf("dx=300: got %q, err %v", got, err)
	}
	if got, err := MouseWheel(ctx, 300, 0, "", 50); err != nil || got != "scrolled dx=700 x=1000/1000 dy=0 y=0/0 end" {
		t.Fatalf("dx=300 --end: got %q, err %v", got, err)
	}
}

// TestMouseWheelScrollDurationSpacesNotches pins actions:A11: with
// ScrollDuration=400, dy=400 is four notches at least ~100ms apart.
func TestMouseWheelScrollDurationSpacesNotches(t *testing.T) {
	ctx, cleanup := newLiveCtxCfg(t, func(c *types.Config) { d := 400; c.Stealth.ScrollDuration = &d })
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	var at []time.Time
	origMeasure, origScroll, origMove := measureScroll, mouseScroll, moveToPoint
	measureScroll = func(el *rod.Element) ([]scrollBox, error) { return []scrollBox{{MaxY: 10000, VH: 800}}, nil }
	mouseScroll = func(p *rod.Page, dx, dy float64) error { at = append(at, time.Now()); return nil }
	moveToPoint = func(p *rod.Page, pt humanize.Point, opts ...humanize.MouseOption) error { return nil }
	t.Cleanup(func() { measureScroll, mouseScroll, moveToPoint = origMeasure, origScroll, origMove })

	if _, err := MouseWheel(ctx, 0, 400, "", 0); err != nil {
		t.Fatalf("MouseWheel: %v", err)
	}
	if len(at) != 4 {
		t.Fatalf("scroll calls = %d, want 4", len(at))
	}
	for i := 1; i < len(at); i++ {
		if gap := at[i].Sub(at[i-1]); gap < 90*time.Millisecond {
			t.Errorf("gap %d = %v, want >= 90ms", i, gap)
		}
	}
}

// TestStealthCheckRawVerdicts pins actions:A13 with the live-page reads
// stubbed: all good signals give PASS; bad ones give the exact FAIL line; a
// read error fails its signal as read-error.
func TestStealthCheckRawVerdicts(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()
	good := map[string]string{
		"webdriver": "undefined", "pluginsLength": "5", "userAgent": "Mozilla/5.0 (X11; Linux x86_64) Chrome/130.0.0.0",
		"webglVendor": "Google Inc. (NVIDIA)", "webglRenderer": "ANGLE (NVIDIA GeForce)", "permissionsConsistent": "true",
		"languages": "en-US,en", "screen": "1920x1080", "windowChrome": "true", "chromeRuntime": "true", "timezone": "Europe/Riga",
	}
	var signals map[string]string
	readErr := ""
	orig := runtimeEvaluateCall
	runtimeEvaluateCall = func(req proto.RuntimeEvaluate, page *rod.Page) (*proto.RuntimeEvaluateResult, error) {
		name, isRead := strings.CutPrefix(req.Expression, "String(window.__detect.")
		name = strings.TrimSuffix(name, ")")
		switch {
		case !isRead && strings.Contains(req.Expression, "__detect.ready"):
			return &proto.RuntimeEvaluateResult{Result: &proto.RuntimeRemoteObject{Value: gson.New(true)}}, nil
		case !isRead: // the probe injection
			return &proto.RuntimeEvaluateResult{Result: &proto.RuntimeRemoteObject{}}, nil
		case name == readErr:
			return nil, errBoom
		}
		return &proto.RuntimeEvaluateResult{Result: &proto.RuntimeRemoteObject{Value: gson.New(signals[name])}}, nil
	}
	t.Cleanup(func() { runtimeEvaluateCall = orig })

	cases := []struct {
		name, readErr string
		bad           map[string]string
		want          string
	}{
		{"all pass", "", nil, "PASS"},
		{"listed fail", "", map[string]string{
			"webdriver": "true", "pluginsLength": "0", "userAgent": "Mozilla/5.0 HeadlessChrome/130.0.0.0",
			"webglRenderer": "Google SwiftShader", "screen": "0x0", "windowChrome": "false", "timezone": "UTC",
		}, "FAIL webdriver=FAIL(true) pluginsLength=FAIL(0) userAgent=FAIL(HeadlessChrome) webglRenderer=FAIL(Google SwiftShader) screen=FAIL(0x0) windowChrome=FAIL(absent) timezone=FAIL(UTC)"},
		{"read error", "languages", nil, "FAIL languages=FAIL(read-error: boom)"},
	}
	for _, c := range cases {
		signals, readErr = map[string]string{}, c.readErr
		for k, v := range good {
			signals[k] = v
		}
		for k, v := range c.bad {
			signals[k] = v
		}
		if got, err := StealthCheck(ctx, "", true, false); err != nil || got != c.want {
			t.Errorf("%s: got %q, err %v; want %q", c.name, got, err, c.want)
		}
	}
}
