package tests

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestJsonOutputFormatting(t *testing.T) {
	runCli("close")

	// Case 1: Application-level validation error (Missing URL for goto)
	// We want to ensure that passing --json returns a valid JSON object with {"error": "..."}
	out, err := runCli("--json", "goto")
	
	// CLI exits with status 1 on error, so err should not be nil
	if err == nil {
		t.Errorf("goto without URL should have failed")
	}

	var jsonRes map[string]interface{}
	if parseErr := json.Unmarshal([]byte(out), &jsonRes); parseErr != nil {
		t.Fatalf("Failed to parse JSON output: %v, raw output: %s", parseErr, out)
	}

	if _, ok := jsonRes["error"]; !ok {
		t.Errorf("Expected JSON to contain 'error' key, got: %s", out)
	}

	// Case 2: Command execution error (Invalid ref for click)
	ts := SetupTestServer()
	defer ts.Close()
	runCli("--json", "goto", ts.URL)
	
	out, err = runCli("--json", "click", "invalid_ref_xyz")
	if err == nil {
		t.Errorf("Click with invalid ref should have failed")
	}

	if parseErr := json.Unmarshal([]byte(out), &jsonRes); parseErr != nil {
		t.Fatalf("Failed to parse JSON output: %v, raw output: %s", parseErr, out)
	}

	if _, ok := jsonRes["error"]; !ok {
		t.Errorf("Expected JSON to contain 'error' key for invalid click, got: %s", out)
	}

	runCli("close")
}

func TestBrowserConnectionFailures(t *testing.T) {
	runCli("close")

	// Attempt to connect to an invalid CDP endpoint
	out, err := runCli("--json", "--cdp-endpoint", "ws://127.0.0.1:9999/invalid", "goto", "http://example.com")
	
	if err == nil {
		t.Errorf("Connecting to invalid CDP endpoint should have failed")
	}

	var jsonRes map[string]interface{}
	if parseErr := json.Unmarshal([]byte(out), &jsonRes); parseErr != nil {
		t.Fatalf("Failed to parse JSON output: %v, raw output: %s", parseErr, out)
	}

	errMsg, ok := jsonRes["error"].(string)
	if !ok || !strings.Contains(errMsg, "Error connecting to browser") && !strings.Contains(errMsg, "connect error") {
		t.Errorf("Expected connection error in JSON, got: %s", out)
	}

	runCli("close")
}

func TestStorageEdgeCases(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")

	runCli("--raw", "goto", ts.URL)

	// Case 1: Get non-existent localstorage key
	out, _ := runCli("--raw", "localstorage-get", "non_existent_key")
	// window.localStorage.getItem returns null, which evaluates to string "<nil>" or "null" in rod
	if !strings.Contains(out, "<nil>") && !strings.Contains(out, "null") {
		t.Errorf("Expected null/<nil> for non-existent key, got: %s", out)
	}

	// Case 2: Unknown storage action
	// We have to bypass the CLI and hit the daemon directly? Or just rely on daemon handling.
	// CLI limits the actions, so we can't easily send an unknown action via CLI args because
	// the commands are hardcoded in daemon.go switch statement.

	runCli("close")
}
