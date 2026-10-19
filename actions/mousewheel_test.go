package actions

import (
	"math"
	"regexp"
	"strings"
	"testing"

	"github.com/agenthands/godoll/humanize"
	"github.com/go-rod/rod"
)

func TestMouseWheelReportsEndOnFakeScroller(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	pos := 0.0
	origMeasure, origScroll, origMove := measureScroll, mouseScroll, moveToPoint
	measureScroll = func(el *rod.Element) ([]scrollBox, error) {
		return []scrollBox{{Y: pos, MaxY: 1000, VH: 800}}, nil
	}
	mouseScroll = func(p *rod.Page, dx, dy float64) error {
		pos = math.Min(1000, math.Max(0, pos+dy))
		return nil
	}
	moveToPoint = func(p *rod.Page, pt humanize.Point, opts ...humanize.MouseOption) error { return nil }
	defer func() { measureScroll, mouseScroll, moveToPoint = origMeasure, origScroll, origMove }()

	if got, err := MouseWheel(ctx, 0, 0, "", 0); err != nil || got != "scrolled dy=800 y=800/1000" {
		t.Fatalf("call 1: got %q, err %v", got, err)
	}
	if got, err := MouseWheel(ctx, 0, 0, "", 0); err != nil || got != "scrolled dy=200 y=1000/1000 end" {
		t.Fatalf("call 2: got %q, err %v", got, err)
	}
}

func TestMouseWheelPageDefault(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	if _, err := Navigate(ctx, srv.URL+"/scroll"); err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	got, err := MouseWheel(ctx, 0, 0, "", 0)
	if err != nil || !regexp.MustCompile(`^scrolled dy=([1-9]\d*) y=\d+/\d+$`).MatchString(got) {
		t.Fatalf("MouseWheel = %q, %v", got, err)
	}
}

// fakeScroller swaps the wheel seams for a clamped scroller (VH=800) whose
// limit is *max; onEnd runs each time a scroll lands at the limit.
func fakeScroller(t *testing.T, max *float64, onEnd func()) {
	pos := 0.0
	origMeasure, origScroll, origMove, origSettle := measureScroll, mouseScroll, moveToPoint, wheelSettle
	measureScroll = func(el *rod.Element) ([]scrollBox, error) {
		return []scrollBox{{Y: pos, MaxY: *max, VH: 800}}, nil
	}
	mouseScroll = func(p *rod.Page, dx, dy float64) error {
		pos = math.Min(*max, math.Max(0, pos+dy))
		if pos == *max && onEnd != nil {
			onEnd()
		}
		return nil
	}
	moveToPoint = func(p *rod.Page, pt humanize.Point, opts ...humanize.MouseOption) error { return nil }
	wheelSettle = 0
	t.Cleanup(func() {
		measureScroll, mouseScroll, moveToPoint, wheelSettle = origMeasure, origScroll, origMove, origSettle
	})
}

func TestMouseWheelEndAbsorbsLazyGrowth(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()
	max, grown := 1000.0, false
	fakeScroller(t, &max, func() {
		if !grown {
			grown, max = true, 1500
		}
	})
	if got, err := MouseWheel(ctx, 0, 0, "", 10); err != nil || got != "scrolled dy=1500 y=1500/1500 end" {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestMouseWheelEndCap(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()
	max := 100000.0
	fakeScroller(t, &max, nil)
	if got, err := MouseWheel(ctx, 0, 0, "", 2); err != nil || !strings.HasSuffix(got, " cap") {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestMouseWheelRefScrollsInnerBox(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	if _, err := Navigate(ctx, srv.URL+"/scroll"); err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	got, err := MouseWheel(ctx, 0, 300, "#box", 0)
	if err != nil || !regexp.MustCompile(`^scrolled dy=([1-9]\d*) y=\d+/1800$`).MatchString(got) {
		t.Fatalf("MouseWheel = %q, %v", got, err)
	}
}

func TestMouseWheelEndLive(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	if _, err := Navigate(ctx, srv.URL+"/scroll"); err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	got, err := MouseWheel(ctx, 0, 0, "", 50)
	m := regexp.MustCompile(` y=(\d+)/(\d+) end$`).FindStringSubmatch(got)
	if err != nil || m == nil || m[1] != m[2] {
		t.Fatalf("MouseWheel = %q, %v", got, err)
	}
}

func TestMouseWheelScrollsIframeUnderPointer(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	if _, err := Navigate(ctx, srv.URL+"/scroll-iframe"); err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	got, err := MouseWheel(ctx, 0, 300, "", 0)
	if err != nil || !regexp.MustCompile(`^scrolled dy=([1-9]\d*) y=\d+/\d+$`).MatchString(got) {
		t.Fatalf("MouseWheel = %q, %v", got, err)
	}
	if v, err := Evaluate(ctx, "document.getElementById('f').contentWindow.scrollY > 0 && window.scrollY === 0", ""); err != nil || v != "true" {
		t.Fatalf("iframe scrolled = %q, %v", v, err)
	}
}

func TestMouseWheelRefIframe(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	if _, err := Navigate(ctx, srv.URL+"/scroll-iframe"); err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	got, err := MouseWheel(ctx, 0, 300, "#f", 0)
	if err != nil || !regexp.MustCompile(`^scrolled dy=([1-9]\d*) y=\d+/\d+$`).MatchString(got) {
		t.Fatalf("MouseWheel = %q, %v", got, err)
	}
	if v, err := Evaluate(ctx, "document.getElementById('f').contentWindow.scrollY > 0 && window.scrollY === 0", ""); err != nil || v != "true" {
		t.Fatalf("iframe scrolled = %q, %v", v, err)
	}
}
