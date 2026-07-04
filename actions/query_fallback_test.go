package actions

import (
	"testing"
	"time"

	"github.com/agenthands/rod-cli/types"
)

// ---------------------------------------------------------------------------
// CSS-selector fallback / null-vs-[] hang regression tests
//
// getElementByRef (actions.go ~362) resolves a ref two ways: through a cached
// Snapshot's LocatorInFrame (when `snapshot` was called earlier in the
// session) or, when no snapshot exists yet, directly against the live page
// via queryElementDirect (= utils.QueryEleByAria). Both paths ultimately
// invoke the same injected engine method, snapshotEngine.queryAll (compiled
// from types/js/snapshotter_raw.js into the embedded types/js/snapshotter.js
// and injected on every new document via page.EvalOnNewDocument in
// types/context.go) -- so these tests run against the REAL embedded engine,
// not a stub.
//
// The critical invariant under test: every "not found" branch of queryAll
// must return a JS `[]`, never `null`. go-rod's Page.ElementByJS (query.go)
// treats a returned `null` as "not rendered yet" and polls forever against
// the page's own (here, undeadlined) context -- a single regressed branch
// hangs the whole daemon. An empty array is a definitive miss: ElementByJS's
// poll loop exits on the very first Evaluate call, and the surrounding
// Go-level retry.Retry (getElementByRef, 3 retries / exponential backoff /
// 500ms base delay, ~3.5s worst case) owns the bounded retry policy instead.
// ---------------------------------------------------------------------------

// clickBounded runs Click in a goroutine and fails the test if it does not
// return within timeout. This is the enforced regression guard: if any
// "not found" branch of queryAll regresses back to returning `null`, Click
// hangs forever instead of erroring, and this guard turns that into a fast,
// loud test failure instead of a wedged `go test` run.
func clickBounded(t *testing.T, ctx *types.Context, ref string, timeout time.Duration) (string, error) {
	t.Helper()

	type result struct {
		out string
		err error
	}
	done := make(chan result, 1)
	go func() {
		out, err := Click(ctx, ref)
		done <- result{out, err}
	}()

	select {
	case r := <-done:
		return r.out, r.err
	case <-time.After(timeout):
		t.Fatal("Click hung — regression of the queryAll null/[] \"not found\" contract (types/js/snapshotter_raw.js)")
		return "", nil // unreachable: t.Fatal calls runtime.Goexit()
	}
}

// TestClickFillType_CSSSelector_NoPriorSnapshot is the headline regression
// test for the fix: click/fill/type must work against a plain CSS selector
// immediately after navigation, with NO prior `snapshot` call in the
// session. Nothing here calls Snapshot(ctx) before these actions, so
// getElementByRef is exercised purely via its direct-page fallback branch
// (ctx.LatestSnapshot() errors -> queryElementDirect against
// ctx.ControlledPage()).
func TestClickFillType_CSSSelector_NoPriorSnapshot(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	if _, err := Click(ctx, "#btn"); err != nil {
		t.Fatalf("Click(#btn) with no prior snapshot: %v", err)
	}
	if _, err := Fill(ctx, "#textinput", "hello", false); err != nil {
		t.Fatalf("Fill(#textinput) with no prior snapshot: %v", err)
	}
	if _, err := Type(ctx, "#textinput", " world"); err != nil {
		t.Fatalf("Type(#textinput) with no prior snapshot: %v", err)
	}
}

// TestClick_CSSSelector_NotFound_FailsFast: a syntactically valid CSS
// selector matching no element must fail quickly, never hang. This is the
// exact scenario the daemon hit live (documented in the fix): a bare `null`
// from queryAll's mismatch branch would otherwise poll forever.
func TestClick_CSSSelector_NotFound_FailsFast(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	_, err := clickBounded(t, ctx, "#does-not-exist", 10*time.Second)
	if err == nil {
		t.Fatal("expected an error for a CSS selector matching no element")
	}
}

// TestClick_InvalidCSSSelector_FailsFast proves the try/catch around
// document.querySelector in queryAll's CSS-fallback branch converts a
// DOMException (invalid selector syntax, e.g. an unclosed/malformed
// selector) into a fast [] miss, not an unhandled JS exception or a hang.
func TestClick_InvalidCSSSelector_FailsFast(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	_, err := clickBounded(t, ctx, ":::invalid-css:::", 10*time.Second)
	if err == nil {
		t.Fatal("expected an error for syntactically invalid CSS")
	}
}

// TestClick_RefAfterSnapshot_StillWorks guards the pre-existing workflow:
// once a real snapshot has been taken, resolving by its [ref=sNeM] token
// must keep working unchanged (exercises getElementByRef's
// snapshot.LocatorInFrame branch, not the new fallback).
func TestClick_RefAfterSnapshot_StillWorks(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	ref := refFor(t, ctx, "Click Me")
	if _, err := Click(ctx, ref); err != nil {
		t.Fatalf("Click(%q) by snapshot ref: %v", ref, err)
	}
}

// TestClick_StaleRef_FailsFast covers the second, independent latent hang
// bug fixed alongside the CSS fallback: queryAll's ref-matched branch used
// to return `null` for a disconnected element (removed from the DOM after
// the snapshot was taken). A stale ref must now fail fast, not hang.
func TestClick_StaleRef_FailsFast(t *testing.T) {
	ctx, cleanup := withLiveFixture(t)
	defer cleanup()

	ref := refFor(t, ctx, "Click Me")

	if _, err := Evaluate(ctx, "() => document.getElementById('btn').remove()", ""); err != nil {
		t.Fatalf("failed to remove element for staleness test: %v", err)
	}

	_, err := clickBounded(t, ctx, ref, 10*time.Second)
	if err == nil {
		t.Fatal("expected an error for a stale/disconnected snapshot ref")
	}
}
