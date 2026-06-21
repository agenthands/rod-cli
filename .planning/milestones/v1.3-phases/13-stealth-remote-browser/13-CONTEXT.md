# Phase 13: Stealth and Remote Browser Integration - Context

**Gathered:** 2026-06-21
**Status:** Ready for planning
**Mode:** Auto-generated (Autonomous mode)

<domain>
## Phase Boundary

Replace inline launcher with `godoll` wrapped launcher.
</domain>

<decisions>
## Implementation Decisions

### Agent's Discretion
Use `godoll.NewBrowser(opts)` and `godoll/browser.ConnectToRemoteBrowser()`.
</decisions>

<code_context>
## Existing Code Insights

- `types/context.go`: The `launchBrowser` and `controlBrowser` methods must be refactored to use `godoll/browser`.
</code_context>

<specifics>
## Specific Ideas

- Refactor `launchBrowser` to use `browser.NewBrowser(opts)` where `opts` is instantiated with `browser.NewBrowserOptions().StealthPreset()`.
- Refactor `controlBrowser` to use `browser.ConnectToRemoteBrowser()` when attaching to remote sessions.
</specifics>

<deferred>
## Deferred Ideas

None.
</deferred>
