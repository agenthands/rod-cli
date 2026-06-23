package actions

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agenthands/rod-cli/types"
	"github.com/go-rod/rod/lib/launcher"
)

// writeFile is a small helper shared by the actions tests.
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func TestPluginLoad(t *testing.T) {
	ctx := types.NewContext(context.Background(), types.Config{})

	// negative: empty path
	if _, err := PluginLoad(ctx, ""); err == nil {
		t.Error("PluginLoad expected error for empty path")
	}

	// negative: nonexistent file
	if _, err := PluginLoad(ctx, "/no/such/plugin.js"); err == nil {
		t.Error("PluginLoad expected error for missing file")
	}

	// positive: a valid plugin script (no page bound)
	dir := t.TempDir()
	pluginPath := filepath.Join(dir, "plugin.js")
	if err := writeFile(pluginPath, "function getFindings(){return JSON.stringify([{ok:true}])}"); err != nil {
		t.Fatal(err)
	}
	out, err := PluginLoad(ctx, pluginPath)
	if err != nil {
		t.Fatalf("PluginLoad: %v", err)
	}
	if !strings.Contains(out, "loaded successfully") {
		t.Errorf("unexpected: %s", out)
	}

	// negative: syntax error script
	badPath := filepath.Join(dir, "bad.js")
	if err := writeFile(badPath, "function ("); err != nil {
		t.Fatal(err)
	}
	if _, err := PluginLoad(ctx, badPath); err == nil {
		t.Error("PluginLoad expected error for syntax error script")
	}
}

func TestPluginLoad_WithPage(t *testing.T) {
	// Exercises the BindLifecycle branch where a controlled page exists.
	u := launcher.New().Headless(true).MustLaunch()
	ctx := types.NewContext(context.Background(), types.Config{CDPEndpoint: u})
	if _, err := ctx.EnsurePage(); err != nil {
		t.Fatalf("EnsurePage: %v", err)
	}
	defer ctx.CloseBrowser()

	dir := t.TempDir()
	pluginPath := filepath.Join(dir, "plugin.js")
	if err := writeFile(pluginPath, "function onLoad(){} function getFindings(){return '[]'}"); err != nil {
		t.Fatal(err)
	}
	if _, err := PluginLoad(ctx, pluginPath); err != nil {
		t.Fatalf("PluginLoad with page: %v", err)
	}
}

func TestPluginList(t *testing.T) {
	ctx := types.NewContext(context.Background(), types.Config{})

	// empty
	if out, err := PluginList(ctx); err != nil {
		t.Fatalf("PluginList: %v", err)
	} else if !strings.Contains(out, "No active plugins") {
		t.Errorf("unexpected: %s", out)
	}

	// after loading one
	dir := t.TempDir()
	pluginPath := filepath.Join(dir, "p.js")
	if err := writeFile(pluginPath, "function f(){return '1'}"); err != nil {
		t.Fatal(err)
	}
	if _, err := PluginLoad(ctx, pluginPath); err != nil {
		t.Fatal(err)
	}
	if out, err := PluginList(ctx); err != nil {
		t.Fatalf("PluginList: %v", err)
	} else if !strings.Contains(out, "p.js") {
		t.Errorf("expected plugin path in list, got: %s", out)
	}
}

func TestPluginRun(t *testing.T) {
	ctx := types.NewContext(context.Background(), types.Config{})

	// negative: empty name
	if _, err := PluginRun(ctx, ""); err == nil {
		t.Error("PluginRun expected error for empty name")
	}

	dir := t.TempDir()
	pluginPath := filepath.Join(dir, "p.js")
	if err := writeFile(pluginPath, "function getFindings(){return JSON.stringify([{a:1}])}"); err != nil {
		t.Fatal(err)
	}
	if _, err := PluginLoad(ctx, pluginPath); err != nil {
		t.Fatal(err)
	}

	// positive
	out, err := PluginRun(ctx, "getFindings")
	if err != nil {
		t.Fatalf("PluginRun: %v", err)
	}
	if !strings.Contains(out, "a") {
		t.Errorf("unexpected: %s", out)
	}

	// negative: missing function
	if _, err := PluginRun(ctx, "doesNotExist"); err == nil {
		t.Error("PluginRun expected error for missing func")
	}
}
