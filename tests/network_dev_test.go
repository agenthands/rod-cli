package tests

import (
	"strings"
	"testing"
)

func TestNetworkAndDevCommands(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close")

	// 1. Setup Route Mocking
	// We'll mock /api/data to return something else
	runCli("goto", ts.URL)
	
	out, err := runCli("--raw", "route", "--body", "Mocked API Data!", "*api/data*")
	if err != nil || !strings.Contains(out, "Added route") {
		t.Errorf("Route failed: err=%v out=%s", err, out)
	}

	// 2. Fetch the mocked route
	runCli("--raw", "goto", ts.URL+"/api/data")
	out, _ = runCli("--raw", "eval", "() => document.documentElement.outerHTML")
	if !strings.Contains(out, "Mocked API Data!") {
		t.Errorf("Route mocking failed, got: '%s'", out)
	}

	// 3. Route List
	out, _ = runCli("--raw", "route-list")
	t.Logf("Route list output: %s", out)
	if !strings.Contains(out, "*api/data*") {
		t.Errorf("Route list missing route: %s", out)
	}

	// 4. Unroute
	runCli("--raw", "unroute", "*api/data*")
	runCli("--raw", "goto", ts.URL+"/api/data")
	out, _ = runCli("--raw", "eval", "() => document.documentElement.outerHTML")
	if strings.Contains(out, "Mocked API Data!") {
		t.Errorf("Unroute failed, still mocked")
	}

	// 5. Console Logs
	runCli("--raw", "eval", "() => console.log('Testing the console logs!')")
	out, _ = runCli("--raw", "console")
	if !strings.Contains(out, "Testing the console logs!") {
		t.Errorf("Console logs did not capture message: %s", out)
	}

	// 6. Network Requests
	// Should have captured /api/data requests
	out, _ = runCli("--raw", "requests")
	if !strings.Contains(out, "GET") || !strings.Contains(out, "/api/data") {
		t.Errorf("Network requests not captured: %s", out)
	}

	// Extract index from requests list
	reqIndex := "0"
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "/api/data") {
			start := strings.Index(line, "[")
			end := strings.Index(line, "]")
			if start != -1 && end != -1 {
				reqIndex = line[start+1 : end]
				break
			}
		}
	}

	// 7. Request Details
	out, _ = runCli("--raw", "request", reqIndex)
	if !strings.Contains(out, "/api/data") {
		t.Errorf("Request detail failed: %s", out)
	}

	runCli("close")
}
