package actions

import (
	"context"
	"strings"
	"testing"

	"github.com/agenthands/rod-cli/types"
	"github.com/go-rod/rod/lib/launcher"
)

func TestNavigateWithRetries(t *testing.T) {
	// Start a local browser instance for testing
	u := launcher.New().MustLaunch()
	ctx := types.NewContext(context.Background(), types.Config{CDPEndpoint: u})
	
	// Create the page via standard types.Context wrapper
	_, err := ctx.EnsurePage()
	if err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}

	// Test navigation to a definitely non-existent local port
	// It should fail, but because of the retry wrapper it will take some time
	// and eventually return the error.
	_, err = Navigate(ctx, "http://127.0.0.1:59999")
	if err == nil {
		t.Fatal("Expected error when navigating to non-existent port")
	}
	
	if !strings.Contains(err.Error(), "Failed to navigate") {
		t.Errorf("Expected network error from go-rod, got: %v", err)
	}
	
	// Test element resolution that will fail (should retry and eventually return error)
	_, err = Click(ctx, "#non-existent-element")
	if err == nil {
		t.Fatal("Expected error when clicking non-existent element")
	}
}
