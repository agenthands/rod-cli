package types

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-rod/rod/lib/launcher"
)

// Phase 30 (CDP-01 / CDP-03, D-04) — the deterministic, offline CDP-footprint
// baseline gate. It drives a REAL headless browser (no network egress: the only
// HTTP server is an in-process httptest loopback) and reads the per-session CDP
// domain-enable ledger (GetEnabledCDPDomains) back from the Context.
//
// SCOPE / what this proves (and what it does NOT): the assertion proves that
// rod-cli's own INSTRUMENTED enable-points (recordCDPDomainLocked, the sole ledger
// mutator) record ZERO of {Runtime, Network, Fetch} on a plain session — i.e. none
// of rod-cli's footprint-adding features fired. It is NOT a wire-level proof that
// Chrome enabled no such domain via a path that bypasses the instrumentation (a
// future go-rod upgrade or new feature enabling a domain without routing through
// recordCDPDomainLocked would leave this green). The wire-level identity-header
// confirmation is the separate WIRE-VERIFY (TestNetworkEvasionHeaders, wave 1) and
// the informational live cdpTell probe (wave 3). The positive controls below prove
// the ledger is REAL — each footprint feature, when opted in, records exactly its
// domain — so a regression that re-routes a feature through these enable-points
// without intending to turns this red.

// cdpFixtureServer is a loopback HTML fixture (zero network egress).
func cdpFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><body><h1>cdp fixture</h1></body></html>`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newCDPContext launches a headless browser-backed Context with the given config
// (so a test can opt a footprint feature in). Skips when no browser is available,
// matching newBrowserContext's discipline.
func newCDPContext(t *testing.T, cfg Config) *Context {
	t.Helper()
	u, err := launcher.New().Headless(true).Launch()
	if err != nil {
		// A browserless lane skips (matching newBrowserContext discipline) — but a
		// gate that silently skips has verified nothing. A gating CI lane should set
		// REQUIRE_BROWSER=1 so a missing Chrome fails LOUD rather than passing green.
		if os.Getenv("REQUIRE_BROWSER") != "" {
			t.Fatalf("REQUIRE_BROWSER set but browser launch failed: %v", err)
		}
		t.Skipf("cannot launch browser (skipping CDP-footprint test): %v", err)
	}
	cfg.CDPEndpoint = u
	if cfg.Mode == "" {
		cfg.Mode = Text
	}
	ctx := NewContext(context.Background(), cfg)
	t.Cleanup(func() { _ = ctx.CloseBrowser() })
	return ctx
}

// TestCDPFootprintBaseline is the falsifiable CDP-01 reduction gate: a plain
// session (no capture flags, no routes, no plugins) navigates the offline fixture
// and rod-cli's instrumented enable-points must have recorded NONE of
// {Runtime, Network, Fetch} (see the package-level SCOPE note).
func TestCDPFootprintBaseline(t *testing.T) {
	srv := cdpFixtureServer(t)
	ctx := newCDPContext(t, Config{})

	if _, err := ctx.EnsurePage(); err != nil {
		t.Fatalf("EnsurePage: %v", err)
	}
	if err := ctx.page.Navigate(srv.URL); err != nil {
		t.Fatalf("Navigate: %v", err)
	}

	enabled := ctx.GetEnabledCDPDomains()
	for _, d := range []string{CDPDomainRuntime, CDPDomainNetwork, CDPDomainFetch} {
		if enabled[d] {
			t.Errorf("plain session unexpectedly enabled CDP domain %q (ledger=%v) — CDP-01 baseline regressed", d, enabled)
		}
	}
	if len(enabled) != 0 {
		t.Errorf("plain session enabled-domain ledger should be empty, got %v", enabled)
	}
}

// TestCDPFootprintPositiveControls proves the ledger is REAL — each footprint
// feature, when opted in, records exactly its domain (so the baseline test above
// can actually go red, not pass vacuously).
func TestCDPFootprintPositiveControls(t *testing.T) {
	t.Run("console-capture enables Runtime", func(t *testing.T) {
		ctx := newCDPContext(t, Config{Stealth: StealthConfig{ConsoleCapture: boolPtr(true)}})
		if _, err := ctx.EnsurePage(); err != nil {
			t.Fatalf("EnsurePage: %v", err)
		}
		enabled := ctx.GetEnabledCDPDomains()
		if !enabled[CDPDomainRuntime] {
			t.Errorf("--console-capture should record Runtime, ledger=%v", enabled)
		}
		if enabled[CDPDomainFetch] {
			t.Errorf("console capture must not enable Fetch, ledger=%v", enabled)
		}
	})

	t.Run("request-capture enables Network", func(t *testing.T) {
		ctx := newCDPContext(t, Config{Stealth: StealthConfig{RequestCapture: boolPtr(true)}})
		if _, err := ctx.EnsurePage(); err != nil {
			t.Fatalf("EnsurePage: %v", err)
		}
		enabled := ctx.GetEnabledCDPDomains()
		if !enabled[CDPDomainNetwork] {
			t.Errorf("--request-capture should record Network, ledger=%v", enabled)
		}
	})

	t.Run("mock route enables Fetch", func(t *testing.T) {
		ctx := newCDPContext(t, Config{})
		if _, err := ctx.EnsurePage(); err != nil {
			t.Fatalf("EnsurePage: %v", err)
		}
		// Before adding a route, Fetch must be absent (the baseline within this session).
		if ctx.GetEnabledCDPDomains()[CDPDomainFetch] {
			t.Fatal("Fetch recorded before any route was added")
		}
		ctx.AddRoute("*api/data*", "mocked")
		enabled := ctx.GetEnabledCDPDomains()
		if !enabled[CDPDomainFetch] {
			t.Errorf("AddRoute should lazily enable + record Fetch, ledger=%v", enabled)
		}
		if enabled[CDPDomainRuntime] || enabled[CDPDomainNetwork] {
			t.Errorf("a mock route must not enable Runtime/Network, ledger=%v", enabled)
		}
	})

	t.Run("plugin lifecycle enables DOM", func(t *testing.T) {
		ctx := newCDPContext(t, Config{})
		if _, err := ctx.EnsurePage(); err != nil {
			t.Fatalf("EnsurePage: %v", err)
		}
		// Simulate the plugin lifecycle path: BindLifecycle enables the DOM
		// domain so DOMChildNodeInserted events fire. The ledger must record it.
		engine := ctx.GetPluginEngine()
		engine.Init()
		engine.BindLifecycle(context.Background(), ctx.page, ctx.RecordCDPDomain)
		enabled := ctx.GetEnabledCDPDomains()
		if !enabled[CDPDomainDOM] {
			t.Errorf("BindLifecycle should record DOM, ledger=%v", enabled)
		}
		// DOM is a footprint-adding domain — it must NOT enable Runtime/Network/Fetch.
		if enabled[CDPDomainRuntime] || enabled[CDPDomainNetwork] || enabled[CDPDomainFetch] {
			t.Errorf("BindLifecycle must not enable Runtime/Network/Fetch, ledger=%v", enabled)
		}
	})
}
