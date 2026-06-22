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
