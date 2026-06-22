# 19-01: State Context API

## Description
Created the `PluginAPI` object to expose `godoll` browser state securely to the scripting environment. Implemented `GetCookies` and `GetSnapshot` methods that interact with the underlying `rod.Page`. Updated `BindLifecycle` to inject `PluginAPI` into the `goja.Runtime` as the global `api` variable.

## Files Modified
- `internal/plugin/api.go`: Added `PluginAPI` and implementation methods for cookies and snapshot.
- `internal/plugin/lifecycle.go`: Updated `BindLifecycle` to call `e.vm.Set("api", NewPluginAPI(page))`.

## Decisions Made
- Used `goja`'s native object binding functionality (`e.vm.Set`) to seamlessly expose the struct methods to JavaScript without needing wrapper proxy functions.
