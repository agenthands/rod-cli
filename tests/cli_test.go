package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// runCli is a helper to run the compiled binary
func runCli(args ...string) (string, error) {
	absPath, _ := filepath.Abs("../rod-cli")
	// Prepend global flags
	args = append([]string{"--no-banner"}, args...)
	cmd := exec.Command(absPath, args...)
	cmd.Dir = ".."
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if len(args) > 0 && args[0] == "close" || (len(args) > 1 && args[1] == "close") {
		time.Sleep(1 * time.Second) // Wait for daemon to fully exit
	}
	return out.String(), err
}

func TestStorageCommands(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	defer os.Remove("state.json")

	// Ensure clean daemon state before test
	runCli("close")

	// 1. Navigate to the storage route
	out, err := runCli("goto", ts.URL+"/storage")
	if err != nil {
		t.Fatalf("Failed to goto: %v\nOutput: %s", err, out)
	}

	// 2. Test LocalStorage Set
	_, err = runCli("localstorage-set", "testkey", "testvalue")
	if err != nil {
		t.Fatalf("Failed to set localstorage: %v", err)
	}

	// 3. Test LocalStorage Get
	out, err = runCli("localstorage-get", "testkey")
	if err != nil {
		t.Fatalf("Failed to get localstorage: %v", err)
	}
	if !strings.Contains(out, "testvalue") {
		t.Errorf("Expected localstorage to contain 'testvalue', got: %s", out)
	}

	// 4. Test LocalStorage Clear
	runCli("localstorage-clear")
	out, err = runCli("localstorage-get", "testkey")
	if err == nil && strings.Contains(out, "testvalue") {
		t.Errorf("Expected localstorage to be empty after clear, got: %s", out)
	}

	// 5. Test LocalStorage Delete
	runCli("localstorage-set", "testkey2", "testvalue2")
	runCli("localstorage-delete", "testkey2")
	out, _ = runCli("localstorage-get", "testkey2")
	if strings.Contains(out, "testvalue2") {
		t.Errorf("Expected localstorage to be empty after delete, got: %s", out)
	}

	// 6. Test Cookies
	_, err = runCli("cookie-set", "mycookie", "myval")
	if err != nil {
		t.Fatalf("Failed to set cookie: %v", err)
	}
	out, _ = runCli("cookie-get")
	if !strings.Contains(out, "mycookie") || !strings.Contains(out, "myval") {
		t.Errorf("Expected cookie to be set, got: %s", out)
	}

	runCli("cookie-delete", "mycookie")
	out, _ = runCli("cookie-get")
	if strings.Contains(out, "mycookie") {
		t.Errorf("Expected cookie to be deleted, got: %s", out)
	}

	// 7. Test State Save and Load
	runCli("cookie-set", "statecookie", "stateval")
	out, err = runCli("state-save", "state.json")
	if err != nil || !strings.Contains(out, "Saved state") {
		t.Fatalf("Failed to save state: %v", err)
	}
	
	runCli("cookie-clear")
	out, _ = runCli("cookie-get")
	if strings.Contains(out, "statecookie") {
		t.Errorf("Expected cookie to be cleared, got: %s", out)
	}

	out, err = runCli("state-load", "state.json")
	if err != nil || !strings.Contains(out, "Loaded state") {
		t.Fatalf("Failed to load state: %v", err)
	}
	out, _ = runCli("cookie-get")
	if !strings.Contains(out, "statecookie") || !strings.Contains(out, "stateval") {
		t.Errorf("Expected cookie to be loaded, got: %s", out)
	}

	// 8. Cleanup
	runCli("close")
}
