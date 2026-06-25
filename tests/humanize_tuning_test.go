package tests

import (
	"testing"
	"time"
)

// TestHumanizeTuning proves HUMANIZE-01's observable-behavior criteria against
// the REAL prebuilt binary:
//
//   - (criterion 2 / tunability) a configured slow --typing-speed measurably
//     lengthens per-keystroke timing vs the default path. A knob that is "set but
//     silently ignored" makes the slow path NOT slower and FAILS this test.
//   - (criterion 3 / zero regression) the default (no humanize flag) path stays
//     within the existing humanize bound, so the tuning surface adds no
//     wall-clock regression on the default path.
//
// Stealth/humanize flags are resolved ONCE at session spawn (the established
// model), so each path runs in its own freshly-spawned daemon: `close` first,
// then the spawning `goto` carries (or omits) the tuning flags.
//
// Operational traps respected: the suite must `go build -o rod-cli .` before this
// runs (runCli execs the prebuilt ../rod-cli); never pkill -f; never touch the
// user's Chromium; never `go test ./...`.
func TestHumanizeTuning(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()

	const text = "Hello World"

	// typeOnce spawns a fresh daemon with the given extra spawn flags, navigates
	// to the form, resolves the input ref, and returns the wall-clock elapsed for
	// a single humanized type of `text`. The tuning flags are passed on the
	// daemon-spawning `goto` so they are frozen onto the session config.
	typeOnce := func(t *testing.T, gotoArgs []string) time.Duration {
		t.Helper()
		runCli("close")

		out, err := runCli(append([]string{"--raw"}, append(gotoArgs, "goto", ts.URL+"/forms")...)...)
		if err != nil {
			t.Fatalf("goto failed: %v (out: %s)", err, out)
		}
		// Tag the username input so we can extract its ref from the snapshot.
		runCli("--raw", "eval", "document.getElementById('username').setAttribute('aria-label', 'UserField')")

		out, err = runCli("--raw", "snapshot")
		if err != nil {
			t.Fatalf("snapshot failed: %v (out: %s)", err, out)
		}
		ref := extractRef(out, "UserField")
		if ref == "" {
			t.Fatalf("could not find UserField ref. snapshot: %s", out)
		}

		start := time.Now()
		if _, err := runCli("--raw", "type", ref, text); err != nil {
			t.Fatalf("type failed: %v", err)
		}
		return time.Since(start)
	}

	// Both comparison paths pin --typo-rate 0 to REMOVE the high-variance term:
	// godoll's typo-correction sleeps are hardcoded constants (up to +400ms each,
	// independent of the speed config) firing at the default 0.02 rate on every
	// path. Left in, a default run that happens to inject a couple of typos while
	// the slow run injects none could erode the slow-vs-default gap toward the
	// margin — an intermittent false-fail. With typos disabled, the base-delay
	// difference is the ONLY signal, making the comparison deterministic.

	// Default-speed path (typos off): godoll's own default typing speed
	// (30-120ms/key). Asserts the pre-existing >=300ms humanize bound still holds,
	// i.e. the tuning surface did not regress the default-speed path (criterion 3).
	// With typos off the deterministic floor is 11 keys × 30ms = 330ms (plus a
	// space bonus), comfortably above the 300ms bound.
	defaultElapsed := typeOnce(t, []string{"--typo-rate", "0"})
	if defaultElapsed < 300*time.Millisecond {
		t.Errorf("zero-regression: default-speed typing took %v, expected >= 300ms (existing humanize bound)", defaultElapsed)
	}

	// Slow path: pin a slow per-keystroke delay (typos off). Bounded (300-400ms)
	// so the test stays fast (~11 keys ⇒ ~4s) while being unambiguously slower:
	// the slow base delay (11×~350ms ≈ 3.8s) vs default (11×~75ms ≈ 0.8s) gap is
	// ~3s, far above the required margin. A silently-ignored knob would clock the
	// same as default and fail.
	slowElapsed := typeOnce(t, []string{"--typing-speed-min", "300", "--typing-speed-max", "400", "--typo-rate", "0"})

	if slowElapsed <= defaultElapsed {
		t.Errorf("tunability: slow --typing-speed (%v) was not measurably longer than default (%v) — the knob appears to be ignored",
			slowElapsed, defaultElapsed)
	}
	// Require a clear margin so residual timing noise cannot produce a false pass:
	// with typos off and a 300-400ms pin, the real gap is ~3s, so >=1s is safe.
	if slowElapsed < defaultElapsed+1*time.Second {
		t.Errorf("tunability: slow typing (%v) did not exceed default (%v) by a clear margin (>=1s) — tuning effect too small to be the knob",
			slowElapsed, defaultElapsed)
	}

	runCli("close")
}
