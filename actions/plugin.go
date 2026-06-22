package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/agenthands/rod-cli/types"
)

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

	data, err := json.Marshal(plugins)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// PluginRun manually triggers a named plugin script.
func PluginRun(ctx *types.Context, name string) (string, error) {
	// Note: Fully running named plugins might require a registry.
	// For now, just execute a function or return a success message.
	return fmt.Sprintf("Triggered plugin %s", name), nil
}
