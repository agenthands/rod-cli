//go:build detection_live

// Tier-2 best-effort LIVE detection smoke check (LIVEWAF-01).
//
// This file is the OPT-IN, NON-BLOCKING sibling of the Tier-1 offline harness
// (tests/detection_test.go). It is excluded from the default build/test and from
// the CI gate BY CONSTRUCTION: the `//go:build detection_live` constraint on the
// first line means a plain `go build ./...` / `go test ./...` (and .github/
// workflows/test.yml, which runs `go test ./... -count=1` with NO -tags) never
// compiles or runs it. Run it deliberately:
//
//	go build -o rod-cli .   # the suite execs the prebuilt ../rod-cli
//	go test -tags detection_live ./tests/ -run TestLiveDetection -v
//
// Non-blocking in BOTH senses (the honesty invariant of this phase):
//   1. Excluded from CI by the build tag.
//   2. Informational WITHIN the suite — every outcome is reported with t.Logf,
//      and an unreachable target / no-egress environment is t.Skip'd. This suite
//      NEVER calls t.Fatal/t.Errorf on a live detection or a network error: a
//      flaky third-party challenge must not produce a red build. A live "green"
//      is NOT the bar — the bar is exclusion + honest, informational reporting.
//      See docs/stealth-validation.md for the ceiling (TLS/JA3/IP/CDP layers a
//      JS-injecting CLI cannot control).

package tests

import (
	"strings"
	"testing"
)

// Live third-party targets. THESE ARE BEST-EFFORT: they are third-party URLs that
// may change behavior, move, rate-limit, or disappear at any time, and reaching
// them requires real network egress that the verify environment may not have.
// Treat every one as "may be unreachable" — never as a guaranteed signal.
const (
	// cloudflareLiveURL is a Cloudflare-challenge-protected page.
	cloudflareLiveURL = "https://nopecha.com/demo/cloudflare"
	// dataDomeLiveURL is a DataDome-protected demo endpoint.
	dataDomeLiveURL = "https://antoinevastel.com/bots/datadome"
	// creepJSLiveURL exposes a fingerprint trust score in the DOM.
	creepJSLiveURL = "https://abrahamjuliot.github.io/creepjs/"
	// cdpProbeLiveURL is a benign, stable page used only to run the self-contained
	// CDP-tell heuristic against a REAL remote page (Phase 30 CDP-03 live leg). Any
	// loaded page works for the heuristic; example.com is the canonical stable URL.
	cdpProbeLiveURL = "https://example.com/"
)

// cdpTellExpr is the self-contained CDP-presence heuristic, identical to the
// offline harness's window.__detect.cdpTell (internal/detect/detect.js): serializing
// an Error triggers the .stack getter, which a CDP remote-object preview (tied to an
// enabled Runtime domain) may observe. HEURISTIC ONLY — "no-signal" is a possible
// false negative; it measures exposure, it does not prove absence. With the Phase-30
// reduction a plain session does not enable Runtime, so the expected plain-path
// verdict is "no-signal".
const cdpTellExpr = `(function(){var fired=false;var e=new Error();` +
	`Object.defineProperty(e,'stack',{configurable:true,get:function(){fired=true;return '';}});` +
	`try{console.debug(e);}catch(_){}return fired?'stack-getter-fired':'no-signal';})()`

// liveEvalBestEffort runs an eval against the live page and returns (value, ok).
// ok is false on any error — the caller decides whether that means "skip"
// (unreachable) or "log a degraded verdict". It NEVER fails the test itself.
func liveEvalBestEffort(t *testing.T, expr string) (string, bool) {
	t.Helper()
	out, err := runCli("eval", expr)
	if err != nil {
		return out, false
	}
	val := out
	if i := strings.Index(out, evalResultPrefix); i >= 0 {
		val = out[i+len(evalResultPrefix):]
	}
	return strings.TrimSpace(val), true
}

// gotoLiveOrSkip navigates the real binary to a live target. A navigation error
// (no egress, DNS failure, target down) is treated as "unreachable" and SKIPS the
// subtest — it is explicitly NOT a failure, per the non-blocking discipline.
func gotoLiveOrSkip(t *testing.T, url string) {
	t.Helper()
	out, err := runCli("goto", url)
	if err != nil {
		// runCli execs the prebuilt ../rod-cli, so this error can mean the target
		// is unreachable (no egress / target down) OR the binary was not built
		// (forgot `go build -o rod-cli .`). Either way it is a best-effort SKIP,
		// never a failure — but don't assert a cause we didn't establish.
		t.Skipf("live target %s unreachable (no network egress / target down) or "+
			"rod-cli binary not built (run `go build -o rod-cli .` first) — "+
			"best-effort skip, not a failure: %v\noutput: %s", url, err, out)
	}
	// Even on a nil error the page may be a "can't reach" interstitial; the
	// per-target verdict logic below interprets the loaded DOM informationally.
}

// TestLiveDetection is the opt-in Tier-2 smoke check. Each target is an
// independently-skippable subtest that navigates the REAL binary to a live WAF /
// fingerprint page and reports the observed verdict INFORMATIONALLY. No subtest
// asserts a pass: a live green is flaky by nature and is not the bar.
func TestLiveDetection(t *testing.T) {
	// Deterministic daemon state, mirroring every other binary-driving test.
	runCli("close")
	defer runCli("close")

	t.Run("cloudflare", func(t *testing.T) {
		gotoLiveOrSkip(t, cloudflareLiveURL)
		title, ok := liveEvalBestEffort(t, "String(document.title)")
		if !ok {
			t.Skipf("cloudflare: could not read page state (best-effort skip): %s", title)
		}
		body, bodyOK := liveEvalBestEffort(t, "String(document.body ? document.body.innerText.slice(0,400) : '')")
		if !bodyOK {
			body = "" // never let eval error text feed the content heuristic
		}
		lc := strings.ToLower(title + " " + body)
		// Heuristic, informational only: Cloudflare's interstitial mentions these.
		challenged := strings.Contains(lc, "just a moment") ||
			strings.Contains(lc, "checking your browser") ||
			strings.Contains(lc, "cf-challenge") ||
			strings.Contains(lc, "attention required") ||
			strings.Contains(lc, "verify you are human")
		if challenged {
			t.Logf("cloudflare: CHALLENGE/BLOCK observed (informational) — title=%q", title)
		} else {
			t.Logf("cloudflare: no challenge interstitial observed (informational, NOT a guarantee of passing — TLS/IP layers are not exercised here) — title=%q", title)
		}
	})

	t.Run("datadome", func(t *testing.T) {
		gotoLiveOrSkip(t, dataDomeLiveURL)
		title, ok := liveEvalBestEffort(t, "String(document.title)")
		if !ok {
			t.Skipf("datadome: could not read page state (best-effort skip): %s", title)
		}
		body, bodyOK := liveEvalBestEffort(t, "String(document.body ? document.body.innerText.slice(0,400) : '')")
		if !bodyOK {
			body = "" // never let eval error text feed the content heuristic
		}
		lc := strings.ToLower(title + " " + body)
		challenged := strings.Contains(lc, "datadome") ||
			strings.Contains(lc, "blocked") ||
			strings.Contains(lc, "captcha") ||
			strings.Contains(lc, "are you a robot") ||
			strings.Contains(lc, "access denied")
		if challenged {
			t.Logf("datadome: CHALLENGE/BLOCK observed (informational) — title=%q", title)
		} else {
			t.Logf("datadome: no block page observed (informational, NOT a guarantee — IP-reputation/TLS layers not exercised) — title=%q", title)
		}
	})

	t.Run("cdp-footprint", func(t *testing.T) {
		// Phase 30 CDP-03 live leg (D-06): observe the CDP footprint against a REAL
		// remote page on the PLAIN session path (no --console-capture / --request-capture,
		// no routes). Informational only — t.Logf/t.Skip, never Fatal/Errorf. The
		// deterministic gate is the offline ledger assertion (types/cdp_footprint_test.go);
		// this is the best-effort realism signal. See docs/cdp-footprint.md.
		gotoLiveOrSkip(t, cdpProbeLiveURL)
		verdict, ok := liveEvalBestEffort(t, cdpTellExpr)
		if !ok {
			t.Skipf("cdp-footprint: could not read CDP-tell verdict (best-effort skip): %s", verdict)
		}
		if verdict == "no-signal" {
			t.Logf("cdp-footprint: cdpTell=%q on the plain session — consistent with the reduced baseline (Runtime not enabled). Informational, NOT a proof of absence (heuristic; CDP transport still uses Page/Target).", verdict)
		} else {
			t.Logf("cdp-footprint: cdpTell=%q observed (informational) — a CDP stack-getter tell fired; see docs/cdp-footprint.md honest ceiling.", verdict)
		}
	})

	t.Run("creepjs", func(t *testing.T) {
		gotoLiveOrSkip(t, creepJSLiveURL)
		// CreepJS computes asynchronously; read the trust-score element best-effort.
		// We do NOT poll-with-fatal — a missing element is just an informational
		// "could not read", never a failure.
		score, ok := liveEvalBestEffort(t,
			`String((document.querySelector('.unblurred')||document.querySelector('#fingerprint-data')||{}).textContent||'').slice(0,200)`)
		if !ok || strings.TrimSpace(score) == "" {
			t.Logf("creepjs: trust score not readable yet (async render / DOM changed) — best-effort, informational; not a failure")
			return
		}
		t.Logf("creepjs: fingerprint readout (informational, best-effort): %s", score)
	})
}
