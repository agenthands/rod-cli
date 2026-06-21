# Phase 14: Network and Humanized Interactions - Context

**Gathered:** 2026-06-21
**Status:** Ready for planning
**Mode:** Auto-generated (Autonomous mode)

<domain>
## Phase Boundary

Implement `godoll/network` interceptor and bezier scrolling.
</domain>

<decisions>
## Implementation Decisions

### Agent's Discretion
Use `godoll/network.NewInterceptor()` to handle `route` and `unroute` commands.
Use `humanize.Scroll()` for the `scroll` action.
</decisions>

<code_context>
## Existing Code Insights

- `actions/actions.go`: The `Scroll` action calls `page.Mouse.Scroll(0, 0, int(dy))`. This needs to be replaced with `humanize.Scroll(page, 0, float64(dy))`. Note: I should check the exact signature of `humanize.Scroll`.
- `daemon/daemon.go`: Routing and mock settings are handled. We might need to refactor how routes are stored in the session or applied to the page.
</code_context>

<specifics>
## Specific Ideas

- Check `godoll/humanize/scroll.go` for the `Scroll` signature.
- Check `godoll/network` for the interceptor signature.
</specifics>

<deferred>
## Deferred Ideas

None.
</deferred>
