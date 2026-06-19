package tests

import (
	"regexp"
	"strings"
	"testing"
)

// Helper to find a snapshot ref ID for a given label
func extractRef(snapshotOut string, label string) string {
	// Looks for: - button "Submit" [ref=s1e8]
	re := regexp.MustCompile(regexp.QuoteMeta(label) + `"?\s+\[ref=([^\]]+)\]`)
	matches := re.FindStringSubmatch(snapshotOut)
	if len(matches) >= 2 {
		return matches[1]
	}
	
	// Try alternative format if order is different: - textbox [ref=s2e3] "UserField"
	re2 := regexp.MustCompile(`\[ref=([^\]]+)\][^<]*` + regexp.QuoteMeta(label))
	matches2 := re2.FindStringSubmatch(snapshotOut)
	if len(matches2) >= 2 {
		return matches2[1]
	}
	return ""
}

func TestClickAndHover(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")

	runCli("--raw", "goto", ts.URL+"/forms")
	out, _ := runCli("--raw", "snapshot")
	
	submitRef := extractRef(out, "Submit")
	if submitRef == "" {
		t.Fatalf("Could not find Submit button in snapshot: %s", out)
	}

	// Case 1: Hover
	out, err := runCli("--raw", "hover", submitRef)
	if err != nil || !strings.Contains(out, "Hovered") {
		t.Errorf("Hover failed: err=%v out=%s", err, out)
	}

	// Case 2: Click
	out, err = runCli("--raw", "click", submitRef)
	if err != nil || !strings.Contains(out, "Click element") {
		t.Errorf("Click failed: err=%v out=%s", err, out)
	}

	// Verify click worked via Eval
	out, _ = runCli("--raw", "eval", "document.getElementById('output').innerText")
	if !strings.Contains(out, "Submitted") {
		t.Errorf("Click did not trigger event, output: %s", out)
	}

	// Case 3: Double Click
	// We can't easily verify the event without more JS, but we can verify the command succeeds
	out, err = runCli("--raw", "dblclick", submitRef)
	if err != nil || !strings.Contains(out, "Double clicked element") {
		t.Errorf("Double click failed: err=%v out=%s", err, out)
	}

	// Case 4: Click missing ref
	out, err = runCli("--raw", "click", "99999")
	if err == nil {
		t.Errorf("Click missing ref should fail: %s", out)
	}

	// Case 5: Click without ref
	out, err = runCli("--raw", "click")
	if err == nil {
		t.Errorf("Click without ref should fail: %s", out)
	}

	runCli("close")
}

func TestInputsAndForms(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")

	runCli("--raw", "goto", ts.URL+"/forms")
	out, _ := runCli("--raw", "snapshot")

	// We need to identify the inputs. Our server renders:
	// <input type="text" id="username" />
	// Let's modify the server to add placeholders or labels so they show in snapshot, 
	// OR we can just inject IDs in the snapshot parser? 
	// Actually, inputs without labels show up with their node name.
	
	// Let's use eval to set their values and test Type / Fill
	// Wait, we need their refs!
	// Let's use eval to add aria-labels so snapshot picks them up cleanly.
	runCli("--raw", "eval", "document.getElementById('username').setAttribute('aria-label', 'UserField')")
	runCli("--raw", "eval", "document.getElementById('password').setAttribute('aria-label', 'PassField')")
	runCli("--raw", "eval", "document.getElementById('dropdown').setAttribute('aria-label', 'DropField')")
	out, _ = runCli("--raw", "snapshot") // Re-take snapshot

	userRef := extractRef(out, "UserField")
	passRef := extractRef(out, "PassField")
	dropRef := extractRef(out, "DropField")

	if userRef == "" || dropRef == "" {
		t.Fatalf("Could not find inputs in snapshot: %s", out)
	}

	// Case 1: Type
	out, err := runCli("--raw", "type", userRef, "alice")
	if err != nil || !strings.Contains(out, "Typed text") {
		t.Errorf("Type failed: err=%v out=%s", err, out)
	}

	// Case 2: Fill (clears before typing)
	out, err = runCli("--raw", "fill", passRef, "secret")
	if err != nil || !strings.Contains(out, "Fill out") {
		t.Errorf("Fill failed: err=%v out=%s", err, out)
	}

	// Case 3: Select
	out, err = runCli("--raw", "select", dropRef, "Two")
	if err != nil || !strings.Contains(out, "Select option") {
		t.Errorf("Select failed: err=%v out=%s", err, out)
	}

	// Case 4: Verify inputs
	out, _ = runCli("--raw", "eval", "document.getElementById('username').value + ' ' + document.getElementById('dropdown').value")
	if !strings.Contains(out, "alice 2") {
		t.Errorf("Inputs were not set correctly: %s", out)
	}

	// Case 5: Fill with submit
	// Wait, does 'fill' support --submit? Yes, fill [ref] [text] [submit]
	out, err = runCli("--raw", "fill", userRef, "bob", "true")
	// If it submits, it triggers the form. Wait, we don't have a form, just a button.
	// `fill` with submit triggers Enter key.
	// We can just verify it doesn't crash.
	if err != nil {
		t.Errorf("Fill with submit failed: err=%v out=%s", err, out)
	}

	runCli("close")
}

func TestDialogs(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")

	runCli("--raw", "goto", ts.URL+"/dialogs")
	out, _ := runCli("--raw", "snapshot")
	
	alertRef := extractRef(out, "Alert")
	confirmRef := extractRef(out, "Confirm")

	// Case 1: Setup async dialog acceptor before clicking alert
	// Wait, dialog-accept is synchronous in rod-cli?
	// actions.HandleDialog starts a goroutine to wait for the dialog.
	out, err := runCli("--raw", "dialog-accept")
	if err != nil {
		t.Errorf("dialog-accept failed: %v", err)
	}

	runCli("--raw", "click", alertRef)
	// The dialog should be automatically accepted.

	// Case 2: dialog-dismiss
	runCli("--raw", "dialog-dismiss")
	runCli("--raw", "click", confirmRef)
	out, _ = runCli("--raw", "eval", "window.res")
	if !strings.Contains(out, "false") {
		t.Errorf("Expected window.res to be false after dismiss, got: %s", out)
	}

	// Case 3: dialog-accept with prompt text
	// Our server doesn't have a prompt, but we can just pass the arg
	runCli("--raw", "dialog-accept", "mytext")
	
	// Case 4: Multiple handlers (should overwrite or queue)
	runCli("--raw", "dialog-accept")
	runCli("--raw", "dialog-dismiss")

	runCli("close")
}

func TestRawInputs(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")

	runCli("--raw", "goto", ts.URL)

	// Case 1: Press Enter
	out, err := runCli("--raw", "press", "Enter")
	if err != nil || !strings.Contains(out, "Pressed") {
		t.Errorf("Press failed: err=%v out=%s", err, out)
	}

	// Case 2: Press single char
	out, err = runCli("--raw", "press", "A")
	if err != nil || !strings.Contains(out, "Pressed key: A") {
		t.Errorf("Press single char failed: err=%v out=%s", err, out)
	}

	// Case 3: MouseMove
	out, err = runCli("--raw", "mousemove", "100", "200")
	if err != nil || !strings.Contains(out, "Moved mouse") {
		t.Errorf("MouseMove failed: err=%v out=%s", err, out)
	}

	// Case 4: MouseDown
	out, err = runCli("--raw", "mousedown", "left")
	if err != nil || !strings.Contains(out, "Mouse down") {
		t.Errorf("MouseDown failed: err=%v out=%s", err, out)
	}

	// Case 5: MouseUp
	out, err = runCli("--raw", "mouseup", "left")
	if err != nil || !strings.Contains(out, "Mouse up") {
		t.Errorf("MouseUp failed: err=%v out=%s", err, out)
	}

	runCli("close")
}
