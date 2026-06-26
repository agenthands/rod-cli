---
author: engineer
responsible: engineer
phase: 30-cdp-footprint-reduction
plan: 03
type: execute-summary
status: complete
requirements: [CDP-02, CDP-03]
commit: ebb265d
---

# Phase 30 Wave 3 — SUMMARY (inventory + honest ceiling + live leg)

## Outcome
CDP-02 (inventory with mitigation status) and the CDP-03 live-measurement leg are
closed honestly. The Phase-24 "Answer: No" findings note is superseded by a current
per-domain inventory that is TRUE against HEAD.

## What landed (per task)
- **Task 1 — cdp-footprint.md rewrite.** Replaced the findings note with a
  per-domain inventory table (Runtime, Network, Fetch, Emulation, DOM, Page/Target)
  with columns when-sent → status (reduced | obfuscated | accepted-visible):
  - Runtime/Network/Fetch → **reduced** (opt-in / lazy, off on a plain session,
    triggering flag/feature named).
  - Emulation.setUserAgentOverride → **obfuscated** (zero-enable identity; coherent
    with the JS identity; WIRE-VERIFY cited).
  - Page/Target → **accepted-visible**, with the `earn-accepted-visible` lens
    applied: stated WHY (irreducible CDP-transport cost; any CDP browser enables
    Page; reducing needs browser-patching/alternate transport → v2).
  Includes a measurement section (Tier-1 ledger gate + WIRE-VERIFY + informational
  cdpTell), an explicit Honest Ceiling naming **v2 CDP-DEEP-01**, no "undetectable"
  claim, and cross-links to stealth-validation.md.
- **Task 2 — stealth-validation.md.** Updated the CDP-transport ceiling bullet from
  "deferred to v2 (CDP-01), not attempted in v1.6" to the v1.7 reduced reality
  (plain session enables none of Runtime/Network/Fetch, asserted by the Tier-1
  baseline; irreducible Page/Target remains; deep obfuscation → v2 CDP-DEEP-01).
  Refreshed the back-link. Tier-1/Tier-2 honesty framing intact.
- **Task 3 — live (Tier-2) observation.** Added a `cdp-footprint` subtest to
  `tests/detection_live_test.go`: navigates a benign live target on the PLAIN
  session path and logs the self-contained `cdpTell` heuristic (identical to the
  offline `detect.js` probe). t.Logf/t.Skip only — never Fatal/Errorf; the
  `//go:build detection_live` tag stays line 1 (excluded from the CI gate; test.yml
  unchanged).

## Deviations / code-review fixes
`anvil-code-reviewer` verdict: **SHIP**. One MAJOR doc-accuracy overclaim caught and
fixed (the cardinal sin this doc guards against):
- **Plugin path (MAJOR).** `BindLifecycle` (`internal/plugin/lifecycle.go`) runs on
  **every** `PluginLoad` and enables `DOM` + subscribes to `Network`
  **unconditionally** — regardless of whether the plugin defines
  `onRequest`/`onResponse`/`onDOMNodeInserted`. My first draft said "only when a
  plugin with lifecycle hooks is loaded," which is rosier than reality. Both the
  Network and DOM rows now key on plugin *load*.
- **Ledger-bypass honesty (MINOR).** The Scope note now names the plugin binder as
  the concrete instance of the disclosed "domain enabled via a path that bypasses
  the ledger" gap (it enables Network/DOM on the wire but does not call
  `recordCDPDomainLocked`).
- **Cross-ref drift (MINOR).** `internal/detect/detect.js` cdpTell comment updated
  from the stale "v2 CDP-01" to "CDP-DEEP-01" so the offline/live/doc trio agree.

## Verification
- `go build ./...` and `go build -tags detection_live ./...` both compile.
- `go vet -tags detection_live ./tests/` clean; live subtest uses only Logf/Skip;
  build tag is line 1; CI `test.yml` runs `go test ./... -count=1` with NO tags, so
  the live suite stays excluded (confirmed `go test -run TestLiveDetection` → "no
  tests to run").
- Doc claims self-verified against HEAD (capture defaults, lazy Fetch, Emulation
  no-enable, DOMEnable sole call site, X-Requested-With absent) and independently
  re-verified by the code-reviewer.
- internal/detect embed test green after the detect.js comment edit.
