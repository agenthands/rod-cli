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

	// Default path: no humanize flags ⇒ godoll's own default typing speed
	// (30-120ms/key). Asserts the pre-existing >=300ms humanize bound still holds,
	// i.e. the tuning surface did not regress the default path (criterion 3).
	defaultElapsed := typeOnce(t, nil)
	if defaultElapsed < 300*time.Millisecond {
		t.Errorf("zero-regression: default-path typing took %v, expected >= 300ms (existing humanize bound)", defaultElapsed)
	}

	// Slow path: pin a slow per-keystroke delay. Bounded (150-200ms) so the test
	// stays fast (~11 keys ⇒ ~2s) while being unambiguously slower than the
	// default. If the knob were silently ignored, this would clock the same as the
	// default and the assertion below would fail.
	slowElapsed := typeOnce(t, []string{"--typing-speed-min", "150", "--typing-speed-max", "200"})

	if slowElapsed <= defaultElapsed {
		t.Errorf("tunability: slow --typing-speed (%v) was not measurably longer than default (%v) — the knob appears to be ignored",
			slowElapsed, defaultElapsed)
	}
	// Require a clear margin so timing noise on the default path cannot produce a
	// false pass: slow must exceed default by at least 500ms.
	if slowElapsed < defaultElapsed+500*time.Millisecond {
		t.Errorf("tunability: slow typing (%v) did not exceed default (%v) by a clear margin (>=500ms) — tuning effect too small to be the knob",
			slowElapsed, defaultElapsed)
	}

	runCli("close")
}
