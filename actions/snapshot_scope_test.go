package actions

import (
	"strings"
	"testing"
)

func TestSnapshotInteractiveIsSmaller(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	full, err := Snapshot(ctx, "", false)
	if err != nil {
		t.Fatalf("Snapshot full: %v", err)
	}
	inter, err := Snapshot(ctx, "", true)
	if err != nil {
		t.Fatalf("Snapshot interactive: %v", err)
	}
	if len(inter) >= len(full) || !strings.Contains(inter, "Click Me") {
		t.Fatalf("interactive (%d bytes) not smaller than full (%d) or lost Click Me:\n%s\n--- full:\n%s", len(inter), len(full), inter, full)
	}
}

func TestSnapshotSubtreeCSS(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	out, err := Snapshot(ctx, "#sel", false)
	if err != nil {
		t.Fatalf("Snapshot #sel: %v", err)
	}
	if !strings.Contains(out, "Apple") || !strings.Contains(out, "- Scope: #sel") || strings.Contains(out, "Click Me") {
		t.Fatalf("subtree snapshot wrong:\n%s", out)
	}
}

func TestSnapshotSubtreeStaleRef(t *testing.T) {
	ctx, cleanup := newLiveCtx(t)
	defer cleanup()
	srv := newFixtureServer()
	defer srv.Close()
	navFixture(t, ctx, srv)

	oldRef := refFor(t, ctx, "Select Box")
	if _, err := Snapshot(ctx, "", false); err != nil {
		t.Fatalf("second Snapshot: %v", err)
	}
	if _, err := Snapshot(ctx, oldRef, false); err == nil || !strings.Contains(err.Error(), "stale ref") {
		t.Fatalf("Snapshot(%s) after newer snapshot: err = %v, want stale ref", oldRef, err)
	}
}
