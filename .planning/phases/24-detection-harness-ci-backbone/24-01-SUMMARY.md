---
phase: 24-detection-harness-ci-backbone
plan: 01
subsystem: testing
tags: [detection-harness, go-embed, testserver, stealth, webrtc, cdp, webgl]

# Dependency graph
requires: []
provides:
  - "internal/detect package: offline 127.0.0.1:0 detection fixture server (New/Start/URL/Close)"
  - "Self-authored //go:embed-bundled detection page (detect.html + detect.js)"
  - "window.__detect JS global with per-signal table-stakes verdicts + informational webrtcIce/cdpTell, gated by window.__detect.ready"
affects: [24-02, 24-03, 24-04, 25-stealth-config, 26-fingerprint-validator, 27-hardening, 28-humanize, 29-livewaf]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Ephemeral-port offline fixture server (net.Listen 127.0.0.1:0 + go http.Serve)"
    - "String-form //go:embed bundling for static assets (matches types/js/js.go)"
    - "window.__detect ready-gated async probe global (permissions + WebRTC settle before ready)"

key-files:
  created:
    - internal/detect/server.go
    - internal/detect/embed.go
    - internal/detect/detect.html
    - internal/detect/detect.js
    - internal/detect/server_test.go
  modified: []

key-decisions:
  - "Detection page self-authored (no bot.sannysoft vendoring — no clean license)"
  - "WebRTC ICE leak and CDP-tell recorded as informational, non-blocking signals (KNOWN-RED, fixed in Phase 27)"
  - "Each probe wrapped in try/catch recording its own error string so one failing probe never blanks the global"
  - "Async probes (permissions, WebRTC) settle into the global before ready=true, with a 3s timeout fallback"
  - "/result POST scorecard sink kept minimal (harness reads window.__detect via eval; sink is convenience only)"

patterns-established:
  - "Ephemeral-port offline fixture: net.Listen tcp 127.0.0.1:0 -> go http.Serve -> URL() from listener.Addr()"
  - "go:embed string bundling for single-binary constraint (import _ \"embed\" + //go:embed file + package-level string var)"
  - "Ready-gated probe global: window.__detect populated by sync + async probes, window.__detect.ready flips true only after async probes settle"

requirements-completed: [HARNESS-01]

# Metrics
duration: 2min
completed: 2026-06-23
status: complete
---

# Phase 24 Plan 01: Detection Harness Fixture Server Summary

**Offline `127.0.0.1:0` detection fixture server (`internal/detect`) serving a self-authored, `//go:embed`-bundled sannysoft-style probe page that writes table-stakes + informational WebRTC/CDP verdicts into `window.__detect`.**

## Performance

- **Duration:** 2 min
- **Started:** 2026-06-23T22:38:27Z
- **Completed:** 2026-06-23T22:40:08Z
- **Tasks:** 2
- **Files modified:** 5 (all created)

## Accomplishments
- New `internal/detect` package builds cleanly and is the deterministic, offline backbone the rest of v1.6 (Phases 25–29) asserts against.
- `detect.js` probes every table-stakes signal — `webdriver`, `pluginsLength`, `mimeTypesLength`, `userAgent`, `webglVendor`/`webglRenderer` (via `WEBGL_debug_renderer_info` 37445/37446), `permissionsConsistent` (Notification.permission vs permissions.query), `languages`, `screen`/`outerSize`, `windowChrome`/`chromeRuntime`, `timezone` — plus informational `webrtcIce` (KNOWN-RED leak) and `cdpTell`, all written into `window.__detect` and gated by `window.__detect.ready`.
- `DetectServer` mirrors `internal/plugin/scanner/testserver/server.go` exactly: loopback-only ephemeral bind, `New`/`Start`/`URL`/`Close` lifecycle, serving embedded HTML at `/` and JS at `/detect.js`, with an optional minimal `/result` scorecard sink.
- `server_test.go` proves the offline server + embed wiring without a browser (loopback URL, embedded page + script served, 200 OK).

## Task Commits

Each task was committed atomically:

1. **Task 1: Embedded self-authored detection page (detect.html + detect.js + embed.go)** - `6ed6bef` (feat)
2. **Task 2: 127.0.0.1:0 detection fixture server (server.go) + unit test** - `aec3fe3` (feat)

**Plan metadata:** see final docs commit below.

## Files Created/Modified
- `internal/detect/embed.go` - String-form `//go:embed` glue binding `detectHTML` and `detectJS`.
- `internal/detect/detect.html` - Minimal offline page shell loading `/detect.js`.
- `internal/detect/detect.js` - Dependency-free probe script populating `window.__detect`, ready-gated on async permissions + WebRTC probes.
- `internal/detect/server.go` - `DetectServer` loopback fixture (New/Start/URL/Close), routes for `/`, `/detect.js`, `/result`.
- `internal/detect/server_test.go` - Stdlib unit test asserting embedded HTML + JS served on loopback.

## Decisions Made
- Self-authored detection page (no bot.sannysoft vendoring) per the locked 24-CONTEXT decision (no clean license).
- WebRTC ICE leak and CDP-tell are recorded as informational, non-blocking signals (KNOWN-RED; HARDEN-01 fixes WebRTC in Phase 27) rather than asserted-blocking — this plan is the measuring instrument, not an evasion fix.
- Every probe records its own error string into its signal instead of throwing, so a single failing probe never blanks the whole global.
- Async probes settle into the global before `ready=true`, with a 3s timeout fallback so the global is always eventually populated for the e2e poller.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Known Stubs
The optional `/result` POST scorecard sink is intentionally minimal — the e2e harness (plan 24-03) reads `window.__detect` directly via the `eval` command and does not depend on this path. It is a documented convenience accessor (`LastScorecard()`), not an unfinished feature.

## Next Phase Readiness
- `internal/detect` package builds (`go build ./...` passes) and its unit test passes; ready for plan 24-03 (e2e test that drives the real `rod-cli` binary against `detect.New()` and reads `window.__detect` via `eval`).
- The WebRTC `webrtcIce` and `cdpTell` informational signals are in place for the CDP-findings note (plan 24-04) and the KNOWN-RED WebRTC baseline (Phase 27 HARDEN-01).

## Self-Check: PASSED

All 5 created files exist on disk; both task commits (`6ed6bef`, `aec3fe3`) present in git history.

---
*Phase: 24-detection-harness-ci-backbone*
*Completed: 2026-06-23*
