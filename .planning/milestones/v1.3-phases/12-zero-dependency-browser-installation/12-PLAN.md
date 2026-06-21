# Phase 12: Zero-Dependency Browser Installation - Plan

## Target
Implement `rod-cli install` command (BROWSER-01).

## Implementation Steps

1. **Update `cmd.go`**:
   - Import `github.com/go-rod/rod/lib/launcher`.
   - Add a new command `install` to the `urfave/cli/v2` app commands array.
   - The action should print a status message, call `launcher.NewBrowser().MustGet()`, and print the installed path.
2. **Update `types/context.go`**:
   - Update `launchBrowser` to no longer fail hard if `launcher.LookPath()` fails. Wait, `types/context.go` currently returns an error if Chromium is not found. Should we auto-download on launch, or just guide the user to `rod-cli install`?
   - The requirement specifically says "Add `install` command to auto-download Chromium", achieving parity with Playwright. Playwright requires `playwright install` and fails if it's missing, telling the user to run the command.
   - We will modify `types/context.go`'s error message to instruct the user: `"the machine does not have Chrome installed. Please run 'rod-cli install' to fetch it."`

## Rollback Plan
- Revert `cmd.go` changes.
- Revert error message in `types/context.go`.
