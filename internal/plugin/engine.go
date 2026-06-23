package plugin

import (
	"fmt"
	"io"
	"os"

	"github.com/dop251/goja"
)

// PluginEngine wraps a goja.Runtime to safely execute external JavaScript.
type PluginEngine struct {
	vm *goja.Runtime
}

// NewPluginEngine creates a new plugin engine instance.
func NewPluginEngine() *PluginEngine {
	return &PluginEngine{}
}

// Init instantiates the underlying goja.Runtime.
func (e *PluginEngine) Init() {
	e.vm = goja.New()
}

// LoadScript reads a file from the given path and executes it in the JS VM.
func (e *PluginEngine) LoadScript(path string) error {
	if e.vm == nil {
		return fmt.Errorf("plugin engine not initialized")
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open script file %s: %w", path, err)
	}
	defer file.Close()

	src, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read script file %s: %w", path, err)
	}

	_, err = e.vm.RunScript(path, string(src))
	if err != nil {
		return fmt.Errorf("failed to execute script %s: %w", path, err)
	}

	return nil
}

// RunFunc looks up a top-level JS function named `name` in the loaded plugin's
// VM, calls it with no arguments, and returns its result stringified.
//
// The lookup mirrors invokeJSFunc (vm.Get + goja.AssertFunction), but unlike the
// fire-and-forget hook dispatch, RunFunc surfaces errors for the CLI:
//   - a nil VM (engine never Init'd) returns "plugin engine not initialized"
//   - a missing name returns a "not found" error
//   - a non-callable name returns a "not a callable function" error
//   - a runtime error during the call is wrapped and returned
//
// A function returning undefined/null yields an empty string and a nil error.
// Otherwise the goja result is stringified via res.String(); the example
// accessors (getFindings/getRequestLog) already JSON.stringify their output, so
// the CLI sees clean JSON without any Go-side json.Marshal.
func (e *PluginEngine) RunFunc(name string) (string, error) {
	if e.vm == nil {
		return "", fmt.Errorf("plugin engine not initialized")
	}

	fnObj := e.vm.Get(name)
	if fnObj == nil {
		return "", fmt.Errorf("function %q not found", name)
	}

	call, ok := goja.AssertFunction(fnObj)
	if !ok {
		return "", fmt.Errorf("%q is not a callable function", name)
	}

	res, err := call(goja.Undefined())
	if err != nil {
		return "", fmt.Errorf("error calling %q: %w", name, err)
	}

	if res == nil || goja.IsUndefined(res) || goja.IsNull(res) {
		return "", nil
	}

	return res.String(), nil
}
