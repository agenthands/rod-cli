package plugin

import (
	"strings"
	"testing"
)

// LoadScript: os.Open succeeds on a directory, but io.ReadAll on a directory
// file descriptor fails ("is a directory"), exercising the read-error branch.
func TestPluginEngine_LoadScript_DirReadError(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	dir := t.TempDir()
	err := engine.LoadScript(dir)
	if err == nil {
		t.Fatal("expected error when loading a directory as a script")
	}
	if !strings.Contains(err.Error(), "failed to read script file") {
		t.Fatalf("expected read-error wrap, got: %v", err)
	}
}

// RunFunc: the called JS function throws a runtime error -> wrapped error.
func TestPluginEngine_RunFunc_RuntimeError(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	if _, err := engine.vm.RunString(`function boom(){ throw new Error("kaboom"); }`); err != nil {
		t.Fatalf("failed to define boom: %v", err)
	}

	out, err := engine.RunFunc("boom")
	if err == nil {
		t.Fatal("expected error from a throwing JS function")
	}
	if out != "" {
		t.Fatalf("expected empty result on error, got %q", out)
	}
}

// RunFunc: a function returning undefined yields ("", nil).
func TestPluginEngine_RunFunc_ReturnsUndefined(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	if _, err := engine.vm.RunString(`function noop(){ /* returns undefined */ }`); err != nil {
		t.Fatalf("failed to define noop: %v", err)
	}

	out, err := engine.RunFunc("noop")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if out != "" {
		t.Fatalf("expected empty string for undefined return, got %q", out)
	}
}

// RunFunc: a function returning null yields ("", nil).
func TestPluginEngine_RunFunc_ReturnsNull(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	if _, err := engine.vm.RunString(`function getNull(){ return null; }`); err != nil {
		t.Fatalf("failed to define getNull: %v", err)
	}

	out, err := engine.RunFunc("getNull")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if out != "" {
		t.Fatalf("expected empty string for null return, got %q", out)
	}
}

// RunFunc: a non-empty scalar return is stringified.
func TestPluginEngine_RunFunc_ReturnsString(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	if _, err := engine.vm.RunString(`function greet(){ return "hi"; }`); err != nil {
		t.Fatalf("failed to define greet: %v", err)
	}

	out, err := engine.RunFunc("greet")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if out != "hi" {
		t.Fatalf("expected 'hi', got %q", out)
	}
}
