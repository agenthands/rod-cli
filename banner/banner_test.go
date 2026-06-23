package banner

import (
	"strings"
	"testing"
)

func TestShowBanner_ContainsExpectedSubstrings(t *testing.T) {
	out := ShowBanner()
	if out == "" {
		t.Fatal("ShowBanner() returned empty string")
	}
	for _, want := range []string{"Rod CLI", "Go Rod Team", "Build:"} {
		if !strings.Contains(out, want) {
			t.Fatalf("ShowBanner() output missing %q; got:\n%s", want, out)
		}
	}
}
