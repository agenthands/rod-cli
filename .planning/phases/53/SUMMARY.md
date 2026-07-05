---
author: engineer
phase: 53
status: complete
---

# Phase 53: Publish Readiness + Auto-Publish Pipeline — SUMMARY

## must_have confirmations

### 1. All three gates green after edits
- **core:** 139 vitest passed, `tsc --noEmit` clean
- **pi:** 38 vitest passed, `tsc --noEmit` clean
- **omp:** 30 bun test passed, `tsc --noEmit` clean
- No tool/core source changed.

### 2. LICENSE in every tarball
`npm pack --dry-run --json` for each package confirms:

**core** (11 files): `LICENSE`, `README.md`, `package.json`, `src/argv.ts`, `src/binary.ts`, `src/exec.ts`, `src/index.ts`, `src/subcommand.ts`, `src/timeouts.ts`, `src/validate.ts`, `src/wait.ts`
- NO `__tests__`, `.smtc-cache`, `tmp`, `log`, `rod`, `node_modules`

**pi** (20 files): `LICENSE`, `README.md`, `package.json`, `src/index.ts`, `src/lifecycle.ts`, `src/types.ts`, `src/tools/click.ts`, `src/tools/cookies.ts`, `src/tools/eval.ts`, `src/tools/fill_form.ts`, `src/tools/goto.ts`, `src/tools/index.ts`, `src/tools/navigate.ts`, `src/tools/screenshot.ts`, `src/tools/scroll.ts`, `src/tools/snapshot.ts`, `src/tools/storage.ts`, `src/tools/tabs.ts`, `src/tools/type.ts`, `src/tools/wait.ts`
- NO `__tests__`, `.smtc-cache`, `tmp`, `log`, `rod`, `node_modules`

**omp** (19 files): `LICENSE`, `README.md`, `package.json`, `src/index.ts`, `src/lifecycle.ts`, `src/tools/click.ts`, `src/tools/cookies.ts`, `src/tools/eval.ts`, `src/tools/fill_form.ts`, `src/tools/goto.ts`, `src/tools/index.ts`, `src/tools/navigate.ts`, `src/tools/screenshot.ts`, `src/tools/scroll.ts`, `src/tools/snapshot.ts`, `src/tools/storage.ts`, `src/tools/tabs.ts`, `src/tools/type.ts`, `src/tools/wait.ts`
- NO `__tests__`, `.smtc-cache`, `tmp`, `log`, `rod`, `node_modules`

### 3. Dep precision

**core/package.json** — no `dependencies`:
```json
"devDependencies": {
  "@types/node": "^22.0.0",
  "typescript": "^5.8.0",
  "vitest": "^3.0.0"
}
```

**pi/package.json** — NO `peerDependencies` block; runtime deps in `dependencies`, type-only in `devDependencies`:
```json
"dependencies": {
  "@agenthands/rod-cli-core": "^0.1.0",
  "typebox": "^1.3.4",
  "@earendil-works/pi-ai": "^0.80.3"
},
"devDependencies": {
  "@earendil-works/pi-coding-agent": "^0.80.3",
  "@types/node": "^22.0.0",
  "typescript": "^5.8.0",
  "vitest": "^3.0.0"
}
```

**omp/package.json** — core as real dep, host type-only in devDeps:
```json
"dependencies": {
  "@agenthands/rod-cli-core": "^0.1.0"
},
"devDependencies": {
  "@oh-my-pi/pi-coding-agent": "^16.3.6",
  "@types/bun": "latest",
  "typescript": "^5.8.0"
}
```

### 4. PI-03 host-LOAD proven
Packed core + pi tarballs installed into a scratch dir (no repo `node_modules`). The extension loaded from the scratch `node_modules/@agenthands/rod-cli-pi/` and registered all **13 tools**:

```
browse_goto, browse_snapshot, browse_click, browse_type, browse_eval,
browse_screenshot, browse_wait, browse_tabs, browse_navigate, browse_scroll,
browse_cookies, browse_storage, browse_fill_form
```

Deps resolved from scratch:
- `require.resolve(@agenthands/rod-cli-core)`: `/tmp/tmp.*/node_modules/@agenthands/rod-cli-core`
- `require.resolve(typebox)`: `/tmp/tmp.*/node_modules/typebox`
- `require.resolve(@earendil-works/pi-ai)`: `/tmp/tmp.*/node_modules/@earendil-works/pi-ai`

Round-trip `browse_goto` + `browse_snapshot` against a local HTTP fixture confirmed the snapshot contains `"rod-cli fixture page"`.

### 5. OMP host-LOAD proven
Packed core + omp tarballs + `zod` installed into a scratch dir. The extension loaded from scratch `node_modules/@agenthands/rod-cli-omp/` and registered all **13 tools**. Same goto+snapshot round-trip confirmed fixture content. Deps resolved from scratch.

### 6. Publish dry-run
All three packages pass `npm publish --dry-run --access public` (exit 0):
- `+ @agenthands/rod-cli-core@0.1.0`
- `+ @agenthands/rod-cli-pi@0.1.0`
- `+ @agenthands/rod-cli-omp@0.1.0`

### 7. Workflow disjointness
- `publish-extensions.yml` triggers on `tags: ["ext-v*"]` — prefix-disjoint from the Go `v*` trigger in `release.yml`.
- Gate steps (typecheck + test for all three) run before any publish step; a gate failure blocks the job.

### 8. Rule sweep
`grep -rnE ': any|as any|await import\(|no-explicit-any' extensions/*/src --include='*.ts'` → **empty** (no violations).

## Files created/modified

| File | Status |
|---|---|
| `LICENSE` | NEW (MIT, Copyright (c) 2026 agenthands) |
| `extensions/core/LICENSE` | NEW |
| `extensions/pi/LICENSE` | NEW |
| `extensions/omp/LICENSE` | NEW |
| `extensions/core/package.json` | MODIFIED (publishConfig, repository, homepage, bugs, engines) |
| `extensions/pi/package.json` | MODIFIED (publish fields + dep hygiene, peerDependencies deleted) |
| `extensions/omp/package.json` | MODIFIED (publish fields + core range ^0.1.0) |
| `extensions/core/README.md` | NEW (stub) |
| `extensions/omp/README.md` | NEW (stub) |
| `extensions/pi/README.md` | MODIFIED (release-doc reference) |
| `.github/workflows/publish-extensions.yml` | NEW |
| `docs/extension-release.md` | NEW |
| `package-lock.json` | MODIFIED (npm install) |

## Honest ceiling note

**Live npm publish is blocked** on a maintainer-provisioned `@agenthands` npm organization and `NPM_TOKEN` secret in the GitHub repository. Everything short of the live push has been verified:
- `npm pack --dry-run` confirms correct tarball contents for all three packages
- Scratch-install host-LOAD proves the packed tarballs install and load correctly with deps resolved from public npm
- `npm publish --dry-run --access public` confirms the publish command shape and exits 0

The `publish-extensions.yml` workflow is ready to execute the live publish once the org/token are provisioned.
