package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// proxyFixtureServer is a loopback HTML fixture (zero network egress).
func proxyFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><title>proxy fixture</title></head><body><h1>proxy test</h1></body></html>`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestProxyTraffic verifies that with --cdp-proxy, the cdp-traffic command
// returns a non-empty log containing expected CDP messages from a navigation.
func TestProxyTraffic(t *testing.T) {
	runCli("close")
	defer runCli("close")

	srv := proxyFixtureServer(t)

	// Navigate with proxy enabled
	out, err := runCli("--cdp-proxy", "goto", srv.URL)
	if err != nil {
		t.Fatalf("goto failed: %v\nOutput: %s", err, out)
	}

	// Read the traffic log
	raw, err := runCli("cdp-traffic", "--json")
	if err != nil {
		t.Fatalf("cdp-traffic failed: %v\nOutput: %s", err, raw)
	}

	var msgs []struct {
		Direction string          `json:"direction"`
		Raw       json.RawMessage `json:"raw"`
	}
	if err := json.Unmarshal([]byte(raw), &msgs); err != nil {
		t.Fatalf("cdp-traffic output not valid JSON: %v\nRaw: %s", err, raw)
	}

	if len(msgs) == 0 {
		t.Fatal("expected non-empty CDP traffic log after navigation")
	}

	// At least one recv message (Chrome response) should be present
	foundRecv := false
	for _, msg := range msgs {
		if msg.Direction == "recv" {
			foundRecv = true
			break
		}
	}
	if !foundRecv {
		t.Error("expected at least one 'recv' message in traffic log after navigation")
	}

	// At least some messages should contain recognizable CDP content
	// (Page.*, Target.*, etc.)
	foundCDP := false
	for _, msg := range msgs {
		if strings.Contains(string(msg.Raw), "\"method\"") ||
			strings.Contains(string(msg.Raw), "\"result\"") {
			foundCDP = true
			break
		}
	}
	if !foundCDP {
		t.Error("expected CDP protocol messages (JSON-RPC) in traffic log")
	}
}

// TestProxyCdpTell verifies that with --cdp-proxy and --console-capture,
// the console.debug stack-getter tell is suppressed (Runtime normalization).
func TestProxyCdpTell(t *testing.T) {
	runCli("close")
	defer runCli("close")

	srv := proxyFixtureServer(t)

	// Navigate with proxy + console-capture enabled
	out, err := runCli("--cdp-proxy", "--console-capture", "goto", srv.URL)
	if err != nil {
		t.Fatalf("goto failed: %v\nOutput: %s", err, out)
	}

	// Run the cdpTell probe from the live page.
	// The probe creates an Error, defines a getter on `stack`, calls
	// console.debug, and checks whether the getter fired.
	cdpTellProbe := `
		(function() {
			var fired = false;
			var e = new Error("cdpTell probe");
			Object.defineProperty(e, "stack", {
				get: function() { fired = true; return "probe"; },
				configurable: true
			});
			try { console.debug(e); } catch(_) {}
			return fired ? "stack-getter-fired" : "no-signal";
		})()
	`
	result, err := runCli("eval", "String("+cdpTellProbe+")")
	if err != nil {
		// eval itself may fail if console-capture causes issues;
		// the important thing is the probe result
		t.Logf("eval returned error (may be expected): %v", err)
	}

	result = strings.TrimSpace(result)
	if result != "no-signal" {
		t.Errorf("cdpTell probe expected 'no-signal' with proxy normalization, got: %q", result)
	}
}
