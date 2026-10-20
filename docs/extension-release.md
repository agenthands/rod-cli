# Extension Release Process

## Tag Convention

Extension releases use the `ext-v*` tag prefix (e.g. `ext-v0.1.0`, `ext-v0.2.0`).

This is **independent** of the Go binary `v*` tags — extension semver is decoupled from the CLI tool's version.

## Versioning

The three packages (`@agenthands/rod-cli-core`, `@agenthands/rod-cli-pi`, `@agenthands/rod-cli-omp`) are **versioned together** per release. All three share the same version number.

## How to Cut a Release

1. **Bump versions** — update `version` in all three `extensions/{core,pi,omp}/package.json` to the new version (e.g. `0.2.0`), and the `@agenthands/rod-cli-core` range in `extensions/{pi,omp}/package.json` `dependencies` to match (e.g. `^0.2.0`; `^0.1.0` does not match `0.2.0`). Then run `npm install` at the repo root to update `package-lock.json`. Done when `npm ci` at the repo root succeeds and `npm run typecheck && npm test` passes in each of `extensions/core`, `extensions/pi`, `extensions/omp` — the same gate the workflow runs.
2. **Commit** the version bumps and `package-lock.json`.
3. **Tag** the release:
   ```bash
   git tag ext-vX.Y.Z
   ```
4. **Push** the tag:
   ```bash
   git push --tags
   ```
5. **Workflow** — the `publish-extensions` GitHub Action (`.github/workflows/publish-extensions.yml`) triggers on `ext-v*` tags, runs `npm ci` at the repo root, runs the CI-01 gate (`npm run typecheck && npm test` for all three packages), then runs `npm publish --access public` in dependency order: core → pi → omp.

## Prerequisites

- The `@agenthands` npm organization must exist.
- An `NPM_TOKEN` secret with publish permissions must be configured in the GitHub repository.
