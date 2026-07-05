# @agenthands/rod-cli-core

Runtime-agnostic core for rod-cli host extensions -- binary discovery, normalized exec contract, input validation, timeout table, and pure argv-builders for the `browse_*` tools.

**This is a shared library, not installed directly by end users.** It is a real `dependencies` entry consumed by the host-specific adapter packages.

## Public Surface

- **`findRodCli()`** -- Locate the `rod-cli` binary on `$PATH` or common install locations.
- **`ExecFn` / `ExecResult`** -- Normalized exec contract: `ExecFn` is the host's exec function, `ExecResult` carries `{ stdout, stderr, exitCode }`.
- **`execRodCli(execFn, args, options?)`** -- Run a rod-cli command through the host's exec function with timeout and signal support.
- **13 pure argv-builders** -- `buildGotoArgs`, `buildSnapshotArgs`, `buildClickArgs`, `buildTypeArgs`, `buildEvalArgs`, `buildScreenshotArgs`, `buildScrollArgs`, `buildTabsArgs`, `buildNavigateArgs`, `buildCookiesArgs`, `buildStorageArgs`, `buildFillFormArgs`, plus `withSession`.
- **`waitForSelector(execFn, params, options?)`** -- Poll the DOM until a CSS selector appears (or timeout).
- **`sleep(ms, signal?)`** -- Promise-based delay with abort support.

## Consumed By

- [`@agenthands/rod-cli-pi`](../pi/README.md) -- Pi extension (TypeBox schemas)
- [`@agenthands/rod-cli-omp`](../omp/README.md) -- OMP extension (injected `pi.zod` schemas)

## Release Process

See [`docs/extension-release.md`](../../docs/extension-release.md) for the release process, versioning convention, and how to cut a release.
