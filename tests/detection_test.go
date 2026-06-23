package tests

// HARNESS-02 — e2e detection harness.
//
// This test boots the offline internal/detect fixture server (plan 24-01),
// drives the REAL ../rod-cli binary to it via the existing runCli helper
// (tests/cli_test.go), and asserts each table-stakes stealth signal by reading
// the verdict back from the LIVE page via the `eval` command
// (window.__detect.<signal>) — NEVER from a Go config/fingerprint field. This is
// the v1.5 "validate-live-not-source" rule: the harness proves the SHIPPED
// binary's stealth, not godoll in isolation.
//
// Convergence (do NOT duplicate):
//   - stealth_test.go already asserts webdriver / plugins / UA via runCli+eval.
//     This file EXTENDS coverage with WebGL, permissions consistency, timezone,
//     window.chrome, languages, and screen — read through window.__detect.
//   - network_evasion_test.go owns the full Sec-CH-UA header dump. The KNOWN-RED
//     Client-Hints check here asserts only the CH-version baseline truth.
//
// KNOWN-RED baseline signals (WebRTC ICE leak, Client-Hints 121) are asserted at
// their CURRENT truth with `// KNOWN-RED (Phase NN: <REQ>)` markers and are kept
// as executing assertions — never skipped (no t.Skip). CI stays green on the
// documented baseline; each marker flips to required-green when its later phase
// lands. The verify gate negative-greps this file for any skip directive.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/agenthands/rod-cli/internal/detect"
)

// evalResultPrefix is what the `eval` command prepends to every value it prints
// (actions/actions.go: "Evaluate code successfully with result: <value>").
const evalResultPrefix = "Evaluate code successfully with result:"

// evalDetect reads a single window.__detect.<signal> back from the LIVE page via
// the eval command and returns the trimmed string value. It wraps the read in
// String(...) so booleans/numbers/objects all serialize to a comparable string.
// READ FROM THE LIVE PAGE — never from a Go Config/Fingerprint field.
func evalDetect(t *testing.T, signal string) string {
	t.Helper()
	out, err := runCli("eval", "String(window.__detect."+signal+")")
	if err != nil {
		t.Fatalf("eval window.__detect.%s failed: %v\nOutput: %s", signal, err, out)
	}
	val := out
	if i := strings.Index(out, evalResultPrefix); i >= 0 {
		val = out[i+len(evalResultPrefix):]
	}
	return strings.TrimSpace(val)
}

// waitForDetectReady polls window.__detect.ready until the async probes
// (permissions + WebRTC) have settled into the global, so every signal is
// populated before we read it. Bounded retry loop; fatal if never ready.
func waitForDetectReady(t *testing.T) {
	t.Helper()
	for i := 0; i < 30; i++ {
		out, err := runCli("eval", "window.__detect && window.__detect.ready === true")
		if err == nil && strings.Contains(out, "result: true") {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	// Capture final state for the failure message.
	out, _ := runCli("eval", "window.__detect ? JSON.stringify(window.__detect) : 'no-global'")
	t.Fatalf("window.__detect.ready never became true; last state: %s", out)
}

// TestDetectionHarness drives the live rod-cli binary against the offline detect
// fixture and asserts each table-stakes signal by reading it back from the live
// page. Headless is the blocking CI gate; a headful row is env-gated below.
func TestDetectionHarness(t *testing.T) {
	ds, err := detect.New()
	if err != nil {
		t.Fatalf("detect.New() failed: %v", err)
	}
	ds.Start()
	defer ds.Close()

	// Deterministic daemon state before the run (every existing test does this).
	runCli("close")
	defer runCli("close")

	// Navigate the real daemon-driven browser to the offline fixture.
	out, err := runCli("goto", ds.URL())
	if err != nil {
		t.Fatalf("goto %s failed: %v\nOutput: %s", ds.URL(), err, out)
	}

	// VALIDATE-03 observability (ties to plan 24-02): runCli folds stderr INTO
	// its stdout buffer (cli_test.go: cmd.Stderr = &out), so a `warning:` line
	// emitted when EvasionManager.Apply()/fingerprint generation fails IS visible
	// here. On the normal/success path navigation must NOT emit a spurious
	// warning — assert the channel is clean so a real evasion failure would stand
	// out. (When Apply()/Generate() fails, the warning surfaces through exactly
	// this captured output.)
	if strings.Contains(strings.ToLower(out), "warning:") {
		t.Errorf("VALIDATE-03: success-path navigation emitted an unexpected warning: %s", out)
	}

	// Let the async probes (permissions, WebRTC) settle into window.__detect.
	waitForDetectReady(t)

	// --- Extended table-stakes signals (read from the LIVE page) --------------
	// EXTEND, do not duplicate stealth_test.go (webdriver/plugins/UA live there).

	// WebGL vendor/renderer: a software renderer (SwiftShader/llvmpipe/"software")
	// is the classic headless WebGL tell. The shipped stealth fingerprint spoofs a
	// real GPU, so assert the renderer is NOT a software rasterizer.
	t.Run("webgl_not_software", func(t *testing.T) {
		vendor := evalDetect(t, "webglVendor")
		renderer := evalDetect(t, "webglRenderer")
		if vendor == "" || vendor == "no-context" || vendor == "no-extension" {
			t.Errorf("webglVendor missing/unmasked (headless tell): %q", vendor)
		}
		combined := strings.ToLower(vendor + " " + renderer)
		for _, tell := range []string{"swiftshader", "llvmpipe", "software"} {
			if strings.Contains(combined, tell) {
				t.Errorf("WebGL reports software renderer %q (headless tell): vendor=%q renderer=%q",
					tell, vendor, renderer)
			}
		}
	})

	// Permissions consistency: Notification.permission vs
	// navigator.permissions.query state. The classic headless mismatch
	// ("default" vs "denied") is the tell; assert the consistent verdict.
	t.Run("permissions_consistent", func(t *testing.T) {
		got := evalDetect(t, "permissionsConsistent")
		if got != "true" {
			t.Errorf("permissionsConsistent is not the non-tell value 'true', got: %q "+
				"(notif=%q queryState=%q)", got,
				evalDetect(t, "notificationPermission"), evalDetect(t, "permissionsQueryState"))
		}
	})

	// Timezone: a resolved, non-empty IANA zone string.
	t.Run("timezone_resolved", func(t *testing.T) {
		tz := evalDetect(t, "timezone")
		if tz == "" || tz == "undefined" || !strings.Contains(tz, "/") {
			t.Errorf("timezone is not a resolved IANA zone, got: %q", tz)
		}
	})

	// window.chrome present: its ABSENCE is the headless tell.
	t.Run("window_chrome_present", func(t *testing.T) {
		got := evalDetect(t, "windowChrome")
		if got != "true" {
			t.Errorf("window.chrome absent (headless tell), windowChrome=%q", got)
		}
	})

	// navigator.languages non-empty.
	t.Run("languages_present", func(t *testing.T) {
		langs := evalDetect(t, "languages")
		if langs == "" || langs == "undefined" {
			t.Errorf("navigator.languages is empty (headless tell), got: %q", langs)
		}
	})

	// screen dims non-zero: a zero-size screen is a headless tell.
	t.Run("screen_nonzero", func(t *testing.T) {
		scr := evalDetect(t, "screen") // "WxH"
		parts := strings.SplitN(scr, "x", 2)
		if len(parts) != 2 || parts[0] == "0" || parts[1] == "0" ||
			parts[0] == "" || parts[1] == "" {
			t.Errorf("screen dims zero/malformed (headless tell), got: %q", scr)
		}
	})

	// --- KNOWN-RED baseline signals (assert CURRENT truth; never skipped) ------

	// KNOWN-RED (Phase 27 HARDEN-01): WebRTC ICE candidate leaks the local IP.
	// We assert the signal is OBSERVABLE and recorded at its current truth so CI
	// stays green on the documented baseline; this assertion flips to
	// required-green (empty / no leaked host IP) when EvadeWebRTC is wired.
	// The probe records "" when no candidate IP was gathered, a comma-joined IP
	// list when it leaks, "no-RTCPeerConnection" when the API is absent, or an
	// "error: ..." string. Any of these is the documented baseline truth — what
	// must NOT happen is the signal being unpopulated (undefined), which would
	// mean the harness cannot see the WebRTC surface at all.
	t.Run("webrtc_ice_known_red", func(t *testing.T) {
		ice := evalDetect(t, "webrtcIce")
		if ice == "undefined" {
			t.Errorf("KNOWN-RED WebRTC signal unpopulated — harness cannot observe "+
				"the WebRTC surface; got: %q", ice)
		}
		// Record the baseline truth in the test log for traceability.
		t.Logf("KNOWN-RED webrtcIce baseline truth: %q (flips to required-empty in Phase 27 HARDEN-01)", ice)
	})

	// KNOWN-RED (Phase 26 FINGERPRINT-03): Client-Hints version is hardcoded and
	// can mismatch the UA (the "121" tell). We assert only the CH-version baseline
	// truth via the header path here — network_evasion_test.go owns the full
	// header dump, so we do NOT duplicate it. This flips to required-green when
	// Sec-CH-UA is derived from the UA.
	t.Run("client_hints_known_red", func(t *testing.T) {
		// Echo headers as JSON from a loopback httptest server, then read the
		// Sec-CH-UA value the SHIPPED binary actually sent.
		hdr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(r.Header)
		}))
		defer hdr.Close()

		if _, err := runCli("goto", hdr.URL); err != nil {
			t.Fatalf("goto header-echo server failed: %v", err)
		}
		body, err := runCli("eval", "document.body.innerText")
		if err != nil {
			t.Fatalf("eval body failed: %v", err)
		}
		// The CH header surface must be present and observable. Its exact version
		// is the KNOWN-RED baseline — we record it rather than fail on it.
		if !strings.Contains(body, "Sec-Ch-Ua") {
			t.Errorf("KNOWN-RED Client-Hints: Sec-Ch-Ua header surface not observable, got: %s", body)
		}
		t.Logf("KNOWN-RED Client-Hints baseline: Sec-Ch-Ua header observed " +
			"(version hardcoded; flips to required-green in Phase 26 FINGERPRINT-03 when derived from UA)")
	})

	// --- Headful matrix row (CONTEXT.md:20) -----------------------------------
	// Headless is the blocking CI gate (the run above). Headful needs xvfb in CI
	// and is slow/flaky, so it is an OPT-IN local row gated by ROD_HEADFUL=1,
	// driven through the same runCli surface — NOT an always-on path.
	if os.Getenv("ROD_HEADFUL") == "1" {
		t.Run("headful_optin", func(t *testing.T) {
			runCli("close")
			out, err := runCli("--headless=false", "goto", ds.URL())
			if err != nil {
				t.Fatalf("headful goto failed: %v\nOutput: %s", err, out)
			}
			waitForDetectReady(t)
			if got := evalDetect(t, "windowChrome"); got != "true" {
				t.Errorf("headful: window.chrome absent, got: %q", got)
			}
		})
	}
}
