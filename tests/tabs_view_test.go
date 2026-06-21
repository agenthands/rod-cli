package tests

import (
	"strings"
	"testing"
)

func TestTabsAndViewCommands(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")

	out, err := runCli("--raw", "goto", ts.URL+"/page1")
	if err != nil {
		t.Fatalf("goto failed: %v, out: %s", err, out)
	}

	// 1. Resize
	out, err = runCli("--raw", "resize", "800", "600")
	if err != nil || !strings.Contains(out, "Viewport resized") {
		t.Errorf("Resize failed: err=%v out=%s", err, out)
	}

	// 2. Mousewheel
	out, err = runCli("--raw", "mousewheel", "0", "100")
	if err != nil || !strings.Contains(out, "Mouse wheel scrolled") {
		t.Errorf("Mousewheel failed: err=%v out=%s", err, out)
	}

	// 3. Tab List
	out, err = runCli("--raw", "tab-list")
	if err != nil || !strings.Contains(out, "[0]") || !strings.Contains(out, "page1") {
		t.Errorf("Tab list failed: err=%v out=%s", err, out)
	}

	// 4. Tab New
	out, err = runCli("--raw", "tab-new", ts.URL+"/page2")
	if err != nil || !strings.Contains(out, "New tab created") {
		t.Errorf("Tab new failed: err=%v out=%s", err, out)
	}

	// Wait, tab-new sets the active page to the new tab!
	// So evaluating document.title should return "Page 2" (well, our server sets title to empty, let's check id="p2")
	out, _ = runCli("--raw", "eval", "document.getElementById('p2') !== null")
	if !strings.Contains(out, "true") {
		t.Errorf("New tab did not become active or load page2 correctly")
	}

	// List tabs again
	out, _ = runCli("--raw", "tab-list")
	t.Logf("Tab list before select: %s", out)
	if !strings.Contains(out, "[1]") {
		t.Errorf("Tab 1 not found in list: %s", out)
	}

	// Find the index of page1
	page1Index := "0"
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "page1") {
			// Extract the number in brackets
			start := strings.Index(line, "[")
			end := strings.Index(line, "]")
			if start != -1 && end != -1 {
				page1Index = line[start+1 : end]
			}
		}
	}

	// 5. Tab Select
	out, err = runCli("--raw", "tab-select", page1Index)
	if err != nil || !strings.Contains(out, "Selected tab "+page1Index) {
		t.Errorf("Tab select failed: err=%v out=%s", err, out)
	}

	// Verify we are back on page 1
	out, _ = runCli("--raw", "eval", "document.getElementById('p1') !== null")
	if !strings.Contains(out, "true") {
		t.Errorf("Tab select did not activate tab %s correctly, out: %s", page1Index, out)
	}

	// 6. Tab Close
	out, err = runCli("--raw", "tab-close", page1Index)
	if err != nil || !strings.Contains(out, "Closed tab "+page1Index) {
		t.Errorf("Tab close failed: err=%v out=%s", err, out)
	}

	// Verify tab 1 is gone
	out, _ = runCli("--raw", "tab-list")
	if strings.Contains(out, "[1]") {
		t.Errorf("Tab 1 was not removed from list: %s", out)
	}

	runCli("close")
}
