# Extension Release Process

## Tag Convention

Extension releases use the `ext-v*` tag prefix (e.g. `ext-v0.1.0`, `ext-v0.2.0`).

This is **independent** of the Go binary `v*` tags — extension semver is decoupled from the CLI tool's version.

## Versioning

The three packages (`@agenthands/rod-cli-core`, `@agenthands/rod-cli-pi`, `@agenthands/rod-cli-omp`) are **versioned together** per release. All three share the same version number.

## How to Cut a Release

1. **Bump versions** — update `version` in all three `extensions/{core,pi,omp}/package.json` to the new version (e.g. `0.2.0`).
2. **Commit** the version bumps.
3. **Tag** the release:
   ```bash
   git tag ext-vX.Y.Z
   ```
4. **Push** the tag:
   ```bash
   git push --tags
   ```
5. **Workflow** — the `publish-extensions` GitHub Action triggers on `ext-v*` tags, runs the CI-01 gate (typecheck + test for all three packages), then publishes in dependency order: core → pi → omp.

## Prerequisites

- The `@agenthands` npm organization must exist.
- An `NPM_TOKEN` secret with publish permissions must be configured in the GitHub repository.
