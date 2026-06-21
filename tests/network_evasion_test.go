package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNetworkEvasionHeaders(t *testing.T) {
	// Start a test server that echoes headers as JSON
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(r.Header)
	}))
	defer ts.Close()

	// Ensure clean daemon state before test
	runCli("close")

	// Navigate to the test server
	out, err := runCli("goto", ts.URL)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Dump the DOM which contains the JSON-encoded headers
	out, err = runCli("eval", "document.body.innerText")
	if err != nil {
		t.Fatalf("Failed to eval body: %v", err)
	}

	// Check for injected headers
	if !strings.Contains(out, "User-Agent") {
		t.Errorf("Expected User-Agent in headers, got: %s", out)
	}
	if !strings.Contains(out, "Accept-Language") {
		t.Errorf("Expected Accept-Language in headers, got: %s", out)
	}
	if !strings.Contains(out, "Sec-Ch-Ua") {
		t.Errorf("Expected Sec-Ch-Ua client hints in headers, got: %s", out)
	}
	
	runCli("close")
}
