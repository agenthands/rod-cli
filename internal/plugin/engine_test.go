package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPluginEngine(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	if engine.vm == nil {
		t.Fatal("engine vm not initialized")
	}

	// Create a temporary JS file
	tmpDir := t.TempDir()
	jsFile := filepath.Join(tmpDir, "test.js")
	jsCode := `
		var x = 10;
		var y = 20;
		var result = x + y;
	`
	if err := os.WriteFile(jsFile, []byte(jsCode), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	err := engine.LoadScript(jsFile)
	if err != nil {
		t.Fatalf("LoadScript failed: %v", err)
	}

	val := engine.vm.Get("result")
	if val == nil {
		t.Fatal("expected result variable")
	}

	if val.ToInteger() != 30 {
		t.Fatalf("expected result to be 30, got %v", val.ToInteger())
	}
}
