package tests

import (
	"strings"
	"testing"
)

func TestGotoCommand(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")

	// Case 1: Valid URL
	out, err := runCli("--raw", "goto", ts.URL)
	if err != nil || !strings.Contains(out, "Navigated to") {
		t.Errorf("Goto valid URL failed: err=%v out=%s", err, out)
	}

	// Case 2: Missing URL
	out, err = runCli("--raw", "goto")
	if err == nil || !strings.Contains(out, "url is required") {
		t.Errorf("Goto missing URL should fail: err=%v out=%s", err, out)
	}

	// Case 3: Invalid URL format
	out, err = runCli("--raw", "goto", "not-a-url")
	if err == nil || !strings.Contains(out, "invalid URL") {
		t.Errorf("Goto invalid URL should fail: err=%v out=%s", err, out)
	}

	// Case 4: 404 URL (valid format, missing route)
	out, err = runCli("--raw", "goto", ts.URL+"/doesnotexist")
	if err != nil {
		t.Errorf("Goto 404 URL should pass navigation but render 404: err=%v out=%s", err, out)
	}

	// Case 5: URL with query params
	out, err = runCli("--raw", "goto", ts.URL+"/?q=test")
	if err != nil || !strings.Contains(out, "?q=test") {
		t.Errorf("Goto with query params failed: err=%v out=%s", err, out)
	}
	runCli("close")
}

func TestCloseCommand(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()

	// Case 1: Close without active session
	runCli("close")
	out, _ := runCli("--raw", "close")
	// closing should not hard fail even if no daemon is there
	if !strings.Contains(out, "closing") && out != "" {
		// Output could be silent if it couldn't connect
	}

	// Case 2: Close active session
	runCli("--raw", "goto", ts.URL)
	out, err := runCli("--raw", "close")
	if err != nil || !strings.Contains(out, "closing") {
		t.Errorf("Close active session failed: err=%v out=%s", err, out)
	}

	// Case 3: Verify session is actually closed
	out, err = runCli("--raw", "snapshot")
	if err == nil {
		t.Errorf("Snapshot should fail after close, got: %s", out)
	}

	// Case 4: Close named session
	runCli("--raw", "-s", "named_session", "goto", ts.URL)
	out, err = runCli("--raw", "-s", "named_session", "close")
	if err != nil || !strings.Contains(out, "closing") {
		t.Errorf("Close named session failed: err=%v out=%s", err, out)
	}

	// Case 5: Close different session doesn't affect default
	runCli("--raw", "goto", ts.URL)
	runCli("--raw", "-s", "other", "goto", ts.URL)
	runCli("--raw", "-s", "other", "close")
	// default should still be active
	out, err = runCli("--raw", "snapshot")
	if err != nil {
		t.Errorf("Default session incorrectly closed: %v", err)
	}
	runCli("close")
}

func TestReloadCommand(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")

	// Case 1: Reload without navigation
	out, err := runCli("--raw", "reload")
	if err == nil || !strings.Contains(out, "Failed to reload") {
		t.Errorf("Reload without goto should fail: err=%v out=%s", err, out)
	}

	// Case 2: Normal reload
	runCli("--raw", "goto", ts.URL+"/forms")
	out, err = runCli("--raw", "reload")
	if err != nil || !strings.Contains(out, "Reload current page successfully") {
		t.Errorf("Normal reload failed: err=%v out=%s", err, out)
	}

	// Case 3: Reload after JS modification
	runCli("--raw", "eval", "document.getElementById('username').value = 'reloaded'")
	runCli("--raw", "reload")
	out, _ = runCli("--raw", "eval", "document.getElementById('username').value")
	if strings.Contains(out, "reloaded") {
		t.Errorf("DOM state was not reset on reload: %s", out)
	}

	// Case 4: Multiple reloads
	runCli("--raw", "reload")
	out, err = runCli("--raw", "reload")
	if err != nil {
		t.Errorf("Multiple reloads failed: %v", err)
	}

	// Case 5: Reload on 404
	runCli("--raw", "goto", ts.URL+"/404")
	out, err = runCli("--raw", "reload")
	if err != nil {
		t.Errorf("Reload on 404 failed: %v", err)
	}
	runCli("close")
}

func TestHistoryCommands(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")

	// Case 1: Go back without history
	out, err := runCli("--raw", "go-back")
	if err == nil {
		t.Errorf("go-back without history should fail: %s", out)
	}

	// Case 2: Normal go-back
	runCli("--raw", "goto", ts.URL+"/page1")
	runCli("--raw", "goto", ts.URL+"/page2")
	out, err = runCli("--raw", "go-back")
	if err != nil || !strings.Contains(out, "Go back successfully") {
		t.Errorf("Normal go-back failed: err=%v out=%s", err, out)
	}

	// Case 3: Verify state after go-back
	out, _ = runCli("--raw", "eval", "document.title") // wait, standard evaluate, we can just check snapshot
	out, _ = runCli("--raw", "snapshot")
	if !strings.Contains(out, "Page 1") {
		t.Errorf("Not on page 1 after go-back: %s", out)
	}

	// Case 4: Normal go-forward
	out, err = runCli("--raw", "go-forward")
	if err != nil || !strings.Contains(out, "Go forward successfully") {
		t.Errorf("Normal go-forward failed: err=%v out=%s", err, out)
	}

	// Case 5: Verify state after go-forward
	out, _ = runCli("--raw", "snapshot")
	if !strings.Contains(out, "Page 2") {
		t.Errorf("Not on page 2 after go-forward: %s", out)
	}
	runCli("close")
}

func TestSessionsCommand(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")

	// Case 1: Empty sessions
	out, err := runCli("--raw", "sessions")
	if err != nil || !strings.Contains(out, "No active sessions") {
		t.Errorf("Empty sessions failed: err=%v out=%s", err, out)
	}

	// Case 2: One session
	runCli("--raw", "goto", ts.URL)
	out, err = runCli("--raw", "sessions")
	if err != nil || !strings.Contains(out, "- default") {
		t.Errorf("One session failed: err=%v out=%s", err, out)
	}

	// Case 3: Multiple sessions
	runCli("--raw", "-s", "second", "goto", ts.URL)
	out, err = runCli("--raw", "sessions")
	if err != nil || !strings.Contains(out, "- second") || !strings.Contains(out, "- default") {
		t.Errorf("Multiple sessions failed: err=%v out=%s", err, out)
	}

	// Case 4: JSON Output
	out, err = runCli("--json", "sessions")
	if err != nil || !strings.Contains(out, `"sessions":`) || !strings.Contains(out, `"default"`) {
		t.Errorf("JSON output failed: err=%v out=%s", err, out)
	}

	// Case 5: Session cleanup
	runCli("--raw", "close")
	runCli("--raw", "-s", "second", "close")
	out, _ = runCli("--raw", "sessions")
	if !strings.Contains(out, "No active sessions") {
		t.Errorf("Cleanup failed, still active sessions: %s", out)
	}
}

func TestSnapshotAndPdf(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")

	// Case 1: Snapshot without navigation
	out, err := runCli("--raw", "snapshot")
	if err == nil {
		t.Errorf("Snapshot without goto should fail: %s", out)
	}

	// Case 2: PDF without navigation
	out, err = runCli("--raw", "pdf")
	if err == nil {
		t.Errorf("PDF without goto should fail: %s", out)
	}

	// Case 3: Valid Snapshot
	runCli("--raw", "goto", ts.URL)
	out, err = runCli("--raw", "snapshot")
	if err != nil || !strings.Contains(out, "Home") {
		t.Errorf("Valid snapshot failed: err=%v out=%s", err, out)
	}

	// Case 4: Valid PDF
	out, err = runCli("--raw", "pdf", "--name", "testpdf")
	if err != nil || !strings.Contains(out, "Save to") {
		t.Errorf("Valid PDF failed: err=%v out=%s", err, out)
	}

	// Case 5: PDF naming
	out, _ = runCli("--raw", "pdf", "--name", "custom_name")
	if !strings.Contains(out, "custom_name.pdf") {
		t.Errorf("PDF naming failed: %s", out)
	}
	runCli("close")
}
