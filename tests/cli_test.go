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
	// rod-cli itself now reorders global flags (reorderGlobalFlags, cmd.go) so
	// they work regardless of position. This helper still front-loads --json/
	// --raw/--headless purely to compute rest[0] (the real subcommand token)
	// reliably for the close-delay check below, and to inject the headless
	// default at a stable position — not to work around a parsing bug anymore.
	var globals, rest []string
	hasHeadlessFlag := false
	for _, a := range args {
		if a == "--json" || a == "--raw" {
			globals = append(globals, a)
			continue
		}
		if a == "--headless" || a == "--hl" || strings.HasPrefix(a, "--headless=") || strings.HasPrefix(a, "--hl=") {
			hasHeadlessFlag = true
		}
		rest = append(rest, a)
	}
	// Default to HEADLESS so the e2e suite never pops a visible Chrome window on a
	// workstation with a display — headless is the blocking CI gate; headful is an
	// opt-in (ROD_HEADFUL=1, or an explicit --headless=false which is respected
	// here). The flag is honored at daemon spawn (EnsureDaemon forwards it).
	if !hasHeadlessFlag && os.Getenv("ROD_HEADFUL") != "1" {
		globals = append(globals, "--headless")
	}
	finalArgs := append([]string{}, globals...)
	finalArgs = append(finalArgs, rest...)
	cmd := exec.Command(absPath, finalArgs...)
	cmd.Dir = ".."
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if len(rest) > 0 && rest[0] == "close" {
		time.Sleep(1 * time.Second)
	}
	return out.String(), err
}

func TestCookieGetCLIByName(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")
	defer runCli("close")

	if out, err := runCli("goto", ts.URL+"/storage"); err != nil {
		t.Fatalf("goto: %v\n%s", err, out)
	}
	for _, kv := range [][2]string{{"k1", "v1"}, {"k2", "v2"}} {
		if out, err := runCli("cookie-set", kv[0], kv[1]); err != nil {
			t.Fatalf("cookie-set %s: %v\n%s", kv[0], err, out)
		}
	}
	out, err := runCli("--raw", "cookie-get", "k2")
	if err != nil {
		t.Fatalf("cookie-get k2: %v\n%s", err, out)
	}
	if !strings.Contains(out, "v2") || strings.Contains(out, "v1") {
		t.Errorf("cookie-get k2 returned: %q", out)
	}
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
