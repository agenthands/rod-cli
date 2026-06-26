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
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	rodfp "github.com/agenthands/godoll/fingerprint"
	"github.com/agenthands/rod-cli/internal/detect"
	"github.com/agenthands/rod-cli/types"
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

	// REQUIRED-GREEN (Phase 27 HARDEN-01, was KNOWN-RED): the WebRTC surface must
	// leak no real host IP. Plan 03 wired both legs — the disable-non-proxied-UDP
	// browser preference (WithWebRTCLeakProtection) and the RTCPeerConnection JS
	// wrapper (EvadeWebRTC) — so the live page must now report a clean ICE state.
	// The probe records the connection-address of each ICE host candidate (field 4
	// of the SDP candidate line) regardless of form. PASS conditions (no leak):
	//   - "" (no candidate gathered — the clean WithWebRTCLeakProtection effect), OR
	//   - "no-RTCPeerConnection" (API absent), OR
	//   - every comma-separated token ends in ".local" (modern-Chrome mDNS masking).
	// FAIL: "undefined" (surface unobservable — loud-fail guard retained), an
	// "error:" string, or ANY token that parses as a routable IPv4/IPv6 address
	// (net.ParseIP succeeds and it is not a .local hostname) — that is a leaked real IP.
	t.Run("webrtc_ice", func(t *testing.T) {
		ice := evalDetect(t, "webrtcIce")
		if ice == "undefined" {
			t.Errorf("WebRTC signal unpopulated — harness cannot observe the WebRTC "+
				"surface; got: %q", ice)
			return
		}
		if strings.HasPrefix(ice, "error:") {
			t.Errorf("WebRTC probe errored: %q", ice)
			return
		}
		if ice == "" || ice == "no-RTCPeerConnection" {
			t.Logf("webrtc_ice clean: %q (no leaked host IP)", ice)
			return
		}
		// One or more candidate addresses were gathered. None may be a routable IP.
		for _, tok := range strings.Split(ice, ",") {
			tok = strings.TrimSpace(tok)
			if tok == "" {
				continue
			}
			if strings.HasSuffix(tok, ".local") {
				continue // mDNS-masked hostname, not a real IP
			}
			// net.ParseIP misses an IPv6 with a zone (fe80::1%eth0); strip the zone
			// before parsing so a zoned link-local can't slip through.
			probe := tok
			if z := strings.IndexByte(probe, '%'); z >= 0 {
				probe = probe[:z]
			}
			if ip := net.ParseIP(probe); ip != nil {
				t.Errorf("WebRTC leaked a real host IP %q (full signal: %q) — "+
					"EvadeWebRTC/WithWebRTCLeakProtection failed", tok, ice)
				continue
			}
			// Default-deny: a non-empty, non-.local token that we cannot positively
			// classify is treated as a potential leak rather than silently passed.
			t.Errorf("WebRTC reported an unrecognized non-.local candidate address %q "+
				"(full signal: %q) — treating as a potential leak (default-deny)", tok, ice)
		}
	})

	// REQUIRED-GREEN (Phase 27 HARDEN-02): seeded canvas noise must be STABLE within
	// a session. We draw deterministic fixed content (no Date / no Math.random) into
	// one canvas and read toDataURL twice; the seeded per-(seed,index) delta means
	// both reads must be byte-identical. The result string is "stable:<len>" when the
	// two reads agree (and len>0 guards against a blanked/unobservable surface) or
	// "UNSTABLE" when they differ — which would mean a per-call-random (buggy) noise
	// implementation, itself a detection tell.
	t.Run("canvas_noise_stable", func(t *testing.T) {
		expr := `(function(){` +
			`var c=document.createElement('canvas');c.width=200;c.height=50;` +
			`var x=c.getContext('2d');` +
			`x.fillStyle='#f60';x.fillRect(10,10,100,30);` +
			`x.fillStyle='#069';x.font='16px sans-serif';x.fillText('rod-cli-detect',12,30);` +
			`var r1=c.toDataURL();var r2=c.toDataURL();` +
			`return r1===r2?('stable:'+r1.length):'UNSTABLE';})()`
		out, err := runCli("eval", expr)
		if err != nil {
			t.Fatalf("eval canvas stability expr failed: %v\nOutput: %s", err, out)
		}
		val := out
		if i := strings.Index(out, evalResultPrefix); i >= 0 {
			val = out[i+len(evalResultPrefix):]
		}
		val = strings.TrimSpace(val)
		if val == "UNSTABLE" {
			t.Errorf("canvas noise is not stable-per-session: two reads differ")
			return
		}
		if !strings.HasPrefix(val, "stable:") {
			t.Errorf("canvas stability probe returned unexpected/blanked result: %q", val)
			return
		}
		lenStr := strings.TrimPrefix(val, "stable:")
		if lenStr == "" || lenStr == "0" {
			t.Errorf("canvas surface unobservable (data URL length %q) — blanked probe "+
				"must not masquerade as a pass", lenStr)
		}

		// Stability alone cannot distinguish "stable noise" from "no noise" (the
		// pre-Phase-27 no-op was also perfectly stable). Anchor it: draw a FLAT fill
		// and assert at least one read-back byte was perturbed off the fill value, so
		// a regression to a no-op canvas (noise disabled/broken) fails here too.
		applied := `(function(){var c=document.createElement('canvas');c.width=16;c.height=16;` +
			`var x=c.getContext('2d');x.fillStyle='rgb(128,128,128)';x.fillRect(0,0,16,16);` +
			`var d=x.getImageData(0,0,16,16).data;var n=0;` +
			`for(var i=0;i<d.length;i+=4){if(d[i]!==128)n++;}` +
			`return 'perturbed:'+n;})()`
		out2, err := runCli("eval", applied)
		if err != nil {
			t.Fatalf("eval canvas-applied expr failed: %v\nOutput: %s", err, out2)
		}
		v2 := out2
		if i := strings.Index(out2, evalResultPrefix); i >= 0 {
			v2 = out2[i+len(evalResultPrefix):]
		}
		v2 = strings.TrimSpace(v2)
		if v2 == "perturbed:0" {
			t.Errorf("canvas noise was NOT applied: a flat rgb(128) fill read back "+
				"unperturbed (got %q) — noise regressed to a no-op", v2)
		}
	})

	// REQUIRED-GREEN (Phase 27 HARDEN-02): seeded AUDIO noise must be STABLE within a
	// session. This is the regression guard for the CR-01 bug — godoll's patched
	// AudioBuffer.getChannelData returns a REFERENCE to the live internal buffer, so
	// perturbing it in place would COMPOUND on every re-read (a per-read-varying
	// audio fingerprint, the exact tell HARDEN-02 forbids). The fix copies-then-
	// perturbs; this subtest reads getChannelData twice in one session and asserts
	// the two reads are sample-identical, with a non-empty/observable guard so a
	// blanked AudioContext surface fails loudly instead of vacuously passing.
	t.Run("audio_noise_stable", func(t *testing.T) {
		expr := `(function(){` +
			`var ac=null;` +
			`var OAC=window.OfflineAudioContext||window.webkitOfflineAudioContext;` +
			`if(OAC){try{ac=new OAC(1,2048,44100);}catch(e){ac=null;}}` +
			`if(!ac){var AC=window.AudioContext||window.webkitAudioContext;if(AC){try{ac=new AC();}catch(e){ac=null;}}}` +
			`if(!ac)return 'no-AudioContext';` +
			`var buf=ac.createBuffer(1,2048,44100);var src=buf.getChannelData(0);` +
			`for(var i=0;i<src.length;i++)src[i]=Math.sin(i/10);` +
			`var a=buf.getChannelData(0);var b=buf.getChannelData(0);` +
			`var s1='',s2='';for(var j=0;j<src.length;j+=100){s1+=a[j].toFixed(8)+',';s2+=b[j].toFixed(8)+',';}` +
			`return s1===s2?('stable:'+s1.length):'UNSTABLE';})()`
		out, err := runCli("eval", expr)
		if err != nil {
			t.Fatalf("eval audio stability expr failed: %v\nOutput: %s", err, out)
		}
		val := out
		if i := strings.Index(out, evalResultPrefix); i >= 0 {
			val = out[i+len(evalResultPrefix):]
		}
		val = strings.TrimSpace(val)
		if val == "no-AudioContext" {
			t.Fatalf("AudioContext surface unobservable (no constructor) — cannot verify audio stability")
		}
		if val == "UNSTABLE" {
			t.Errorf("audio noise is not stable-per-session: two getChannelData reads " +
				"differ (CR-01 compounding-drift regression)")
			return
		}
		if !strings.HasPrefix(val, "stable:") {
			t.Errorf("audio stability probe returned unexpected/blanked result: %q", val)
			return
		}
		lenStr := strings.TrimPrefix(val, "stable:")
		if lenStr == "" || lenStr == "0" {
			t.Errorf("audio surface unobservable (sampled length %q) — blanked probe "+
				"must not masquerade as a pass", lenStr)
		}
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

	// builtin_profiles — PROF-02 HARNESS leg. Drive a representative SUBSET of the
	// built-in library (Phase 32) through the offline fixture via --profile=<name>
	// and assert the table-stakes signals read back from the LIVE page are coherent:
	// the profile's UA + platform reach the page, no HeadlessChrome, navigator.webdriver
	// is not a tell, WebGL is not a software rasterizer, and — built-ins enable
	// spoofClientHints — the Sec-Ch-Ua major + platform agree with the UA. A profile
	// failing a key signal FAILS the test ("profiles that fail key signals are not
	// shipped"). Each profile resolves to the EMBEDDED built-in (no temp file).
	//
	// Subset rationale: each profile is a full browser spawn, so driving all 6 would
	// bloat the suite. We pick one per OS family plus a HW/screen variant —
	// windows-11-chrome, windows-10-laptop (4-core / 1366x768), macos-applesilicon-chrome
	// (Retina MacIntel). The OFFLINE consistency gate types.TestBuiltinProfilesAreVetted
	// covers ALL 6 deterministically; this leg proves the live end-to-end story for a
	// representative subset.
	t.Run("builtin_profiles", func(t *testing.T) {
		defer restoreDefaultDaemon(t, ds.URL())
		subset := []string{"windows-11-chrome", "windows-10-laptop", "macos-applesilicon-chrome"}
		for _, name := range subset {
			name := name
			t.Run(name, func(t *testing.T) {
				prof, ok, err := types.LoadBuiltinProfile(name)
				if err != nil || !ok {
					t.Fatalf("LoadBuiltinProfile(%q): ok=%v err=%v", name, ok, err)
				}
				wantUAMajor := chromeMajor(prof.UserAgent)
				wantOS := osFromUA(prof.UserAgent)

				runCli("close")
				out, err := runCli("--profile", name, "goto", ds.URL())
				if err != nil {
					t.Fatalf("goto with built-in %q failed: %v\nOutput: %s", name, err, out)
				}
				// A built-in is coherent by construction; the daemon-spawn consistency
				// gate must NOT reject it or warn.
				if lo := strings.ToLower(out); strings.Contains(lo, "warning:") || strings.Contains(lo, "contradict") {
					t.Errorf("built-in %q spawn emitted an unexpected warning/rejection: %s", name, out)
				}
				waitForDetectReady(t)

				// Live-page JS surfaces FIRST (window.__detect present on the fixture page).
				ua := liveEval(t, "navigator.userAgent")
				if strings.Contains(ua, "HeadlessChrome") {
					t.Errorf("built-in %q live UA leaks HeadlessChrome: %q", name, ua)
				}
				if got := chromeMajor(ua); got != wantUAMajor {
					t.Errorf("built-in %q UA major: got %q want %q (UA=%q)", name, got, wantUAMajor, ua)
				}
				// godoll masks navigator.webdriver (reads back undefined/false) — a `true`
				// is the automation tell.
				if wd := liveEval(t, "navigator.webdriver"); wd == "true" {
					t.Errorf("built-in %q navigator.webdriver is true (automation tell)", name)
				}
				if platform := liveEval(t, "navigator.platform"); platform != prof.Platform {
					t.Errorf("built-in %q navigator.platform: got %q want %q", name, platform, prof.Platform)
				}
				vendor := evalDetect(t, "webglVendor")
				renderer := evalDetect(t, "webglRenderer")
				combined := strings.ToLower(vendor + " " + renderer)
				for _, tell := range []string{"swiftshader", "llvmpipe", "software"} {
					if strings.Contains(combined, tell) {
						t.Errorf("built-in %q WebGL software renderer %q (headless tell): vendor=%q renderer=%q",
							name, tell, vendor, renderer)
					}
				}

				// Header surfaces LAST (navigates to the loopback echo server). Built-ins
				// enable spoofClientHints, so CH must be populated and UA-derived.
				chMajor := liveSecChUaMajor(t)
				if chMajor != wantUAMajor {
					t.Errorf("built-in %q Sec-Ch-Ua major: got %q want %q (UA-derived)", name, chMajor, wantUAMajor)
				}
				// Unconditional: built-ins enable spoofClientHints, so an ABSENT
				// Sec-Ch-Ua-Platform is itself the regression this leg guards — fail on
				// empty rather than skipping (don't gate the assertion on its own subject).
				chPlat := stripQuotes(liveHeader(t, "Sec-Ch-Ua-Platform"))
				if chPlat == "" {
					t.Errorf("built-in %q Sec-Ch-Ua-Platform absent (spoofClientHints is on)", name)
				} else if got := osFromChPlatform(chPlat); got != wantOS {
					t.Errorf("built-in %q Sec-Ch-Ua-Platform OS family: got %q (=>%s) want %s",
						name, chPlat, got, wantOS)
				}
			})
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

// TestAdvancedEvasionDimensions proves the Phase 33 fingerprint-dimension
// hardening (EVAD-02 "asserted by harness"). All three legs read back from the
// LIVE page (validate-live-not-source), drive the shipped ../rod-cli binary
// against the loopback detect fixture, and run offline with zero network egress:
//
//   - COHERENCE (toggles ON / default): the new vectors read back
//     coherent/non-headless-default — mediaDevices enumerates a plausible device
//     set, getBattery reports a level in [0,1] + boolean charging, codecs report
//     plausible canPlayType support.
//   - STABILITY (success criterion 4): a vector reads identically on re-read
//     within a session.
//   - TOGGLE-OFF (independence): with --media-devices-spoof=false the vector
//     reverts to the un-hardened default reading, proving the toggle is effective.
//
// Coverage note: fonts ships a toggle + probe but godoll's scriptMockFonts is an
// observable no-op on measureText widths, so fonts is NOT asserted live here (see
// 33-02-SUMMARY). The 2+ EVAD-02 surfaces proven are mediaDevices + battery (+
// codecs); the toggle-off leg proves mediaDevices independence.
func TestAdvancedEvasionDimensions(t *testing.T) {
	ds, err := detect.New()
	if err != nil {
		t.Fatalf("detect.New() failed: %v", err)
	}
	ds.Start()
	defer ds.Close()

	runCli("close")
	defer runCli("close")

	// Default daemon (all dimension toggles ON) on the detect fixture.
	if _, err := runCli("goto", ds.URL()); err != nil {
		t.Fatalf("goto fixture failed: %v", err)
	}
	waitForDetectReady(t)

	// --- COHERENCE (toggles ON) ----------------------------------------------

	// mediaDevices: godoll injects >=3 devices (1-2 webcams + 1-2 micros + 1-2
	// speakers). Signature is "<count>:<sorted,kind,set>". Assert a plausible,
	// non-headless-default reading with only known device kinds.
	t.Run("media_devices_coherent_on", func(t *testing.T) {
		sig := evalDetect(t, "mediaDevices")
		if sig == "" || sig == "undefined" || strings.HasPrefix(sig, "error") ||
			sig == "no-mediaDevices-api" {
			t.Fatalf("mediaDevices unreadable/absent (sig=%q)", sig)
		}
		countStr, kinds := sig, ""
		if i := strings.Index(sig, ":"); i >= 0 {
			countStr, kinds = sig[:i], sig[i+1:]
		}
		n, perr := strconv.Atoi(countStr)
		if perr != nil || n < 1 {
			t.Errorf("mediaDevices count not plausible (headless-default tell?): %q", sig)
		}
		for _, k := range strings.Split(kinds, ",") {
			if k == "" {
				continue
			}
			switch k {
			case "videoinput", "audioinput", "audiooutput":
			default:
				t.Errorf("mediaDevices reported implausible kind %q (full sig %q)", k, sig)
			}
		}
	})

	// battery: getBattery overridden to resolve a fixed BatteryManager. The
	// injector only replaces an EXISTING getBattery, so a Chrome build without it
	// is not a hardening failure — skip in that case.
	t.Run("battery_coherent_on", func(t *testing.T) {
		if present := evalDetect(t, "batteryPresent"); present != "true" {
			t.Skipf("navigator.getBattery not present in this Chrome (battery=%q); injector "+
				"overrides only an existing API", evalDetect(t, "battery"))
		}
		lvlStr := evalDetect(t, "batteryLevel")
		lvl, perr := strconv.ParseFloat(lvlStr, 64)
		if perr != nil || lvl < 0 || lvl > 1 {
			t.Errorf("battery level not in [0,1] (coherence): %q", lvlStr)
		}
		if charging := evalDetect(t, "batteryCharging"); charging != "true" && charging != "false" {
			t.Errorf("battery charging not boolean (coherence): %q", charging)
		}
	})

	// codecs: canPlayType overridden. At least one representative type must report
	// support — an all-"no" result would be an implausible/blanked reading.
	t.Run("codecs_coherent_on", func(t *testing.T) {
		sig := evalDetect(t, "codecs")
		if sig == "" || sig == "undefined" || strings.HasPrefix(sig, "error") {
			t.Fatalf("codecs unreadable: %q", sig)
		}
		if !strings.Contains(sig, "=probably") && !strings.Contains(sig, "=maybe") {
			t.Errorf("codecs report no supported types (implausible): %q", sig)
		}
	})

	// --- CONSISTENCY-ON-REREAD (live, weak) ----------------------------------

	// A live guard that the injected vector is consistent (not re-randomized /
	// blanked) on re-read within the session. NOTE: this is a WEAK check — the
	// dimension is injected once and memoized into window.__detect, so two reads of
	// the same page are stable by construction. The LOAD-BEARING proof of success
	// criterion 4 (same session seed => same regenerated dimensions, so a recreated
	// page reproduces them) is the deterministic unit test
	// TestSeededFingerprintDimensions below, which exercises the seeded generator
	// directly without a browser.
	t.Run("media_devices_consistent_on_reread", func(t *testing.T) {
		s1 := evalDetect(t, "mediaDevices")
		s2 := evalDetect(t, "mediaDevices")
		if s1 == "" || strings.HasPrefix(s1, "error") {
			t.Fatalf("mediaDevices probe blanked: %q", s1)
		}
		if s1 != s2 {
			t.Errorf("mediaDevices inconsistent on re-read within session: %q vs %q", s1, s2)
		}
	})

	// --- TOGGLE-OFF (independence / effectiveness) ---------------------------

	// With --media-devices-spoof=false the dimension must NOT be injected, so the
	// vector reverts to the un-hardened headless default — a reading that differs
	// from the ON (mocked) signature. Proves the toggle is effective, not cosmetic.
	t.Run("media_devices_toggle_off_reverts", func(t *testing.T) {
		onSig := evalDetect(t, "mediaDevices")
		runCli("close")
		if _, err := runCli("--media-devices-spoof=false", "goto", ds.URL()); err != nil {
			t.Fatalf("goto with --media-devices-spoof=false failed: %v", err)
		}
		waitForDetectReady(t)
		defer restoreDefaultDaemon(t, ds.URL())

		offSig := evalDetect(t, "mediaDevices")
		if offSig == onSig {
			t.Errorf("media-devices-spoof toggle is INEFFECTIVE: ON and OFF signatures "+
				"identical (%q) — the off path still injected the mocked device set", onSig)
		}
	})
}

// fontSet returns a presence set of a font slice for subset/exclusion checks.
func fontSet(fonts []string) map[string]bool {
	m := make(map[string]bool, len(fonts))
	for _, f := range fonts {
		m[f] = true
	}
	return m
}

// TestSeededFingerprintDimensions is the deterministic, browser-free proof for two
// Phase 33 guarantees that the live harness cannot establish on its own:
//
//   - SUCCESS CRITERION 4 (stability): the godoll fingerprint generator is SEEDED
//     with the per-session noiseSeed, so the same seed reproduces the exact same
//     dimensions. That is what makes a page recreated within one session reproduce
//     identical fonts/devices/codecs/battery — the property the (necessarily weak)
//     live re-read subtest cannot exercise because ctx.page is created once per
//     session and `close` regenerates the seed.
//   - COHERENT-NOT-RANDOM (coherence): generation is constrained to the profile OS
//     (FPWithOS), so a Windows draw carries Windows-family fonts and a macOS draw
//     macOS-family — never an unconstrained random set. This is asserted here at
//     the generation layer because godoll's font injector is an observable no-op on
//     measureText, so OS-coherent fonts cannot be read back from the live page.
//
// Offline, no browser, no network — pure generator determinism.
func TestSeededFingerprintDimensions(t *testing.T) {
	const seed = int64(0x5EEDC0FFEE)
	gen := func(os string) *rodfp.Fingerprint {
		t.Helper()
		g := rodfp.NewFingerprintGeneratorSeeded(seed,
			rodfp.FPWithBrowserNames("chrome"), rodfp.FPWithOS(os))
		f, err := g.Generate()
		if err != nil {
			t.Fatalf("Generate(%s) failed: %v", os, err)
		}
		if f == nil {
			t.Fatalf("Generate(%s) returned nil fingerprint", os)
		}
		return f
	}

	// (1) DETERMINISM: same seed + same OS => byte-identical dimensions.
	t.Run("same_seed_same_dimensions", func(t *testing.T) {
		a, b := gen("windows"), gen("windows")
		if !reflect.DeepEqual(a.Fonts, b.Fonts) {
			t.Errorf("fonts not deterministic for same seed:\n a=%v\n b=%v", a.Fonts, b.Fonts)
		}
		if !reflect.DeepEqual(a.MediaDevices, b.MediaDevices) {
			t.Errorf("media devices not deterministic for same seed")
		}
		if !reflect.DeepEqual(a.VideoCodecs, b.VideoCodecs) ||
			!reflect.DeepEqual(a.AudioCodecs, b.AudioCodecs) {
			t.Errorf("codecs not deterministic for same seed")
		}
		if !reflect.DeepEqual(a.Battery, b.Battery) {
			t.Errorf("battery not deterministic for same seed")
		}
	})

	// (2) OS-COHERENCE: a Windows draw must contain NO mac/linux-only fonts (and
	// symmetrically), proving the font set tells the profile's OS story.
	t.Run("os_coherent_fonts", func(t *testing.T) {
		winOnly := []string{"Segoe UI", "Calibri", "Tahoma", "Consolas", "Cambria"}
		macOnly := []string{"Helvetica Neue", "Menlo", "Monaco", "Avenir", "Lucida Grande"}
		linuxOnly := []string{"DejaVu Sans", "Ubuntu", "Liberation Sans", "Noto Sans", "FreeSans"}

		excludes := func(os string, set map[string]bool, forbidden []string) {
			t.Helper()
			for _, f := range forbidden {
				if set[f] {
					t.Errorf("OS=%s font draw contains foreign font %q (incoherent — coherent-not-random)", os, f)
				}
			}
		}

		win := fontSet(gen("windows").Fonts)
		mac := fontSet(gen("macos").Fonts)
		lin := fontSet(gen("linux").Fonts)

		excludes("windows", win, append(append([]string{}, macOnly...), linuxOnly...))
		excludes("macos", mac, append(append([]string{}, winOnly...), linuxOnly...))
		excludes("linux", lin, append(append([]string{}, winOnly...), macOnly...))

		// The OS draws must differ from each other — not one shared random set.
		if reflect.DeepEqual(gen("windows").Fonts, gen("macos").Fonts) {
			t.Errorf("windows and macos font draws are identical — OS constraint not applied")
		}
	})
}

// fontProbeJS returns a JS expression that probes font availability via canvas
// measurement and returns a deterministic hash of the measured widths.
// The probe measures common fonts; font spoofing changes these widths.
const fontProbeJS = `(function(){
const c=document.createElement('canvas');
const x=c.getContext('2d');
const fs=['Arial','Times New Roman','Courier New','Georgia','Verdana',
  'Comic Sans MS','Impact','Trebuchet MS','Lucida Console','Palatino Linotype'];
var h=0;
for(var i=0;i<fs.length;i++){
x.font='72px '+fs[i]+',sans-serif';
var w=x.measureText('abcdefghijklmnopqrstuvwxyz').width;
h=((h<<5)-h+Math.round(w))|0;
}
return String(h);
})()`

// TestFontSpoof verifies FONT-01, FONT-02, and FONT-03:
//   - With font-spoof ON, font-probe hash differs from the host baseline.
//   - Within a session, two reads return the same hash (stability).
//   - font-spoof OFF restores the genuine host font behavior.
func TestFontSpoof(t *testing.T) {
	ds, err := detect.New()
	if err != nil {
		t.Fatalf("detect.New() failed: %v", err)
	}
	ds.Start()
	defer ds.Close()

	// --- Baseline: font-spoof OFF ---
	runCli("close")
	defer runCli("close")

	if _, err := runCli("--font-spoof=false", "goto", ds.URL()); err != nil {
		t.Fatalf("baseline goto failed: %v", err)
	}
	waitForDetectReady(t)

	baseline := evalFontHash(t)
	if baseline == "" || baseline == "0" {
		t.Fatal("font-probe returned empty baseline hash")
	}
	t.Logf("baseline font hash (off): %s", baseline)

	// --- FONT-02: stability within session ---
	baseline2 := evalFontHash(t)
	if baseline != baseline2 {
		t.Errorf("FONT-02: baseline re-read unstable: %q vs %q", baseline, baseline2)
	}

	// --- FONT-01: font-spoof ON changes the hash ---
	runCli("close")
	if _, err := runCli("--font-spoof=true", "goto", ds.URL()); err != nil {
		t.Fatalf("spoofed goto failed: %v", err)
	}
	waitForDetectReady(t)

	spoofed := evalFontHash(t)
	t.Logf("spoofed font hash (on): %s", spoofed)

	if spoofed == baseline {
		t.Errorf("FONT-01: font-spoof ON produced the same hash as OFF — injector is still a no-op (both=%q)", baseline)
	}

	// --- FONT-02: stability within session (spoofed) ---
	spoofed2 := evalFontHash(t)
	if spoofed != spoofed2 {
		t.Errorf("FONT-02: spoofed re-read unstable: %q vs %q", spoofed, spoofed2)
	}

	// --- FONT-02: font-spoof OFF restores baseline ---
	runCli("close")
	if _, err := runCli("--font-spoof=false", "goto", ds.URL()); err != nil {
		t.Fatalf("restore goto failed: %v", err)
	}
	waitForDetectReady(t)

	restored := evalFontHash(t)
	t.Logf("restored font hash (off again): %s", restored)

	if restored != baseline {
		t.Errorf("FONT-02: font-spoof OFF did not restore baseline: %q vs %q", restored, baseline)
	}
}

// evalFontHash runs the font-probe JS via eval and returns the string result.
func evalFontHash(t *testing.T) string {
	t.Helper()
	out, err := runCli("eval", fontProbeJS)
	if err != nil {
		t.Fatalf("eval font-probe failed: %v\nOutput: %s", err, out)
	}
	val := out
	if i := strings.Index(out, evalResultPrefix); i >= 0 {
		val = out[i+len(evalResultPrefix):]
	}
	return strings.TrimSpace(val)
}
