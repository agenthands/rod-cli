package tests

import (
	"os"
	"strings"
	"testing"
)

func TestFormsExtraCommands(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")

	out, err := runCli("--raw", "goto", ts.URL+"/forms")
	if err != nil {
		t.Fatalf("goto failed: %v, out: %s", err, out)
	}

	// Add labels to the new inputs so we can find them in the snapshot
	runCli("--raw", "eval", "document.getElementById('terms').setAttribute('aria-label', 'Terms')")
	runCli("--raw", "eval", "document.getElementById('gender').setAttribute('aria-label', 'Gender')")
	runCli("--raw", "eval", "document.getElementById('fileup').setAttribute('aria-label', 'UploadFile')")

	out, _ = runCli("--raw", "snapshot")

	termsRef := extractRef(out, "Terms")
	genderRef := extractRef(out, "Gender")
	uploadRef := extractRef(out, "UploadFile")

	if termsRef == "" || genderRef == "" || uploadRef == "" {
		t.Fatalf("Could not find new inputs in snapshot: %s", out)
	}

	// Case 1: Check
	out, err = runCli("--raw", "check", termsRef)
	if err != nil || !strings.Contains(out, "Checked element") {
		t.Errorf("Check failed: err=%v out=%s", err, out)
	}
	out, err = runCli("--raw", "check", genderRef)
	if err != nil || !strings.Contains(out, "Checked element") {
		t.Errorf("Check radio failed: err=%v out=%s", err, out)
	}

	// Verify checked via Eval
	out, _ = runCli("--raw", "eval", "document.getElementById('terms').checked && document.getElementById('gender').checked")
	if !strings.Contains(out, "true") {
		t.Errorf("Elements were not actually checked")
	}

	// Case 2: Uncheck
	out, err = runCli("--raw", "uncheck", termsRef)
	if err != nil || !strings.Contains(out, "Unchecked element") {
		t.Errorf("Uncheck failed: err=%v out=%s", err, out)
	}

	// Verify unchecked via Eval
	out, _ = runCli("--raw", "eval", "document.getElementById('terms').checked")
	if !strings.Contains(out, "false") {
		t.Errorf("Element was not actually unchecked")
	}

	// Case 3: Upload
	// Create a dummy file
	testFile := "test_upload.txt"
	os.WriteFile(testFile, []byte("hello world"), 0644)
	defer os.Remove(testFile)

	out, err = runCli("--raw", "upload", uploadRef, testFile)
	if err != nil || !strings.Contains(out, "Uploaded files") {
		t.Errorf("Upload failed: err=%v out=%s", err, out)
	}

	// Verify upload via Eval
	out, _ = runCli("--raw", "eval", "document.getElementById('fileup').files[0].name")
	if !strings.Contains(out, "test_upload.txt") {
		t.Errorf("File was not actually uploaded: %s", out)
	}

	// Case 4: Drop
	out, err = runCli("--raw", "drop", "--path", testFile, uploadRef)
	if err != nil || !strings.Contains(out, "Dropped file(s)") {
		t.Errorf("Drop failed: err=%v out=%s", err, out)
	}

	// Case 5: Drag
	// We'll just drag from termsRef to genderRef
	out, err = runCli("--raw", "drag", termsRef, genderRef)
	if err != nil || !strings.Contains(out, "Dragged from") {
		t.Errorf("Drag failed: err=%v out=%s", err, out)
	}

	runCli("close")
}
