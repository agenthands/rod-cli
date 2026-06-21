package tests

import (
	"strings"
	"testing"
	"time"
)

func TestHumanizedInteractions(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()

	// Ensure clean daemon state before test
	runCli("close")

	out, err := runCli("--raw", "goto", ts.URL+"/forms")
	if err != nil {
		t.Fatalf("Failed to navigate: %v (output: %s)", err, out)
	}

	// Add ARIA label to username to extract its ref
	runCli("--raw", "eval", "document.getElementById('username').setAttribute('aria-label', 'UserField')")

	// Inject a script to count mousemove events on the document
	out, err = runCli("--raw", "eval", "() => { window.mouseMoveCount = 0; document.body.addEventListener('mousemove', () => { window.mouseMoveCount++; }); }")
	if err != nil {
		t.Fatalf("Failed to inject mousemove tracker: %v (out: %s)", err, out)
	}

	// Must take a snapshot before element interactions
	out, err = runCli("--raw", "snapshot")
	if err != nil {
		t.Fatalf("Failed to snapshot: %v (out: %s)", err, out)
	}

	submitRef := extractRef(out, "Submit")
	userFieldRef := extractRef(out, "UserField")

	if submitRef == "" || userFieldRef == "" {
		t.Fatalf("Failed to find element refs. Snapshot: %s", out)
	}

	// 1. Test Humanized Hover
	out, err = runCli("--raw", "hover", submitRef)
	if err != nil {
		t.Fatalf("Failed to hover: %v (out: %s)", err, out)
	}

	// Read the mouseMoveCount
	out, err = runCli("--raw", "eval", "window.mouseMoveCount")
	if err != nil {
		t.Fatalf("Failed to read mouseMoveCount: %v", err)
	}
	
	if strings.TrimSpace(out) == "0" {
		t.Errorf("Humanized hover failure: expected mousemove events > 0, got 0")
	}

	// 2. Test Humanized Typing
	start := time.Now()
	_, err = runCli("--raw", "type", userFieldRef, "Hello World")
	if err != nil {
		t.Fatalf("Failed to type: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed < 300*time.Millisecond {
		t.Errorf("Humanized typing failure: typing took %v, expected at least 300ms", elapsed)
	}

	runCli("close")
}
