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
// KNOWN-RED baseline signals are asserted at their CURRENT truth with
// `// KNOWN-RED (Phase NN: <REQ>)` markers and are kept as executing assertions —
// never skipped (no t.Skip). CI stays green on the documented baseline; each
// marker flips to required-green when its later phase lands. The verify gate
// negative-greps this file for any skip directive.
//
// Phase-26 flip (FINGERPRINT-03): the Client-Hints "121" KNOWN-RED is now a
// REQUIRED-GREEN assertion (client_hints_ua_derived) — Sec-Ch-Ua major == UA
// Chrome major == navigator.userAgentData version, all read from the live page.
// The blocking consistency_invariant subtest additionally proves those surfaces
// plus navigator.platform tell one OS+version story (success criterion 1). Only
// the WebRTC ICE leak remains KNOWN-RED (flips in Phase 27 HARDEN-01).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/agenthands/rod-cli/internal/detect"
)

// chromeMajorRe extracts the Chrome major version from a UA string (digits only).
var chromeMajorRe = regexp.MustCompile(`Chrome/(\d+)`)

// chromeMajor parses the Chrome major version out of a UA string, "" if absent.
func chromeMajor(ua string) string {
	m := chromeMajorRe.FindStringSubmatch(ua)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// writeProfile writes a stealth profile JSON to a temp file and returns its path.
// It is the only CLI-reachable way to enable Client-Hints spoofing: there is no
// --spoof-client-hints flag, only the profile/config field SpoofClientHints, so
// the FINGERPRINT-02/03 invariant (Sec-Ch-Ua / userAgentData coherence) can only
// be ACTIVATED end-to-end through a --profile JSON. See SUMMARY "Coherence gap":
// Client-Hints spoofing is opt-in, NOT the default identity, and --user-agent
// alone does not enable it. Activating the feature is legitimate test setup; the
// assertions below remain hard live-page reads, never weakened.
func writeProfile(t *testing.T, fields map[string]interface{}) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stealth-profile-*.json")
	if err != nil {
		t.Fatalf("create temp profile failed: %v", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(fields); err != nil {
		t.Fatalf("encode temp profile failed: %v", err)
	}
	return f.Name()
}

// liveEval evaluates an arbitrary JS expression against the LIVE page and returns
// the trimmed String()-coerced value. READ FROM THE LIVE PAGE — never a Go field.
func liveEval(t *testing.T, expr string) string {
	t.Helper()
	out, err := runCli("eval", "String("+expr+")")
	if err != nil {
		t.Fatalf("eval %s failed: %v\nOutput: %s", expr, err, out)
	}
	val := out
	if i := strings.Index(out, evalResultPrefix); i >= 0 {
		val = out[i+len(evalResultPrefix):]
	}
	return strings.TrimSpace(val)
}

// liveUserAgentDataMajor reads the navigator.userAgentData "Google Chrome" brand
// version directly from the LIVE page (godoll spoofs this UA-derived in Plan 02).
func liveUserAgentDataMajor(t *testing.T) string {
	t.Helper()
	expr := `(function(){var b=(navigator.userAgentData&&navigator.userAgentData.brands)||[];` +
		`for(var i=0;i<b.length;i++){if(b[i].brand==='Google Chrome')return b[i].version;}return '';})()`
	return liveEval(t, expr)
}

// liveHeader navigates the live daemon-driven browser to a loopback header-echo
// server, reads back the named request header the SHIPPED binary actually sent
// (zero network egress — only an httptest loopback server), and returns it.
func liveHeader(t *testing.T, name string) string {
	t.Helper()
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
		t.Fatalf("eval header-echo body failed: %v", err)
	}
	// The echoed body is http.Header JSON: {"Sec-Ch-Ua":["..."], ...}.
	var headers map[string][]string
	start := strings.Index(body, "{")
	end := strings.LastIndex(body, "}")
	if start < 0 || end < start {
		t.Fatalf("header-echo body is not JSON: %s", body)
	}
	if err := json.Unmarshal([]byte(body[start:end+1]), &headers); err != nil {
		t.Fatalf("failed to parse header-echo JSON: %v\nbody: %s", err, body)
	}
	if vals, ok := headers[name]; ok && len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// liveSecChUaMajor reads the live Sec-Ch-Ua header and parses the Google-Chrome
// brand version (the "Google Chrome";v="<major>" entry).
func liveSecChUaMajor(t *testing.T) string {
	t.Helper()
	val := liveHeader(t, "Sec-Ch-Ua")
	if val == "" {
		t.Fatalf("Sec-Ch-Ua header not observable on the live page")
	}
	re := regexp.MustCompile(`"Google Chrome";v="(\d+)"`)
	m := re.FindStringSubmatch(val)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// stripQuotes trims surrounding double-quotes from a Sec-Ch-Ua-Platform value
// (the header form is quoted, e.g. "Windows").
func stripQuotes(s string) string {
	return strings.Trim(strings.TrimSpace(s), `"`)
}

// osFromUA classifies the OS family from a UA string's OS token.
func osFromUA(ua string) string {
	switch {
	case strings.Contains(ua, "Windows"):
		return "windows"
	case strings.Contains(ua, "Macintosh"), strings.Contains(ua, "Mac OS X"):
		return "macos"
	case strings.Contains(ua, "Linux"):
		return "linux"
	}
	return ""
}

// osFromPlatform classifies the OS family from a navigator.platform value.
func osFromPlatform(p string) string {
	switch {
	case strings.HasPrefix(p, "Win"):
		return "windows"
	case p == "MacIntel", strings.Contains(p, "Mac"):
		return "macos"
	case strings.Contains(p, "Linux"):
		return "linux"
	}
	return ""
}

// osFromChPlatform classifies the OS family from a Sec-Ch-Ua-Platform value.
func osFromChPlatform(p string) string {
	switch p {
	case "Windows":
		return "windows"
	case "macOS":
		return "macos"
	case "Linux":
		return "linux"
	}
	return ""
}

// restoreDefaultDaemon tears down a pinned/profile session spawned inside a
// subtest and re-establishes the DEFAULT-identity daemon on the detect fixture,
// so a pinned session never bleeds into a later subtest (the daemon-leak guard
// the plan calls out). Used via defer at the end of profile-pinned subtests.
func restoreDefaultDaemon(t *testing.T, url string) {
	t.Helper()
	runCli("close")
	if _, err := runCli("goto", url); err != nil {
		t.Fatalf("restore default daemon: goto %s failed: %v", url, err)
	}
	waitForDetectReady(t)
}

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
	// The probe records the connection-address of each ICE host candidate (field
	// 4 of the SDP candidate line) regardless of form: a real IPv4/IPv6 address is
	// the leak Phase 27 EvadeWebRTC must eliminate, while a modern-Chrome mDNS
	// `<uuid>.local` hostname is the masked baseline truth. It records "" only when
	// no candidate was gathered, "no-RTCPeerConnection" when the API is absent, or
	// an "error: ..." string. Any of these is the documented baseline — what must
	// NOT happen is the signal being unpopulated (undefined), which would mean the
	// harness cannot see the WebRTC surface at all.
	t.Run("webrtc_ice_known_red", func(t *testing.T) {
		ice := evalDetect(t, "webrtcIce")
		if ice == "undefined" {
			t.Errorf("KNOWN-RED WebRTC signal unpopulated — harness cannot observe "+
				"the WebRTC surface; got: %q", ice)
		}
		// Record the baseline truth in the test log for traceability.
		t.Logf("KNOWN-RED webrtcIce baseline truth: %q (flips to required-empty in Phase 27 HARDEN-01)", ice)
	})

	// FINGERPRINT-03 (required-green, was KNOWN-RED): the Client-Hints version is
	// now DERIVED from the active UA (Plan 03 killed the hardcoded "121" literal in
	// the rod-cli interceptor; Plan 02 made godoll's navigator.userAgentData
	// UA-derived). We assert the triple-agreement from the LIVE page:
	//   live Sec-Ch-Ua major == UA Chrome major == navigator.userAgentData version.
	// network_evasion_test.go owns the full header dump; here we assert only the
	// version coherence baseline. No t.Skip, no t.Logf-only baseline — a future
	// regression that re-introduces a mismatch FAILS this subtest.
	t.Run("client_hints_ua_derived", func(t *testing.T) {
		// Activate Client-Hints spoofing for this subtest (it is opt-in — see
		// writeProfile and the SUMMARY coherence note). Use the DEFAULT-OS identity
		// (Windows Chrome/121, Win32) so this proves the default-identity story is
		// coherent once CH is enabled. Close the default daemon, spawn pinned, and
		// restore the default daemon for any following subtests.
		prof := writeProfile(t, map[string]interface{}{
			"userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
				"(KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
			"platform":         "Win32",
			"spoofClientHints": true,
		})
		runCli("close")
		if _, err := runCli("--profile", prof, "goto", ds.URL()); err != nil {
			t.Fatalf("goto with CH profile failed: %v", err)
		}
		waitForDetectReady(t)
		defer restoreDefaultDaemon(t, ds.URL())

		// (1) Read the live Sec-Ch-Ua header the SHIPPED binary actually sent, via
		// a loopback header-echo server (zero network egress).
		chMajor := liveSecChUaMajor(t)

		// (2) Read the live UA's Chrome major from the page.
		uaMajor := chromeMajor(liveEval(t, "navigator.userAgent"))

		// (3) Read the live navigator.userAgentData Google-Chrome brand version.
		uadMajor := liveUserAgentDataMajor(t)

		if chMajor == "" || uaMajor == "" || uadMajor == "" {
			t.Fatalf("FINGERPRINT-03: a major version is empty (cannot assert coherence): "+
				"Sec-Ch-Ua=%q UA=%q userAgentData=%q", chMajor, uaMajor, uadMajor)
		}
		if !(chMajor == uaMajor && uaMajor == uadMajor) {
			t.Errorf("FINGERPRINT-03: Client-Hints major NOT UA-derived — surfaces disagree: "+
				"Sec-Ch-Ua=%s UA=%s navigator.userAgentData=%s (all three must match)",
				chMajor, uaMajor, uadMajor)
		}
		if uaMajor != "121" {
			t.Errorf("FINGERPRINT-03: default identity UA major changed unexpectedly, got %s (want 121)", uaMajor)
		}
	})

	// consistency_invariant — SUCCESS CRITERION 1 (FINGERPRINT-01/02), a BLOCKING
	// harness test. It proves Sec-CH-UA, navigator.userAgentData, the UA, and
	// navigator.platform all tell ONE OS+version story, read from the LIVE page
	// (never a Go config field), with zero network egress. A disagreement among
	// any surface fails the phase gate.
	t.Run("consistency_invariant", func(t *testing.T) {
		// Activate Client-Hints spoofing (opt-in; see writeProfile + SUMMARY) on the
		// default-OS identity so the BLOCKING invariant proves real coherence across
		// every surface rather than passing vacuously on empty CH.
		prof := writeProfile(t, map[string]interface{}{
			"userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
				"(KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
			"platform":         "Win32",
			"spoofClientHints": true,
		})
		runCli("close")
		if _, err := runCli("--profile", prof, "goto", ds.URL()); err != nil {
			t.Fatalf("goto with CH profile failed: %v", err)
		}
		waitForDetectReady(t)
		defer restoreDefaultDaemon(t, ds.URL())

		// Read ALL live-page (JS) signals FIRST, while the detect fixture page (with
		// window.__detect injected) is current. The header reads below navigate the
		// session to the loopback echo server, after which window.__detect is gone —
		// so order matters.
		uaString := liveEval(t, "navigator.userAgent")
		uaMajor := chromeMajor(uaString)
		uadMajor := liveUserAgentDataMajor(t)
		platform := liveEval(t, "navigator.platform")
		vendor := evalDetect(t, "webglVendor")
		renderer := evalDetect(t, "webglRenderer")

		// Now the header surface (navigates to the loopback echo server, last).
		chMajor := liveSecChUaMajor(t)
		chPlatform := stripQuotes(liveHeader(t, "Sec-Ch-Ua-Platform"))

		// --- Version story: Sec-Ch-Ua major == UA Chrome major == userAgentData ---
		if chMajor == "" || uaMajor == "" || uadMajor == "" {
			t.Fatalf("consistency_invariant: empty major — Sec-Ch-Ua=%q UA=%q userAgentData=%q",
				chMajor, uaMajor, uadMajor)
		}
		if !(chMajor == uaMajor && uaMajor == uadMajor) {
			t.Errorf("consistency_invariant: VERSION story disagrees — "+
				"Sec-Ch-Ua=%s UA=%s userAgentData=%s", chMajor, uaMajor, uadMajor)
		}

		// --- OS story: UA OS token == navigator.platform == Sec-Ch-Ua-Platform ---
		uaOS := osFromUA(uaString)
		if uaOS == "" {
			t.Errorf("consistency_invariant: could not classify OS from UA %q", uaString)
		}
		// platform must match the UA-implied OS family.
		if got := osFromPlatform(platform); got != uaOS {
			t.Errorf("consistency_invariant: OS story disagrees — UA implies %s but "+
				"navigator.platform=%q (=>%s)", uaOS, platform, got)
		}
		// Sec-Ch-Ua-Platform must match the UA-implied OS family.
		if chPlatform != "" {
			if got := osFromChPlatform(chPlatform); got != uaOS {
				t.Errorf("consistency_invariant: OS story disagrees — UA implies %s but "+
					"Sec-Ch-Ua-Platform=%q (=>%s)", uaOS, chPlatform, got)
			}
		}

		// --- WebGL: must not be a software rasterizer (headless tell) ------------
		combined := strings.ToLower(vendor + " " + renderer)
		for _, tell := range []string{"swiftshader", "llvmpipe", "software"} {
			if strings.Contains(combined, tell) {
				t.Errorf("consistency_invariant: WebGL reports software renderer %q "+
					"(headless tell): vendor=%q renderer=%q", tell, vendor, renderer)
			}
		}
	})

	// pinned_identity_macos — FINGERPRINT-01 (the --user-agent pin reaches the live
	// page end-to-end) + FINGERPRINT-03 (UA-derived CH) on a NON-DEFAULT identity.
	// Pin a macOS Chrome/130 UA via the --user-agent GLOBAL flag (stealth flags
	// apply at daemon SPAWN, so close first) with --platform MacIntel, and enable
	// Client-Hints via a --profile (opt-in; CLI flags override the profile UA/
	// platform per ResolveStealth precedence). Then read EVERY surface back from
	// the live page and assert the pinned 130 / MacIntel story. Zero egress (only
	// ds.URL() and the loopback header-echo). Closes the pinned session at the end
	// so it never bleeds into a later subtest (daemon-leak guard).
	t.Run("pinned_identity_macos", func(t *testing.T) {
		const macUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
			"(KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"
		// Profile only supplies spoofClientHints=true (the CLI flags below pin the
		// actual identity); without it the CH surfaces would be empty, not coherent.
		prof := writeProfile(t, map[string]interface{}{"spoofClientHints": true})

		runCli("close")
		out, err := runCli("--user-agent", macUA, "--platform", "MacIntel",
			"--profile", prof, "goto", ds.URL())
		if err != nil {
			t.Fatalf("pinned spawn (goto) failed: %v\nOutput: %s", err, out)
		}
		// The pinned identity is internally consistent (Mac UA + MacIntel), so the
		// daemon-spawn consistency gate must NOT reject it.
		if strings.Contains(strings.ToLower(out), "warning:") ||
			strings.Contains(strings.ToLower(out), "contradict") {
			t.Errorf("pinned spawn emitted an unexpected warning/rejection: %s", out)
		}
		waitForDetectReady(t)
		defer restoreDefaultDaemon(t, ds.URL())

		// --- Live-page JS surfaces first (window.__detect present here) ----------
		ua := liveEval(t, "navigator.userAgent")
		if !strings.Contains(ua, "Chrome/130") {
			t.Errorf("pinned UA did not reach the live page: got %q (want Chrome/130)", ua)
		}
		if strings.Contains(ua, "HeadlessChrome") {
			t.Errorf("pinned live UA leaks HeadlessChrome: %q", ua)
		}
		if platform := liveEval(t, "navigator.platform"); platform != "MacIntel" {
			t.Errorf("pinned navigator.platform: got %q (want MacIntel)", platform)
		}
		if uad := liveUserAgentDataMajor(t); uad != "130" {
			t.Errorf("pinned navigator.userAgentData major: got %q (want 130)", uad)
		}

		// --- Header surfaces last (navigates to the loopback echo server) --------
		if chMajor := liveSecChUaMajor(t); chMajor != "130" {
			t.Errorf("pinned Sec-Ch-Ua major: got %q (want 130)", chMajor)
		}
		if chPlat := stripQuotes(liveHeader(t, "Sec-Ch-Ua-Platform")); chPlat != "macOS" {
			t.Errorf("pinned Sec-Ch-Ua-Platform: got %q (want macOS)", chPlat)
		}
	})

	// stealth_check — VALIDATE-01 (per-signal verdict read from the LIVE page) +
	// VALIDATE-02 (token-efficient single-line --raw with only failing signals, no
	// page dump). Drives the stealth-check command three ways against the offline
	// fixture (zero egress): default human table, --raw single line, --json object.
	t.Run("stealth_check", func(t *testing.T) {
		// Ensure we are on the default detect fixture (prior pinned subtest restored
		// the default daemon via defer; re-goto is a cheap idempotent guard).
		if _, err := runCli("goto", ds.URL()); err != nil {
			t.Fatalf("stealth_check: goto fixture failed: %v", err)
		}
		waitForDetectReady(t)

		// (1) Default human mode: must name multiple per-signal verdicts.
		human, err := runCli("stealth-check")
		if err != nil {
			t.Fatalf("stealth-check (human) failed: %v\nOutput: %s", err, human)
		}
		for _, label := range []string{"webdriver", "webglVendor"} {
			if !strings.Contains(human, label) {
				t.Errorf("stealth-check human output missing per-signal label %q; got:\n%s", label, human)
			}
		}

		// (2) --raw: a SINGLE trimmed line starting with PASS or FAIL. If FAIL it
		// lists only name=FAIL(reason) failing tokens — never a full-signal/page
		// dump (e.g. it must not embed the full UA string).
		rawOut, err := runCli("--raw", "stealth-check")
		if err != nil {
			t.Fatalf("stealth-check --raw failed: %v\nOutput: %s", err, rawOut)
		}
		raw := strings.TrimSpace(rawOut)
		if strings.Contains(raw, "\n") {
			t.Errorf("--raw stealth-check is NOT a single line:\n%q", raw)
		}
		if !(strings.HasPrefix(raw, "PASS") || strings.HasPrefix(raw, "FAIL")) {
			t.Errorf("--raw stealth-check must start with PASS or FAIL, got: %q", raw)
		}
		if strings.Contains(raw, "Mozilla/5.0") || strings.Contains(raw, "AppleWebKit") {
			t.Errorf("--raw stealth-check leaked a full-signal/page dump (UA string): %q", raw)
		}
		if strings.HasPrefix(raw, "FAIL") {
			// Failing form carries only name=FAIL(reason) tokens — assert the shape.
			if !strings.Contains(raw, "=FAIL(") {
				t.Errorf("--raw FAIL line missing name=FAIL(reason) tokens: %q", raw)
			}
		}

		// (3) --json: must parse as a JSON object.
		jsonOut, err := runCli("--json", "stealth-check")
		if err != nil {
			t.Fatalf("stealth-check --json failed: %v\nOutput: %s", err, jsonOut)
		}
		js := strings.TrimSpace(jsonOut)
		start := strings.Index(js, "{")
		end := strings.LastIndex(js, "}")
		if start < 0 || end < start {
			t.Fatalf("stealth-check --json is not a JSON object: %s", js)
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(js[start:end+1]), &parsed); err != nil {
			t.Errorf("stealth-check --json did not parse as JSON: %v\nOutput: %s", err, js)
		}

		// CR-01 regression guard: the shipped default stealth masks/deletes
		// navigator.webdriver (godoll scriptHideAutomation), so it reads back as
		// `undefined` on the LIVE page. computeVerdict MUST treat that as PASS —
		// otherwise stealth-check FAILs against the binary's own correct stealth.
		// Assert the actual per-signal verdict (not just output shape) so this
		// defect cannot silently regress.
		signals, ok := parsed["signals"].(map[string]interface{})
		if !ok {
			t.Fatalf("stealth-check --json missing 'signals' object; got: %s", js)
		}
		wd, ok := signals["webdriver"].(map[string]interface{})
		if !ok {
			t.Fatalf("stealth-check --json missing webdriver signal; got: %s", js)
		}
		if verdict, _ := wd["verdict"].(string); verdict != "PASS" {
			t.Errorf("CR-01: webdriver MUST PASS on the default-stealthed page "+
				"(undefined/false are non-tells); got verdict=%q value=%q reason=%q",
				verdict, wd["value"], wd["reason"])
		}
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
