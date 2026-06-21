# Phase 12: Zero-Dependency Browser Installation - Context

**Gathered:** 2026-06-21
**Status:** Ready for planning
**Mode:** Auto-generated (Autonomous mode)

<domain>
## Phase Boundary

Add `install` command to auto-download Chromium.
</domain>

<decisions>
## Implementation Decisions

### Agent's Discretion
Use `go-rod`'s `launcher.NewBrowser().MustGet()` to automatically fetch the browser binary. We should add an `install` subcommand to `urfave/cli/v2` in `cmd.go`.
</decisions>

<code_context>
## Existing Code Insights

- `cmd.go`: Houses the `urfave/cli/v2` app commands. We will add a new `Command` named `"install"`.
- Since we don't need the daemon for installing, this command will run directly on the client side without contacting the daemon.
</code_context>

<specifics>
## Specific Ideas

Add an `install` command:
```go
{
	Name:  "install",
	Usage: "Install the Chromium browser required by rod-cli",
	Action: func(c *cli.Context) error {
		fmt.Println("Downloading Chromium...")
		browserPath := launcher.NewBrowser().MustGet()
		fmt.Printf("Chromium installed successfully at: %s\n", browserPath)
		return nil
	},
}
```
</specifics>

<deferred>
## Deferred Ideas

None.
</deferred>
