package tests

import (
	"strings"
	"testing"
)

func TestEvalCommand(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")

	// Case 1: Eval without navigation
	out, err := runCli("--raw", "eval", "1 + 1")
	if err == nil {
		t.Errorf("Eval without navigation should fail: %s", out)
	}

	// Case 2: Valid arithmetic evaluation
	runCli("--raw", "goto", ts.URL)
	out, err = runCli("--raw", "eval", "10 + 20")
	if err != nil || !strings.Contains(out, "30") {
		t.Errorf("Eval arithmetic failed: err=%v out=%s", err, out)
	}

	// Case 3: DOM manipulation evaluation
	runCli("--raw", "eval", "document.body.innerHTML = 'evaltest'")
	out, err = runCli("--raw", "eval", "document.body.innerHTML")
	if err != nil || !strings.Contains(out, "evaltest") {
		t.Errorf("Eval DOM manipulation failed: err=%v out=%s", err, out)
	}

	// Case 4: Invalid JS Syntax
	out, err = runCli("--raw", "eval", "document.not.a.function()")
	if err == nil || !strings.Contains(out, "Exception") {
		t.Errorf("Eval invalid JS should fail gracefully: err=%v out=%s", err, out)
	}

	runCli("close")
}

func TestCookieCommands(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")

	runCli("--raw", "goto", ts.URL)

	// Case 1: Get empty cookies
	out, err := runCli("--raw", "cookie-get")
	// If it's empty, it returns `[]` or something similar, but it shouldn't error
	if err != nil {
		t.Errorf("cookie-get failed: err=%v out=%s", err, out)
	}

	// Case 2: Set cookie via eval and get it
	runCli("--raw", "eval", "document.cookie = 'mycookie=myvalue'")
	out, err = runCli("--raw", "cookie-get")
	if err != nil || !strings.Contains(out, "mycookie") || !strings.Contains(out, "myvalue") {
		t.Errorf("cookie-get did not find set cookie: err=%v out=%s", err, out)
	}

	// Case 3: Clear cookies
	runCli("--raw", "cookie-clear")
	out, err = runCli("--raw", "cookie-get")
	if err != nil || strings.Contains(out, "myvalue") {
		t.Errorf("cookie-clear did not remove cookie: err=%v out=%s", err, out)
	}

	runCli("close")
}

func TestHighlightCommands(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")

	runCli("--raw", "goto", ts.URL+"/forms")
	out, _ := runCli("--raw", "snapshot")

	submitRef := extractRef(out, "Submit")
	if submitRef == "" {
		t.Fatalf("Could not find Submit button in snapshot: %s", out)
	}

	// Case 1: Highlight missing element
	out, err := runCli("--raw", "highlight", "invalid_ref")
	if err == nil {
		t.Errorf("highlight missing ref should fail: %s", out)
	}

	// Case 2: Highlight valid element
	out, err = runCli("--raw", "highlight", submitRef)
	if err != nil || !strings.Contains(out, "Highlighted element") {
		t.Errorf("highlight failed: err=%v out=%s", err, out)
	}

	// Verify class was added
	out, _ = runCli("--raw", "eval", "document.querySelector('.rod-cli-highlighted') !== null")
	if !strings.Contains(out, "true") {
		t.Errorf("highlight did not inject class into DOM: %s", out)
	}

	// Case 3: Clear highlights
	out, err = runCli("--raw", "highlight-clear")
	if err != nil || !strings.Contains(out, "Highlights cleared") {
		t.Errorf("highlight-clear failed: err=%v out=%s", err, out)
	}

	// Verify class was removed
	out, _ = runCli("--raw", "eval", "document.querySelector('.rod-cli-highlighted') === null")
	if !strings.Contains(out, "true") {
		t.Errorf("highlight-clear did not remove class from DOM: %s", out)
	}

	runCli("close")
}
