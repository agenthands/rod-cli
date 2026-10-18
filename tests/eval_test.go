package tests

import (
	"strings"
	"testing"
)

func TestEvalCLIBareValue(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	defer runCli("close")

	if out, err := runCli("--raw", "goto", ts.URL+"/page1"); err != nil {
		t.Fatalf("goto: %v\n%s", err, out)
	}
	out, err := runCli("--raw", "eval", "1+1")
	if err != nil {
		t.Fatalf("eval: %v\n%s", err, out)
	}
	if got := strings.TrimSpace(out); got != "2" {
		t.Errorf("eval output = %q, want %q", got, "2")
	}
}
