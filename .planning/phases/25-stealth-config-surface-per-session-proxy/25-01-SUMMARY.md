---
phase: 25-stealth-config-surface-per-session-proxy
plan: 01
subsystem: config
tags: [stealth, proxy, profile, cli-flags, daemon, precedence, godoll]

# Dependency graph
requires:
  - phase: 24-detection-harness-and-ci-backbone
    provides: detection harness + live-binary test conventions used as the regression net
provides:
  - StealthConfig sub-struct on types.Config (Proxy, ProxyAuth, ProfilePath) — the single home for every stealth knob
  - ResolveStealth precedence resolver (CLI flag > profile file > built-in default), computed once at daemon spawn
  - StealthFlags input struct capturing raw CLI flag values
  - --proxy / --proxy-auth / --profile global CLI flags forwarded verbatim across the daemon-spawn boundary
  - already-running-session stderr warning (no silent ignore, no auto-restart)
  - loud-failure on a missing/bad --profile file
affects: [25-02 (proxy wiring rides Stealth.Proxy), 26 (fingerprint pins overlay ResolveStealth), 27 (WebRTC/canvas fields), 28 (humanize knobs)]

# Tech tracking
tech-stack:
  added: []  # no new third-party modules; godoll/stealth already in go.mod
  patterns:
    - "Single stealth-config funnel: flag → EnsureDaemon forward → ResolveStealth → Config.Stealth → NewContext freeze"
    - "Reservation comment marks where Ph26-28 fingerprint/hardening/humanize fields land without committing field shapes now"
    - "Deprecated-but-bridged compatibility shim (Config.Proxy ← Stealth.Proxy) to avoid breaking the in-flight launchBrowser call site until Plan 02"

key-files:
  created: []
  modified:
    - types/config.go
    - types/config_test.go
    - cmd.go
    - cmd_test.go
    - main.go
    - rod-cli.yaml

key-decisions:
  - "Profile path resolution: a bare --profile name resolves under ~/.rod-cli/profiles/<name>.json; a value containing a separator or .json suffix (or an existing file) is used verbatim."
  - "This phase tracks only that a profile was selected (ProfilePath); the profile's identity fields are deliberately NOT overlaid onto Config yet — that is Phase 26's job."
  - "Config.Proxy kept as a deprecated bridged shim rather than removed, so types/context.go:71 still compiles until Plan 02 rewires it through godoll's proxy API."
  - "Stealth flags are forwarded verbatim into the daemon argv (EnsureDaemon needs no change); persistence is automatic via one-daemon-per-session process isolation."

patterns-established:
  - "Stealth config is resolved exactly once, before NewContext freezes Config, and inherited by every later command on the session."
  - "Warnings to stderr only; stdout stays clean for --raw/piped callers."
  - "Credential safety: --proxy-auth is never echoed to stdout/stderr and never logged in runDaemonServer."

requirements-completed: [PROFILE-01, PROFILE-02]

# Metrics
duration: ~20min
completed: 2026-06-24
status: complete
---

# Phase 25 Plan 01: Stealth Config Surface — Session-Persistent Substrate Summary

**A cohesive `StealthConfig` substrate with a CLI > profile > default precedence resolver, three forwarded global flags, and a loud-failure profile load — established as the single place every later v1.6 stealth feature plugs into.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-06-24
- **Tasks:** 3 of 3
- **Files modified:** 6

## Accomplishments
- Added `StealthConfig` (Proxy, ProxyAuth, ProfilePath) to `types.Config` with a documented reservation block for the Phase 26–28 fingerprint/hardening/humanize fields, plus `ResolveStealth` implementing the locked precedence CLI flag > profile file > built-in default with a loud error on a missing/bad profile.
- Registered `--proxy` / `--proxy-auth` / `--profile` global flags and forwarded each non-empty value verbatim across the daemon-spawn boundary via `EnsureDaemon`'s flags arg — the persistence linchpin.
- Wired `ResolveStealth` into `runDaemonServer` before `NewContext`, so stealth config is resolved exactly once per session and frozen; a bad profile aborts daemon start instead of shipping a default identity.
- Added the already-running-session stderr warning (never stdout) and verified the proxy-auth credential is never echoed or logged.

## Task Commits

Each task was committed atomically:

1. **Task 1: StealthConfig sub-struct + ResolveStealth resolver** — `da1fdb2` (feat, TDD)
2. **Task 2: Register + forward --proxy/--proxy-auth/--profile flags** — `c888289` (feat)
3. **Task 3: Resolve precedence in runDaemonServer before NewContext** — `8a193de` (feat)

_Note: Task 1 was TDD — the implementation and its tests landed together in one feat commit (single-feature plan; types package must compile to host the tests)._

## Files Created/Modified
- `types/config.go` — Added `StealthConfig`, `StealthFlags`, `ResolveStealth`, `resolveProfilePath`; bridged the deprecated `Config.Proxy` from `Stealth.Proxy`; added `Stealth` to `DefaultConfig`.
- `types/config_test.go` — Tests for all four precedence tiers, CLI-over-profile, loud missing-profile error, proxy-auth bridge, and YAML stealth-block round-trip.
- `cmd.go` — Three new global flags; forwarding block; `daemonRunning` helper; already-running stderr warning with credential redaction.
- `cmd_test.go` — `captureStderr` helper; tests for flags-in-help, single stderr warning + no credential leak + clean stdout, no-warning gate, and bad-profile loud daemon failure.
- `main.go` — `StealthFlags` construction from forwarded flags + `ResolveStealth` call before `NewContext`.
- `rod-cli.yaml` — Regenerated default config now carries the `stealth:` block (schema reflects `StealthConfig`).

## Decisions Made
- Bare `--profile <name>` resolves under `~/.rod-cli/profiles/<name>.json`; path-like or `.json`/existing values used verbatim (Claude's discretion per CONTEXT decision).
- Profile identity fields are NOT overlaid onto Config in this phase — only `ProfilePath` is recorded; identity population is Phase 26's responsibility.
- `Config.Proxy` retained as a deprecated bridged shim (not removed) so `types/context.go` launchBrowser still compiles until Plan 02 rewires the proxy through godoll.

## Threat Mitigations Applied
- **T-25-02 (credential in stdout/stderr/log):** proxy-auth never echoed in the warning; `runDaemonServer` does not log the resolved proxy. Verified by test asserting no `s3cret` substring in stdout/stderr.
- **T-25-03 (malformed/missing profile):** `ResolveStealth` returns a wrapped loud error; `runDaemonServer` aborts daemon start. Verified by `TestRunDaemonServerBadProfile`.
- **T-25-04 (silent ignore on running daemon):** stderr warning emitted ("apply at spawn — close first"), command proceeds. Verified by `TestStealthFlagAlreadyRunningWarns`.

## Deviations from Plan
None — plan executed exactly as written. (Task 1's TDD test+impl landed in a single feat commit rather than separate test→feat commits, because the `types` package must compile to host the new tests; the RED/GREEN intent is preserved in the test coverage.)

## Verification
- `go build ./...` and `go vet ./...` clean.
- `rod-cli --help` lists `--proxy`, `--proxy-auth`, `--profile` as global flags (confirmed via live binary).
- `go test ./types/... ./daemon/...` passes (full suites green).
- An already-running session + stealth flag emits exactly one stderr warning and still succeeds; stdout unpolluted; no credential leak.
- A `--profile` at a non-existent file fails the daemon loudly.

## Known Stubs
None. Profile identity-field population is intentionally deferred to Phase 26 (documented above and in the `ProfilePath`-only resolution comment), not a stub blocking this plan's goal.

## Self-Check: PASSED

All modified files exist on disk and all three task commits (`da1fdb2`, `c888289`, `8a193de`) are present in git history.
