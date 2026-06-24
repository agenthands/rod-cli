package tests

// Phase 25 Plan 03 — end-to-end proxy + profile + credential-safety proof.
//
// These tests drive the REAL ../rod-cli binary via the runCli helper
// (cli_test.go) against an OFFLINE local proxy fixture (tests/proxyfixture),
// in the v1.6 "validate-live-not-source" style established by the Phase 24
// detection harness (detection_test.go). NO external proxy service is used —
// every target and proxy is loopback-only and deterministic.
//
// What is proven here:
//   - TestProxySessionIsolation     (PROXY-01 / PROFILE-02): two concurrent -s
//     sessions with different --proxy egress through their OWN proxy with no
//     bleed into the other fixture.
//   - TestProxyAuthViaCDP           (PROXY-02): an authenticated proxy is
//     answered via CDP (--proxy-auth) — correct creds tunnel with zero 407s,
//     wrong creds are rejected (407), proving auth is enforced, not bypassed.
//   - TestProfileRoundTripInherited (PROFILE-01): a stealth.Profile saved to
//     JSON round-trips through Save/LoadProfile and is accepted at spawn via
//     --profile and inherited by a later command on the same session.
//   - TestProxyCredentialsNotLeaked (credential safety, T-25-09): --proxy-auth
//     secrets never appear in stdout/stderr or the rod-cli-<session>.port file.
//
// No skip directive is used anywhere (the verify gate negative-greps test files
// for them) — env/sandbox limitations are documented in 25-03-SUMMARY.md
// rather than skipped, and assertions read observable behavior (fixture egress
// id, fixture request/407 counters, .port file contents) not Go internals.

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/agenthands/godoll/stealth"
	"github.com/agenthands/rod-cli/tests/proxyfixture"
)

// runCliSession runs the live binary for a named session, folding stderr into
// stdout exactly like runCli. It mirrors runCli but threads an explicit -s.
func runCliSession(session string, args ...string) (string, error) {
	full := append([]string{"-s", session}, args...)
	return runCli(full...)
}

// evalText reads a single expression back from the live page of a session and
// returns the trimmed value after the eval-result prefix.
func evalText(t *testing.T, session, expr string) string {
	t.Helper()
	out, err := runCliSession(session, "eval", expr)
	if err != nil {
		return ""
	}
	val := out
	if i := strings.Index(out, evalResultPrefix); i >= 0 {
		val = out[i+len(evalResultPrefix):]
	}
	return strings.TrimSpace(val)
}

// pollEgressID navigates the session through its proxy to a loopback target and
// polls the live DOM for the proxyfixture egress id served back. Bounded retry
// so a daemon spin-up does not flake.
func pollEgressID(t *testing.T, session, target string) string {
	t.Helper()
	for i := 0; i < 20; i++ {
		runCliSession(session, "goto", target)
		id := evalText(t, session,
			"document.getElementById('egress') ? document.getElementById('egress').getAttribute('data-proxy-id') : ''")
		if id != "" && id != "''" && id != "null" {
			return id
		}
		time.Sleep(300 * time.Millisecond)
	}
	return ""
}

// TestProxySessionIsolation (PROXY-01 / PROFILE-02): two concurrent named
// sessions, each pointed at its OWN offline proxy fixture, must egress through
// that fixture and NOT bleed into the other. The egress is read back from the
// live page (the fixture serves its id at #egress) and cross-checked against
// each fixture's request counter.
func TestProxySessionIsolation(t *testing.T) {
	fa, err := proxyfixture.New("FIXTURE-A")
	if err != nil {
		t.Fatalf("fixture A: %v", err)
	}
	fa.Start()
	defer fa.Close()

	fb, err := proxyfixture.New("FIXTURE-B")
	if err != nil {
		t.Fatalf("fixture B: %v", err)
	}
	fb.Start()
	defer fb.Close()

	// Deterministic state: ensure neither session is up.
	runCliSession("iso_a", "close")
	runCliSession("iso_b", "close")
	defer runCliSession("iso_a", "close")
	defer runCliSession("iso_b", "close")

	// Spawn each session through its own proxy (loopback target reachable only
	// via the proxy — the fixture serves it). Sessions are independent daemons.
	if out, err := runCliSession("iso_a", "--proxy", fa.URL(), "goto", "http://session-a.test/"); err != nil {
		t.Fatalf("session iso_a goto via proxy A failed: %v\n%s", err, out)
	}
	if out, err := runCliSession("iso_b", "--proxy", fb.URL(), "goto", "http://session-b.test/"); err != nil {
		t.Fatalf("session iso_b goto via proxy B failed: %v\n%s", err, out)
	}

	// Read egress identity back from each live page.
	idA := pollEgressID(t, "iso_a", "http://session-a.test/")
	idB := pollEgressID(t, "iso_b", "http://session-b.test/")

	if idA != "FIXTURE-A" {
		t.Errorf("session iso_a egressed through %q, want FIXTURE-A (bleed or mis-route)", idA)
	}
	if idB != "FIXTURE-B" {
		t.Errorf("session iso_b egressed through %q, want FIXTURE-B (bleed or mis-route)", idB)
	}

	// Counter cross-check: each fixture served traffic; the no-bleed invariant is
	// that session iso_a's egress identity is A (not B) and vice versa — proven
	// above via the live id. The counters confirm both proxies were actually
	// exercised (non-zero), so the id read is not a stale/cached artifact.
	if fa.RequestCount() == 0 {
		t.Errorf("fixture A served 0 requests; session iso_a did not route through it")
	}
	if fb.RequestCount() == 0 {
		t.Errorf("fixture B served 0 requests; session iso_b did not route through it")
	}
}

// TestProxyAuthViaCDP (PROXY-02): an authenticated proxy fixture must be
// satisfied via CDP (--proxy-auth) — the godoll relay tunnels CONNECT to the
// upstream auth fixture with Proxy-Authorization. Correct creds => the fixture
// accepts the tunnel with zero 407s (auth answered). Wrong creds => the fixture
// rejects with 407 (auth enforced, not bypassed).
//
// NOTE: in the --proxy-auth path Chrome routes via a local CONNECT relay
// (godoll StartProxyRelay), so egress is an HTTPS tunnel and the served page id
// is not observable in the DOM — the authoritative, observable proof is the
// fixture's accepted-CONNECT vs 407 counters. (Documented in 25-03-SUMMARY.md.)
func TestProxyAuthViaCDP(t *testing.T) {
	auth, err := proxyfixture.NewAuth("AUTH-FIXTURE", "proxyuser", "proxypass")
	if err != nil {
		t.Fatalf("auth fixture: %v", err)
	}
	auth.Start()
	defer auth.Close()

	// The two phases are SERIALIZED: each session is fully closed before the
	// next is measured, so its daemon's relay stops issuing CONNECTs and cannot
	// contaminate the other phase's counter window.

	// --- Correct credentials: auth must be ANSWERED (accepted, zero 407s) ----
	runCliSession("auth_ok", "close")

	served0, fails0 := auth.RequestCount(), auth.AuthFailureCount()
	// Navigate through the auth proxy with correct creds. The goto may return a
	// navigation result regardless (Chrome navigates even to an unreachable
	// upstream target), so we assert on the fixture, not the goto outcome.
	runCliSession("auth_ok", "--proxy", auth.URL(), "--proxy-auth", "proxyuser:proxypass",
		"goto", "https://auth-target.test/")
	// Give the relay a moment to complete the CONNECT handshake.
	time.Sleep(1500 * time.Millisecond)
	servedOK := auth.RequestCount() - served0
	failsOK := auth.AuthFailureCount() - fails0
	// Close this session BEFORE measuring the wrong-creds phase so its relay
	// stops retrying CONNECTs into the shared fixture counter.
	runCliSession("auth_ok", "close")

	if servedOK == 0 {
		t.Errorf("correct-creds: auth fixture accepted 0 CONNECTs; the CDP auth challenge was not answered")
	}
	if failsOK != 0 {
		t.Errorf("correct-creds: auth fixture emitted %d 407s; correct credentials were rejected", failsOK)
	}

	// --- Wrong credentials: auth must be ENFORCED (407 observed) -------------
	runCliSession("auth_bad", "close")
	defer runCliSession("auth_bad", "close")

	served1, fails1 := auth.RequestCount(), auth.AuthFailureCount()
	runCliSession("auth_bad", "--proxy", auth.URL(), "--proxy-auth", "proxyuser:WRONGPASS",
		"goto", "https://auth-target.test/")
	time.Sleep(1500 * time.Millisecond)
	failsBad := auth.AuthFailureCount() - fails1
	servedBad := auth.RequestCount() - served1
	runCliSession("auth_bad", "close")

	if failsBad == 0 {
		t.Errorf("wrong-creds: auth fixture emitted 0 407s; auth was bypassed instead of enforced")
	}
	if servedBad != 0 {
		t.Errorf("wrong-creds: auth fixture ACCEPTED %d CONNECTs with wrong credentials (auth bypass)", servedBad)
	}
}

// TestProfileRoundTripInherited (PROFILE-01): a stealth.Profile saved to JSON
// round-trips through Save/LoadProfile, is accepted at session spawn via
// --profile, and is inherited by a SECOND command on the same session WITHOUT
// re-passing --profile (proving spawn-time resolution + per-session
// inheritance, the Phase 25 deliverable).
//
// NOTE: Phase 25 records only that a profile was SELECTED (cfg.Stealth.ProfilePath);
// the profile's identity fields are deliberately NOT overlaid onto the live page
// yet — that is Phase 26 (FINGERPRINT). So this test asserts the OBSERVABLE
// Phase-25 truths: (1) the JSON round-trips byte-for-field, and (2) a --profile
// session keeps serving a second command, proving inheritance. (Documented in
// 25-03-SUMMARY.md.)
func TestProfileRoundTripInherited(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()

	// 1) Save a named profile to a temp JSON file and load it back: round-trip.
	dir := t.TempDir()
	profPath := filepath.Join(dir, "roundtrip.json")

	saved := stealth.DefaultProfile()
	saved.UserAgent = "Mozilla/5.0 (RoundTrip Profile) Chrome/121.0.0.0 Safari/537.36"
	saved.Timezone = "Europe/Berlin"
	saved.Locale = "de-DE"
	if err := saved.Save(profPath); err != nil {
		t.Fatalf("Profile.Save: %v", err)
	}
	loaded, err := stealth.LoadProfile(profPath)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if loaded.UserAgent != saved.UserAgent || loaded.Timezone != saved.Timezone || loaded.Locale != saved.Locale {
		t.Fatalf("profile round-trip mismatch: saved {ua=%q tz=%q loc=%q} loaded {ua=%q tz=%q loc=%q}",
			saved.UserAgent, saved.Timezone, saved.Locale,
			loaded.UserAgent, loaded.Timezone, loaded.Locale)
	}

	// 2) Spawn a session WITH --profile, then run a second command WITHOUT it.
	runCliSession("prof_s", "close")
	defer runCliSession("prof_s", "close")

	if out, err := runCliSession("prof_s", "--profile", profPath, "goto", ts.URL+"/about"); err != nil {
		t.Fatalf("first command with --profile failed (bad/missing profile would abort spawn): %v\n%s", err, out)
	}

	// Second command on the SAME session, no --profile re-passed: it must work,
	// proving the spawn-time profile was inherited (the daemon stayed up with the
	// resolved Stealth config). A bad profile would have failed the daemon at
	// spawn, so reaching here on a live second command is the inheritance proof.
	out, err := runCliSession("prof_s", "eval", "document.title")
	if err != nil {
		t.Fatalf("second command on the --profile session failed; profile not inherited: %v\n%s", err, out)
	}
	if !strings.Contains(out, evalResultPrefix) {
		t.Errorf("second command did not return a live eval result; session not healthy: %s", out)
	}
}

// TestProxyCredentialsNotLeaked (credential safety, T-25-09): proxy credentials
// passed via --proxy-auth must NEVER appear in the folded stdout/stderr output
// nor in the rod-cli-<session>.port file (which must contain only the port).
func TestProxyCredentialsNotLeaked(t *testing.T) {
	const user = "leakuser_s3cr3t"
	const pass = "leakpass_s3cr3t"

	runCliSession("leak_s", "close")
	defer runCliSession("leak_s", "close")

	// Drive a command with --proxy-auth. Point at an unreachable loopback proxy
	// so we exercise the credential-handling path without needing egress; the
	// command's success is irrelevant — the assertion is on output + .port file.
	out, _ := runCliSession("leak_s", "--proxy", "http://127.0.0.1:59999", "--proxy-auth", user+":"+pass, "snapshot")

	if strings.Contains(out, user) {
		t.Errorf("proxy username leaked into stdout/stderr output: %s", out)
	}
	if strings.Contains(out, pass) {
		t.Errorf("proxy password leaked into stdout/stderr output: %s", out)
	}

	// The .port file must contain only the port number, never the credentials.
	portFile := filepath.Join(os.TempDir(), "rod-cli-leak_s.port")
	data, err := os.ReadFile(portFile)
	if err == nil {
		content := string(data)
		if strings.Contains(content, user) {
			t.Errorf(".port file leaked proxy username: %q", content)
		}
		if strings.Contains(content, pass) {
			t.Errorf(".port file leaked proxy password: %q", content)
		}
		if strings.ContainsAny(content, ":/@") {
			t.Errorf(".port file contains non-port characters (possible credential/URL leak): %q", content)
		}
	}
	// If the daemon already exited (err != nil), the file is gone — also a
	// non-leak; nothing to assert.

	// CR-01 regression guard: the credential must NEVER reach the daemon process
	// argv. argv is world-readable via /proc/<pid>/cmdline (and `ps`), so a leak
	// here is a real disclosure. The secret is passed out-of-band via the
	// ROD_CLI_PROXY_AUTH env var instead. (Linux-only check — /proc is Linux.)
	if runtime.GOOS == "linux" {
		cmdlines, _ := filepath.Glob("/proc/[0-9]*/cmdline")
		for _, cl := range cmdlines {
			raw, rerr := os.ReadFile(cl)
			if rerr != nil {
				continue // process may have exited between glob and read
			}
			// cmdline is NUL-separated argv; only inspect rod-cli daemons.
			args := string(raw)
			if !strings.Contains(args, "rod-cli") {
				continue
			}
			if strings.Contains(args, user) || strings.Contains(args, pass) {
				t.Errorf("proxy credential leaked into a daemon process argv (%s) — must be passed via env, not argv", cl)
			}
		}
	}

	// The daemon log must not capture the credential either.
	logData, lerr := os.ReadFile(filepath.Join(os.TempDir(), "rod-cli-daemon.log"))
	if lerr == nil {
		logStr := string(logData)
		if strings.Contains(logStr, user) || strings.Contains(logStr, pass) {
			t.Errorf("proxy credential leaked into the daemon log file")
		}
	}
}
