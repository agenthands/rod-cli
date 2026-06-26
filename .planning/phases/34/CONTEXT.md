---
author: architect
responsible: architect
phase: 34
phase_type: implementation
hard_bar: true
security_relevant: true
design_fork: false
status: locked
parent_artifacts:
  - .planning/REQUIREMENTS.md (BUILD-01, BUILD-02)
  - .planning/ROADMAP.md (Phase 34)
  - .planning/research/assistant-onboarding-SUMMARY.md (go.mod state confirmed)
---

# Phase 34: Toolchain Bump & Vuln Gate — CONTEXT

## What this phase delivers

A **supply-chain hardening phase** — the foundation every later v1.8 phase builds on:
- `go.mod` pins the toolchain to **go1.26.1** and `go build ./...` + `go test ./...` pass.
- `govulncheck ./...` reports **zero called-path stdlib vulns** (the 13 F1 vulns are closed).
- CI runs on go1.26.1 with a **govulncheck gate** so the supply-chain fix cannot regress.

## Success criteria (from REQUIREMENTS)

### BUILD-01 — Toolchain pin + build green
- `go.mod` declares `toolchain go1.26.1` and the `go` directive is aligned.
- `go build ./...` and `go test ./...` pass on go1.26.1.

### BUILD-02 — Vuln gate green + CI-wired
- `govulncheck ./...` reports **no known-vulnerable called paths** for the 13 stdlib vulns the v1.7 F1 audit flagged (GO-2026-4592 through GO-2026-4601, all fixed in go1.26.1).
- The `test.yml` CI workflow includes a `govulncheck` step so the gate cannot silently regress.
- Any residual finding (e.g. un-called-path vulns in deps) is documented with justification.

## Current state (before this phase)

| Item | Current | Target |
|------|---------|--------|
| `go.mod` `go` directive | `go 1.25.1` | aligned with toolchain |
| `go.mod` `toolchain` directive | **absent** | `toolchain go1.26.1` |
| Dev go version | go1.26.0 | go1.26.1 |
| `govulncheck` called-path vulns | **13** | **0** |
| `test.yml` go-version | `1.25.x` | `1.26.x` |
| `test.yml` govulncheck step | **absent** | **present** |
| `release.yml` go-version | `1.23` (stale!) | `1.26.x` |
| `release-binary.yml` go-version | `1.23.7` (stale!) | `1.26.x` |

The 13 vulns are all stdlib, all fixed in go1.26.1. No code changes are needed — the fix is purely the toolchain bump. But the CI files are stale (release workflows still reference go1.23 / go1.23.7).

## What the work entails

1. **`go.mod`**: Add `toolchain go1.26.1` directive; align `go` directive; run `go mod tidy`.
2. **`test.yml`**: Bump `go-version` from `1.25.x` to `1.26.x`; add a `govulncheck` step after the test step.
3. **`release.yml`**: Bump `go-version` from `1.23` to `1.26.x`.
4. **`release-binary.yml`**: Bump `go-version` from `1.23.7` to `1.26.x`.
5. **Verify**: `go build ./...` and `go test ./...` pass on go1.26.1; `govulncheck ./...` reports 0 called-path vulns.

## Key centerspiece symbols (SMTC-grounding)

The change is confined to three config files (`go.mod`, `.github/workflows/test.yml`, `.github/workflows/release.yml`, `.github/workflows/release-binary.yml`). No Go source symbols change — this is a pure toolchain/config phase. The engineering risk is: are we sure no code broke under go1.26.1? That's what the build+test pass verifies.

## Security relevance

This phase closes 13 called-path stdlib vulnerabilities (all fixed in go1.26.1). The `govulncheck` CI gate is a supply-chain regression guard. `security_relevant: true`.

## Dependencies

- **Depends on**: Phase 33 (v1.7 shipped — this is the first v1.8 phase).
- **Blocking for**: Phases 35, 36, 37 (all build on the fixed toolchain).

## Out of scope

- Upgrading non-stdlib dependencies beyond what `go mod tidy` touches.
- Fixing the 4 un-called-path vulns in deps (document as known-accepted if present after bump).
- Renovate/Dependabot configuration.
