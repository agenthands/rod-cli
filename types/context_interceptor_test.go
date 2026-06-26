package types

import (
	"context"
	"testing"

	"github.com/agenthands/godoll/network"
	rodfingerprint "github.com/agenthands/godoll/fingerprint"
)

func TestContextInterceptorRules(t *testing.T) {
	ctx := NewContext(context.Background(), Config{})
	ctx.interceptor = network.NewInterceptor(nil)
	
	// Set mock routes
	ctx.routes = map[string]string{
		"**/*.png": "mocked png",
	}

	// Generate a dummy fingerprint
	fg := rodfingerprint.NewFingerprintGenerator(rodfingerprint.FPWithBrowserNames("chrome"))
	fp, err := fg.Generate()
	if err != nil {
		t.Fatalf("failed to generate fingerprint: %v", err)
	}
	ctx.fingerprint = fp

	// Update interceptor rules
	ctx.updateInterceptorRules()

	// Interceptor should now have the rules
	if ctx.interceptor == nil {
		t.Fatal("interceptor should not be nil")
	}

	// Phase 30 (CDP-01): the always-on identity catch-all rule was removed —
	// header coherence now rides on Emulation.setUserAgentOverride, so the
	// interceptor carries ONLY mock-route rules. One route ⇒ exactly one rule.
	rules := ctx.interceptor.Rules()
	if len(rules) != 1 {
		t.Fatalf("expected exactly 1 mock rule (no catch-all), got %d", len(rules))
	}

	foundMock := false
	for _, r := range rules {
		if r.URLPattern == "**/*.png" {
			foundMock = true
			if r.MockResponse == nil || r.MockResponse.Body != "mocked png" {
				t.Errorf("mock response body mismatch: %v", r.MockResponse)
			}
		}
		if r.URLPattern == "*" {
			t.Errorf("unexpected catch-all rule: identity moved to Emulation override")
		}
	}

	if !foundMock {
		t.Errorf("expected mock route rule for **/*.png")
	}
}
