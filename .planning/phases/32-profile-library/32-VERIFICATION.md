# Phase 32: Profile Library — Verification

**Verdict:** ✅ PASSED
**Verified:** 2026-06-26 (independent qa, goal-backward against HEAD)
**Requirements:** PROF-01, PROF-02, PROF-03, PROF-04 — all delivered in code.

## Success criteria (ROADMAP Phase 32)
1. **5-10 coherent embedded profiles, validated** — ✅ 6 Chrome-only desktop profiles
   in `types/profiles/*.json`, embedded via `//go:embed` (`types/profiles_embed.go:20`);
   ship in the binary.
2. **`--profile=list` + select by name** — ✅ `maybeHandleProfileList` (cmd.go:27)
   intercepts before any daemon spawn; human/`--raw`/`--json` all print 6 names and
   exit; `list` reserved (never loaded). `--profile=<name>` selects a built-in.
3. **Custom profiles + precedence** — ✅ built-in resolves before user-dir
   (`config.go:307`); path-like values load verbatim (PROF-04); CLI>profile>default
   unchanged; loud-failure preserved.
4. **Each passes the v1.6 consistency validator** — ✅ `TestBuiltinProfilesAreVetted`
   (`types/profiles_test.go:43`) iterates ALL built-ins through
   `deriveAndValidateFingerprint` + structural backstops; non-vacuous (verified).
   Harness leg `TestDetectionHarness/builtin_profiles` drives win11/win10-laptop/
   macos-applesilicon end-to-end through real Chrome.

## Notable
- The vetting gate uses a STRONGER userSet than ResolveStealth — qa assessed it as a
  legitimate safe-direction strengthening (gate rejects more; runtime cannot diverge),
  not a gap.
- deviceMemory validator bound tightened 64→8 (W3C Device Memory API cap) — correct,
  no config/test regression.
- Constraint honored: no TLS-spoofing surface in the schema; all profiles real Chrome
  ([[no-tls-spoofing-real-chrome]]).

## Post-verify fix (architect)
- qa flagged windows-11-chrome.json and windows-10-chrome.json as byte-identical
  (Chrome UA can't distinguish Win10/Win11; the distinguishing high-entropy client
  hint isn't in the schema). Differentiated windows-10-chrome → hardwareConcurrency 12,
  timezone America/Chicago (still en-US coherent) so no two shipped profiles share a
  fingerprint. Consistency gate re-run: PASS.

## Evidence
`go build ./...`, `go vet ./...` clean. `go test ./types/ -run "Builtin|Profile"` PASS
(0.005s). `go test ./tests/ -run TestDetectionHarness/builtin_profiles` PASS (~22.7s).
`./rod-cli --profile=list` (+ --raw/--json) prints 6 names, no daemon.
