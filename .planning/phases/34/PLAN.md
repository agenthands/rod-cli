---
author: architect
responsible: architect
phase: 34
status: ready_for_execution
parent_artifacts:
  - .planning/phases/34/CONTEXT.md
  - .planning/REQUIREMENTS.md (BUILD-01, BUILD-02)
---

# Phase 34: Toolchain Bump & Vuln Gate — PLAN

## §1. go.mod — pin toolchain to go1.26.1

**Files:** `go.mod`

**Changes:**
1.1 — Add `toolchain go1.26.1` directive (on its own line after `go 1.25.1`).
1.2 — Update `go` directive from `1.25.1` to `1.26.1` (align with toolchain).
1.3 — Run `go mod tidy` to normalize the module file.

**Expected diff:**
```diff
-go 1.25.1
+go 1.26.1
+
+toolchain go1.26.1
```

## §2. test.yml — bump Go + add govulncheck gate

**Files:** `.github/workflows/test.yml`

**Changes:**
2.1 — Bump `go-version` from `'1.25.x'` to `'1.26.x'`.
2.2 — After the test step, add a `govulncheck` step:
```yaml
      - name: Run govulncheck
        run: go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

## §3. release.yml — bump Go

**Files:** `.github/workflows/release.yml`

**Changes:**
3.1 — Bump `go-version` from `'1.23'` to `'1.26.x'`.

## §4. release-binary.yml — bump Go

**Files:** `.github/workflows/release-binary.yml`

**Changes:**
4.1 — Bump `go-version` from `'1.23.7'` to `'1.26.x'`.

## §5. Verification gates

| Gate | Command | Expected |
|------|---------|----------|
| Build | `go build ./...` | exit 0 |
| Test | `go test ./... -count=1` | exit 0, all pass |
| Vuln | `govulncheck ./...` | `No vulnerabilities found.` for called paths |
| mod tidy | `go mod tidy && git diff --exit-code` | no diff |

## §6. Edge cases / risks

- **go1.26.1 not installed locally**: The engineer's machine may need `go install golang.org/dl/go1.26.1@latest && go1.26.1 download`. CI uses `actions/setup-go@v5` which resolves `1.26.x` automatically.
- **Residual govulncheck findings**: Un-called-path vulns in deps (4 reported) are NOT blocking — document them in a one-line note if they persist post-bump.
- **go mod tidy may change indirect deps**: Accept the changes; they are the go1.26.1-resolved versions.

## §7. Acceptance criteria (engineer handshake)

The engineer accepts this plan if:
1. All changes are limited to the 4 files listed.
2. All 5 verification gates are clearly defined.
3. The phase has no Go source code changes — pure config.

The engineer should refuse if:
- A verification gate is ambiguous or missing.
- The plan requires source changes not enumerated here.
- A non-obvious risk is identified (document the `basis + alternative`).
