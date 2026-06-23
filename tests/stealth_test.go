package tests

import (
	"strings"
	"testing"
)

func TestStealthInitialization(t *testing.T) {
	// Start the test server to get a real page context
	ts := SetupTestServer()
	defer ts.Close()

	// Ensure clean daemon state before test
	runCli("close")

	// Ensure the daemon is running and navigate to the page
	out, err := runCli("goto", ts.URL+"/dummy")
	if err != nil {
		t.Fatalf("Failed to navigate: %v (output: %s)", err, out)
	}

	// 1. Check navigator.webdriver (should be false/undefined, usually evaluate returns false)
	out, err = runCli("eval", "navigator.webdriver")
	if err != nil {
		t.Fatalf("Failed to eval navigator.webdriver: %v", err)
	}
	if !strings.Contains(out, "result: false") && !strings.Contains(out, "result: undefined") && !strings.Contains(out, "false") {
		t.Errorf("Stealth failure: expected navigator.webdriver to be false or undefined, got: %s", out)
	}

	// 2. Check navigator.plugins length (should not be 0 in a stealth headless browser)
	out, err = runCli("eval", "navigator.plugins.length > 0")
	if err != nil {
		t.Fatalf("Failed to eval plugins: %v", err)
	}
	if !strings.Contains(out, "true") {
		t.Errorf("Stealth failure: expected navigator.plugins.length > 0 to be true")
	}

	// 3. Check for a Chrome-family string in User-Agent, but NO "HeadlessChrome".
	// The stealth fingerprint generator randomizes the UA and may pick Chrome on
	// iOS ("CriOS"), which is still Chrome-family, so accept either spelling.
	out, err = runCli("eval", "navigator.userAgent")
	if err != nil {
		t.Fatalf("Failed to eval userAgent: %v", err)
	}
	ua := strings.TrimSpace(out)
	if !strings.Contains(ua, "Chrome") && !strings.Contains(ua, "CriOS") {
		t.Errorf("Expected userAgent to contain 'Chrome' or 'CriOS', got: %s", ua)
	}
	if strings.Contains(ua, "HeadlessChrome") {
		t.Errorf("Stealth failure: userAgent contains 'HeadlessChrome': %s", ua)
	}
	
	runCli("close")
}
