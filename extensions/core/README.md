# @agenthands/rod-cli-core

Runtime-agnostic core for rod-cli host extensions -- binary discovery, normalized exec contract, input validation, timeout table, and pure argv-builders for the `browse_*` tools.

**This is a shared library, not installed directly by end users.** The host adapter packages list it in `dependencies`.

## Public Surface

- **`findRodCli()`** -- Locate the `rod-cli` binary: `$ROD_CLI_PATH` first, then each `$PATH` directory, then `$GOBIN` (default `~/go/bin`). Returns the absolute path, or `null` when not found.
- **`ExecFn` / `ExecResult` / `ExecOptions`** -- Normalized exec contract: `ExecFn` is the host's exec function `(args, { signal?, timeout? }) => Promise<ExecResult>`, `ExecResult` carries `{ stdout, stderr, code }`.
- **`execRodCli(execFn, args, options?)`** -- Validate `args`, then run `rod-cli --raw --max-output 51200 <args>` through the host's exec function with the per-command timeout from `TIMEOUTS` and the caller's abort `signal`. Throws `rod-cli exited <code>: <stderr>` on a non-zero exit.
- **Pure argv-builders** -- `buildGotoArgs`, `buildSnapshotArgs`, `buildClickArgs`, `buildTypeArgs`, `buildEvalArgs`, `buildScreenshotArgs`, `buildScrollArgs`, `buildTabsArgs`, `buildNavigateArgs`, `buildCookiesArgs`, `buildStorageArgs`, `buildFillFormArgs`, `buildPressArgs`, `buildSelectArgs`, `buildHoverArgs`, plus `withSession`.
- **`captureScreenshot(execFn, params, signal?)`** -- Take a screenshot into a private temp file (`<os tmpdir>/rod-cli-<uuid>.png`) and return rod-cli's text plus the PNG as base64 image content (`ScreenshotContent`). The file is always deleted.
- **`waitForSelector(execFn, params, options?)`** -- Poll `document.querySelector` through `rod-cli eval` every 500 ms until the CSS selector appears; throws after `params.timeoutMs`. It always checks once, so `timeoutMs <= 0` is a single check with no polling. A failed `rod-cli` call rejects at once and is not retried. `buildWaitPollArgs` builds the poll argv.
- **`sleep(ms, signal?)`** -- Promise-based delay with abort support.
- **`validateInput(args)`** -- Reject a `goto` URL that is not `http://`/`https://`, an empty `click`/`fill`/`type` selector, empty `fill` text, an `eval` expression over 10,000 characters, and a `-s`/`--session` name outside `^[A-Za-z0-9._-]{1,64}$` (or `.`/`..`).
- **`timeoutFor(args)` / `TIMEOUTS`** -- Per-command timeout table (`goto` 60 s; `snapshot`, `click`, `fill`, `type`, `eval` 15 s; `screenshot`, `wait` 30 s; `close`, `--version` 5 s; anything else 30 s).
- **`findSubcommand(args)`** -- Index of the subcommand in an argv, skipping `-s`/`--session <name>` and other flags.

## Consumed By

- [`@agenthands/rod-cli-pi`](../pi/README.md) -- Pi extension (TypeBox schemas)
- [`@agenthands/rod-cli-omp`](../omp/README.md) -- OMP extension (injected `pi.zod` schemas)

## Release Process

See [`docs/extension-release.md`](../../docs/extension-release.md) for the release process, versioning convention, and how to cut a release.
