package actions

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// shotDims parses `saved <path> <W>x<H>` into path, W and H.
func shotDims(t *testing.T, out string) (string, int, int) {
	t.Helper()
	f := strings.Fields(out)
	if len(f) != 3 || f[0] != "saved" {
		t.Fatalf("unexpected screenshot output: %q", out)
	}
	wh := strings.SplitN(f[2], "x", 2)
	if len(wh) != 2 {
		t.Fatalf("no WxH in %q", out)
	}
	w, err1 := strconv.Atoi(wh[0])
	h, err2 := strconv.Atoi(wh[1])
	if err1 != nil || err2 != nil {
		t.Fatalf("bad WxH in %q", out)
	}
	return f[1], w, h
}

func TestScreenshotSelectorIsCropped(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()
	vw, err := Evaluate(ctx, "window.innerWidth", "")
	if err != nil {
		t.Fatal(err)
	}
	viewport, _ := strconv.Atoi(strings.TrimSpace(vw))
	out, err := Screenshot(ctx, "x", "#sel", "", false)
	if err != nil {
		t.Fatalf("Screenshot: %v", err)
	}
	_, w, _ := shotDims(t, out)
	if viewport == 0 || w >= viewport {
		t.Errorf("selector shot width %d, want < viewport %d", w, viewport)
	}
}

func TestScreenshotNameCannotEscape(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()
	withTempCwd(t, func(dir string) {
		out, err := Screenshot(ctx, "../evil", "", "", false)
		if err != nil {
			t.Fatalf("Screenshot: %v", err)
		}
		want := filepath.Join(dir, "tmp", "screenshots", "evil.png")
		if path, _, _ := shotDims(t, out); path != want {
			t.Errorf("saved to %q, want %q", path, want)
		}
		if _, err := os.Stat(want); err != nil {
			t.Errorf("missing %s: %v", want, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "tmp", "evil.png")); err == nil {
			t.Error("file escaped tmp/screenshots/")
		}
	})
}

func TestScreenshotFullPage(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	if _, err := Navigate(ctx, srv.URL+"/scroll"); err != nil {
		t.Fatal(err)
	}
	vh, err := Evaluate(ctx, "window.innerHeight", "")
	if err != nil {
		t.Fatal(err)
	}
	viewport, _ := strconv.Atoi(strings.TrimSpace(vh))
	out, err := Screenshot(ctx, "full", "", "", true)
	if err != nil {
		t.Fatalf("Screenshot: %v", err)
	}
	if _, _, h := shotDims(t, out); viewport == 0 || h <= viewport {
		t.Errorf("full-page height %d, want > viewport %d", h, viewport)
	}
}
