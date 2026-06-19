package tests

import (
	"bytes"
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
		time.Sleep(500 * time.Millisecond) // Wait for daemon to fully exit
	}
	return out.String(), err
}

func TestStorageCommands(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()

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

	// 5. Cleanup
	runCli("close")
}
