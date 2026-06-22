# 18-01: Lifecycle Event Emitters

## Description
Exposed `godoll` browser events to the plugin engine by attaching CDP event listeners (`network.EventRequestWillBeSent`, `network.EventResponseReceived`, `page.EventLoadEventFired`, `dom.EventChildNodeInserted`) and forwarding them to corresponding global JavaScript functions (`onRequest`, `onResponse`, `onLoad`, `onDOMNodeInserted`) if defined in the plugin script.

## Files Modified
- `internal/plugin/lifecycle.go`: Created new file containing `LifecycleEmitter` interface and `BindLifecycle` method. Added `invokeJSFunc` helper to `PluginEngine` to safely call JS globals.

## Decisions Made
- Chose to bind directly to `go-rod` CDP types (like `proto.NetworkRequestWillBeSent`) and serialize them to JavaScript via `goja`'s `ToValue` mapper instead of creating intermediate translation structs, to provide plugins with full raw CDP payload access.
