package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
)

// --- Positive Tests ---

func TestNewPluginEngine(t *testing.T) {
	engine := NewPluginEngine()
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
	if engine.vm != nil {
		t.Fatal("expected vm to be nil before Init()")
	}
}

func TestPluginEngine_Init(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	if engine.vm == nil {
		t.Fatal("engine vm not initialized after Init()")
	}
}

func TestPluginEngine_LoadScript(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	tmpDir := t.TempDir()
	jsFile := filepath.Join(tmpDir, "test.js")
	jsCode := `var x = 10; var y = 20; var result = x + y;`
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

func TestPluginEngine_LoadScript_WithFunctions(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	tmpDir := t.TempDir()
	jsFile := filepath.Join(tmpDir, "func.js")
	jsCode := `function add(a, b) { return a + b; } var result = add(3, 7);`
	os.WriteFile(jsFile, []byte(jsCode), 0644)

	if err := engine.LoadScript(jsFile); err != nil {
		t.Fatalf("LoadScript with functions failed: %v", err)
	}

	val := engine.vm.Get("result")
	if val.ToInteger() != 10 {
		t.Fatalf("expected 10, got %v", val.ToInteger())
	}
}

func TestPluginEngine_LoadScript_MultipleScripts(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	tmpDir := t.TempDir()

	// First script sets a variable
	js1 := filepath.Join(tmpDir, "first.js")
	os.WriteFile(js1, []byte(`var counter = 1;`), 0644)

	// Second script reads and modifies it
	js2 := filepath.Join(tmpDir, "second.js")
	os.WriteFile(js2, []byte(`counter = counter + 1;`), 0644)

	if err := engine.LoadScript(js1); err != nil {
		t.Fatalf("LoadScript first failed: %v", err)
	}
	if err := engine.LoadScript(js2); err != nil {
		t.Fatalf("LoadScript second failed: %v", err)
	}

	val := engine.vm.Get("counter")
	if val.ToInteger() != 2 {
		t.Fatalf("expected counter=2, got %v", val.ToInteger())
	}
}

func TestPluginEngine_InvokeJSFunc(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	tmpDir := t.TempDir()
	jsFile := filepath.Join(tmpDir, "hooks.js")
	jsCode := `
		var lastEvent = null;
		function onRequest(ev) { lastEvent = ev; }
	`
	os.WriteFile(jsFile, []byte(jsCode), 0644)
	engine.LoadScript(jsFile)

	// Invoke the function
	engine.invokeJSFunc("onRequest", map[string]string{"url": "http://test.com"})

	val := engine.vm.Get("lastEvent")
	if val == nil {
		t.Fatal("expected lastEvent to be set")
	}
}

func TestPluginEngine_InvokeJSFunc_NonExistent(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	// Should not panic when function doesn't exist
	engine.invokeJSFunc("doesNotExist", "test")
}

// --- Negative Tests ---

func TestPluginEngine_LoadScript_NotInitialized(t *testing.T) {
	engine := NewPluginEngine()

	err := engine.LoadScript("/some/path.js")
	if err == nil {
		t.Fatal("expected error when loading script without Init()")
	}
	if err.Error() != "plugin engine not initialized" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestPluginEngine_LoadScript_FileNotFound(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	err := engine.LoadScript("/nonexistent/path/script.js")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestPluginEngine_LoadScript_SyntaxError(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	tmpDir := t.TempDir()
	jsFile := filepath.Join(tmpDir, "bad.js")
	os.WriteFile(jsFile, []byte(`function( { broken syntax`), 0644)

	err := engine.LoadScript(jsFile)
	if err == nil {
		t.Fatal("expected error for JS syntax error")
	}
}

func TestPluginEngine_InvokeJSFunc_NilVM(t *testing.T) {
	engine := NewPluginEngine()
	// vm is nil, should not panic
	engine.invokeJSFunc("anyFunc", "data")
}

func TestPluginEngine_InvokeJSFunc_NonFunction(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	// Set a non-function value
	engine.vm.Set("notAFunc", "just a string")

	// Should not panic when the value is not a function
	engine.invokeJSFunc("notAFunc", "data")
}

// --- Edge Case Tests ---

func TestPluginEngine_LoadScript_EmptyFile(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	tmpDir := t.TempDir()
	jsFile := filepath.Join(tmpDir, "empty.js")
	os.WriteFile(jsFile, []byte(""), 0644)

	err := engine.LoadScript(jsFile)
	if err != nil {
		t.Fatalf("expected no error for empty file, got: %v", err)
	}
}

func TestPluginEngine_LoadScript_UnicodeContent(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	tmpDir := t.TempDir()
	jsFile := filepath.Join(tmpDir, "unicode.js")
	os.WriteFile(jsFile, []byte(`var msg = "Привіт 🌍";`), 0644)

	err := engine.LoadScript(jsFile)
	if err != nil {
		t.Fatalf("expected no error for unicode file, got: %v", err)
	}

	val := engine.vm.Get("msg")
	if val.String() != "Привіт 🌍" {
		t.Fatalf("expected unicode string, got %s", val.String())
	}
}

func TestPluginEngine_LoadScript_LargeScript(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	tmpDir := t.TempDir()
	jsFile := filepath.Join(tmpDir, "large.js")

	// Generate a large script with many variable declarations
	var code string
	for i := 0; i < 1000; i++ {
		code += "var v" + itoa(i) + " = " + itoa(i) + ";\n"
	}
	code += "var total = v0 + v999;"
	os.WriteFile(jsFile, []byte(code), 0644)

	err := engine.LoadScript(jsFile)
	if err != nil {
		t.Fatalf("expected no error for large script, got: %v", err)
	}

	val := engine.vm.Get("total")
	if val.ToInteger() != 999 {
		t.Fatalf("expected 999, got %v", val.ToInteger())
	}
}

func TestPluginEngine_Init_DoubleInit(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	// Set a value
	engine.vm.Set("testVar", 42)

	// Re-init should create a fresh VM
	engine.Init()

	val := engine.vm.Get("testVar")
	if val != nil && !goja.IsUndefined(val) {
		t.Fatal("expected testVar to be undefined after re-init")
	}
}

// --- API Tests ---

func TestNewPluginAPI_NilPage(t *testing.T) {
	api := NewPluginAPI(nil)
	if api == nil {
		t.Fatal("expected non-nil API even with nil page")
	}

	cookies, err := api.GetCookies()
	if err != nil {
		t.Fatalf("expected nil error for nil page, got: %v", err)
	}
	if cookies != nil {
		t.Fatal("expected nil cookies for nil page")
	}

	snapshot, err := api.GetSnapshot()
	if err != nil {
		t.Fatalf("expected nil error for nil page, got: %v", err)
	}
	if snapshot != "" {
		t.Fatal("expected empty snapshot for nil page")
	}
}

// Helper: simple int to string without strconv import
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
