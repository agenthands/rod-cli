# Phase 13: Stealth and Remote Browser Integration - Plan

## Target
Replace inline launcher with `godoll` wrapped launcher (BROWSER-02, BROWSER-03).

## Implementation Steps

1. **Update `types/context.go`**:
   - Refactor `launchBrowser` to create `browser.BrowserOptions`. Apply proxy, executable path, headless, and user data dir directly via `opts.Set()`.
   - Apply stealth presets: `opts.Set("prefs", opts.StealthPreset())`.
   - Launch with `godoll.NewBrowser(opts)` instead of `launcher.MustLaunch()`. Note: `godoll` exposes `godoll.NewBrowser(opts)` in the root package, or `browser.NewBrowser` inside `godoll/browser`. We'll use `godoll/browser`.
   - Refactor `controlBrowser` to use `browser.ConnectToRemoteBrowser()` when remote control url is provided. Since `controlBrowser` takes a standard websocket URL, it might just need `browser.ConnectToRemoteBrowserWithURL()` or directly using `godoll/browser`.

## Rollback Plan
- Revert changes to `types/context.go`.
