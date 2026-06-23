package types

import (
	"strings"
	"testing"
)

func TestBuildSnapshot_RichDOM(t *testing.T) {
	srv := newFixtureServer(t)
	ctx := newBrowserContext(t)

	page, err := ctx.EnsurePage()
	if err != nil {
		t.Fatalf("EnsurePage: %v", err)
	}
	if err := page.Navigate(srv.URL); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	if err := page.WaitLoad(); err != nil {
		t.Fatalf("wait load: %v", err)
	}

	snap, err := BuildSnapshot(page)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}

	out := snap.String()
	if out == "" {
		t.Fatal("empty snapshot string")
	}
	// Header sections from the template.
	for _, want := range []string{"Page URL", "Page Title", "Frame Count", "Page Snapshot", "Fixture Page"} {
		if !strings.Contains(out, want) {
			t.Fatalf("snapshot missing %q:\n%s", want, out)
		}
	}

	// One frame should have been recorded (the main page).
	if len(snap.frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(snap.frames))
	}

	// LocatorInFrame: resolve a ref present in the aria snapshot.
	// The aria snapshot embeds [ref=...] markers; pull the first one out.
	ref := firstRef(snap.textSnapshot)
	if ref == "" {
		t.Skip("no [ref=...] found in snapshot; cannot exercise LocatorInFrame positively")
	}
	ele, err := snap.LocatorInFrame(ref)
	if err != nil {
		t.Fatalf("LocatorInFrame(%q): %v", ref, err)
	}
	if ele == nil {
		t.Fatal("expected element from LocatorInFrame")
	}
}

func TestBuildSnapshot_IframeParent(t *testing.T) {
	// A page containing an iframe exercises the iframe branch of walk().
	ctx := newBrowserContext(t)
	page, err := ctx.EnsurePage()
	if err != nil {
		t.Fatalf("EnsurePage: %v", err)
	}

	html := `data:text/html,<html><body><h1>Parent</h1>` +
		`<iframe srcdoc="<html><body><p>child content</p></body></html>"></iframe>` +
		`</body></html>`
	if err := page.Navigate(html); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	if err := page.WaitLoad(); err != nil {
		t.Fatalf("wait load: %v", err)
	}

	snap, err := BuildSnapshot(page)
	if err != nil {
		t.Fatalf("BuildSnapshot with iframe: %v", err)
	}
	if snap.String() == "" {
		t.Fatal("empty iframe snapshot")
	}
}

func TestLocatorInFrame_BadFrameIndex(t *testing.T) {
	srv := newFixtureServer(t)
	ctx := newBrowserContext(t)
	page, err := ctx.EnsurePage()
	if err != nil {
		t.Fatalf("EnsurePage: %v", err)
	}
	if err := page.Navigate(srv.URL); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	if err := page.WaitLoad(); err != nil {
		t.Fatalf("wait load: %v", err)
	}
	snap, err := BuildSnapshot(page)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}

	// Frame index out of range (only frame 0 exists).
	if _, err := snap.LocatorInFrame("f999s5"); err == nil {
		t.Fatal("expected out-of-range frame error")
	}

	// A nonexistent ref with no frame prefix -> QueryEleByAria failure path.
	if _, err := snap.LocatorInFrame("does-not-exist-ref-xyz"); err == nil {
		t.Fatal("expected query-by-aria failure for bogus ref")
	}
}

// firstRef extracts the first ref token inside a [ref=...] marker.
func firstRef(s string) string {
	idx := strings.Index(s, "[ref=")
	if idx < 0 {
		return ""
	}
	rest := s[idx+len("[ref="):]
	end := strings.IndexByte(rest, ']')
	if end < 0 {
		return ""
	}
	return rest[:end]
}
