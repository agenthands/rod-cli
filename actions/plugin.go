package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/agenthands/rod-cli/types"
)

// seam for testing — overridden in *_test.go to exercise the otherwise
// unreachable json.Marshal error branch. Defaults to encoding/json.Marshal.
var jsonMarshal = json.Marshal

// PluginLoad registers a plugin into the active session's daemon.
func PluginLoad(ctx *types.Context, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("plugin path is required")
	}

	engine := ctx.GetPluginEngine()
	if err := engine.LoadScript(path); err != nil {
		return "", err
	}

	page, err := ctx.ControlledPage()
	if err == nil && page != nil {
		engine.BindLifecycle(context.Background(), page)
	}

	ctx.AddLoadedPlugin(path)
	return fmt.Sprintf("Plugin loaded successfully from %s", path), nil
}

// PluginList displays all active plugins running in the daemon.
func PluginList(ctx *types.Context) (string, error) {
	plugins := ctx.GetLoadedPlugins()
	if len(plugins) == 0 {
		return "No active plugins loaded.", nil
	}

	data, err := jsonMarshal(plugins)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// PluginRun invokes the named JS function in the loaded plugin VM and returns
// its result. It delegates lookup, calling, and stringification to RunFunc.
func PluginRun(ctx *types.Context, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("plugin function name is required")
	}

	return ctx.GetPluginEngine().RunFunc(name)
}
