package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestSmokeAntigravity drives one session through the commands the
// antigravity-borrow plan changed and checks each output format.
func TestSmokeAntigravity(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")
	defer runCli("close")

	run := func(args ...string) string {
		t.Helper()
		out, err := runCli(append([]string{"--raw"}, args...)...)
		if err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(out)
	}
	match := func(step, pattern, out string) []string {
		t.Helper()
		m := regexp.MustCompile(pattern).FindStringSubmatch(out)
		if m == nil {
			t.Errorf("%s: output %q does not match %s", step, out, pattern)
		}
		return m
	}

	run("goto", ts.URL+"/page1")

	// T04: tab-new prints "New tab created: <ID> <Title> (<url>)".
	match("tab-new", `^New tab created: [0-9A-F]{8} .*\(`+regexp.QuoteMeta(ts.URL+"/page2")+`\)$`,
		run("tab-new", ts.URL+"/page2"))

	// T10: snapshot -i keeps headings and controls (page2 holds one heading).
	match("snapshot -i", `heading "Page 2"`, run("snapshot", "-i"))

	// T06: mousewheel prints "scrolled dy=<moved> y=<pos>/<max>[ end]".
	match("mousewheel", `^scrolled dy=-?\d+ y=\d+/\d+( end)?$`, run("mousewheel"))

	// T04: tab-list marks the current tab with "*" and an 8-char id.
	cur := match("tab-list", `(?m)^\* \[(\d+)\] ([0-9A-F]{8}) .*page2\)$`, run("tab-list"))
	if cur == nil {
		t.FailNow()
	}

	// T04: closing the current tab by id reports the new current tab.
	match("tab-close", `^Closed tab `+cur[1]+` `+cur[2]+`; current: \[\d+\] [0-9A-F]{8} .*page1\)$`,
		run("tab-close", cur[2]))

	// T01: eval prints the bare value.
	if got := run("eval", "1+1"); got != "2" {
		t.Errorf("eval: got %q, want %q", got, "2")
	}

	// T12: screenshot --out prints "saved <abs path> <W>x<H>".
	png := filepath.Join(t.TempDir(), "smoke.png")
	match("screenshot", `^saved `+regexp.QuoteMeta(png)+` \d+x\d+$`, run("screenshot", "--out", png))
	if _, err := os.Stat(png); err != nil {
		t.Errorf("screenshot file: %v", err)
	}
}
