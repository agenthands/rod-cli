---
phase: 24-detection-harness-ci-backbone
reviewed: 2026-06-24T00:00:00Z
depth: deep
files_reviewed: 10
files_reviewed_list:
  - internal/detect/server.go
  - internal/detect/embed.go
  - internal/detect/detect.html
  - internal/detect/detect.js
  - internal/detect/server_test.go
  - types/context.go
  - tests/detection_test.go
  - .github/workflows/test.yml
  - docs/cdp-footprint.md
  - .gitignore
findings:
  critical: 0
  warning: 2
  info: 5
  total: 7
status: findings
---

# Phase 24: Code Review Report

**Reviewed:** 2026-06-24
**Depth:** deep
**Files Reviewed:** 10
**Status:** findings

## Summary

Phase 24 ships an offline, self-authored bot-detection fixture (`internal/detect`),
an e2e harness that drives the real `rod-cli` binary against it
(`tests/detection_test.go`), a VALIDATE-03 observability change in
`types/context.go`, and a CI workflow. Overall quality is high for test/CI
infrastructure: the fixture binds loopback-only with no reflected input, every JS
probe is try/catch-wrapped, async probes settle before `ready`, and the
KNOWN-RED assertions are real executing assertions (no `t.Skip`).

The VALIDATE-03 change is **correct**: the inner `if err := em.Apply(); err != nil`
and `if err := ctx.interceptor.Enable(); err != nil` blocks use a fresh `err`
scoped to the `if`, so the outer page-creation `err` (from `browser.Page()` at
line 282) is preserved for the final check at line 327. Warnings go to stderr only;
stdout stays clean for `--raw`; no daemon hard-fail. No blockers found.

The findings below are correctness/robustness nits in the detection probes and the
e2e harness, none of which block shipping. The two warnings concern probes that
**silently under-detect** their target signal, which weakens the harness's value as
a regression net (a real leak could go unnoticed while the test still passes).

## Warnings

### WR-01: WebRTC ICE probe only matches IPv4 — misses mDNS/IPv6 candidates, so the KNOWN-RED leak signal can read empty even when a leak surface exists

**File:** `internal/detect/detect.js:208`
**Issue:** The candidate parser is `/([0-9]{1,3}(?:\.[0-9]{1,3}){3})/`, which only
captures dotted-quad IPv4. Modern Chrome obfuscates host candidates behind mDNS
`*.local` hostnames by default (and may emit IPv6), so `ev.candidate.candidate`
frequently contains **no** IPv4 literal. The probe then records `d.webrtcIce = ""`.

The KNOWN-RED test (`tests/detection_test.go:189`) only fails when the value is
`"undefined"`, treating `""` as acceptable baseline truth. The net effect: when
Phase 27 (HARDEN-01) is supposed to flip this to required-empty, the signal may
**already read empty for the wrong reason** (mDNS masking, not evasion), so the
assertion provides no real regression protection and could mask a future IPv4 host
leak that the regex also fails to catch in some candidate formats. This is a
correctness gap in what the probe claims to measure, not a crash.

**Fix:** Broaden the parser to also record mDNS and IPv6 candidates so the recorded
truth reflects what was actually gathered, e.g. capture the full candidate
address token rather than IPv4-only:
```javascript
// Record any host token: IPv4, IPv6, or mDNS *.local hostname.
var cand = ev.candidate.candidate || "";
var m = /candidate:\S+ \d+ \S+ \d+ (\S+) /.exec(cand);
if (m) ips[m[1]] = true; // address field of the SDP candidate line
```
At minimum, document in the test that `""` may mean "mDNS-masked" rather than
"no leak surface," so the Phase 27 flip is written against the right baseline.

### WR-02: `cdpTell` probe is unreliable — `console.debug(e)` is not guaranteed to read the instance `stack` getter, so it can record `"no-signal"` even under CDP

**File:** `internal/detect/detect.js:100-117`
**Issue:** The probe defines a `stack` getter on a fresh `Error` instance and relies
on `console.debug(e)` triggering serialization that reads `.stack`. Under
`Runtime.enable`, CDP builds a *remote object preview* of the argument, but the
preview enumerates a bounded set of own properties and is **not guaranteed to invoke
accessor getters** for an instance-level `stack` (the classic tell historically
targets `Error.prototype.stack` / `Error.captureStackTrace` interception, or
`e.stack` being read during string coercion — not a preview of a defineProperty'd
instance getter). As written, the getter may never fire even when CDP is attached,
so the probe records `"no-signal"` — a false negative for the very condition the doc
(`docs/cdp-footprint.md:31`) says it surfaces.

It is explicitly informational/non-blocking (not asserted in the test), so this is a
WARNING, not a blocker — but a measurement probe that can silently report the wrong
truth undermines the "measure the exposure" goal of the phase.

**Fix:** Force a read of `.stack` directly rather than relying on console preview
semantics, e.g.:
```javascript
// Coerce the error to a string, which reads .stack and fires the getter
// under stack-trace-hooking detectors.
try { String(e.stack); } catch (_) {}
// or: ("" + e); / e.toString();
```
Or document the probe's known limitation and the Chrome versions where the
`console.debug` preview path actually fires the getter.

## Info

### IN-01: `permissionsConsistent` logic treats "prompt" the same as a healthy state and can't distinguish a real consistency check

**File:** `internal/detect/detect.js:159-160`
**Issue:** The verdict is
`notif === "denied" ? status.state === "denied" : status.state !== "denied"`.
For the common headless tell (`Notification.permission === "default"` while
`permissions.query` returns `"denied"`), this correctly yields `false`. But for
`notif === "default"` and `status.state === "prompt"` it yields `true`, and for
`notif === "granted"` / `status.state === "prompt"` it also yields `true` — i.e.
several genuinely inconsistent pairings are flagged "consistent." The single-axis
`!== "denied"` heuristic is coarse. For a baseline harness this is acceptable, but
the verdict name oversells what it checks.
**Fix:** Compare the two states for true equivalence (mapping `default`↔`prompt`)
or rename the signal to `permissionsNotDenied` to match what it actually asserts.

### IN-02: `Start()` discards the `http.Serve` return error

**File:** `internal/detect/server.go:56-58`
**Issue:** `go http.Serve(s.listener, s.mux)` ignores the returned error. On normal
`Close()` this returns a benign "use of closed network connection," but a genuine
serve failure (other than Close) would be silently swallowed, and the test would
fail later with a confusing connection-refused rather than the root cause.
**Fix:** Capture and stash the error on the struct (guarded by the mutex) or log it
when it is not the expected post-Close error, so a real failure is diagnosable:
```go
go func() {
    if err := http.Serve(s.listener, s.mux); err != nil {
        // store/log unless it's the expected ErrServerClosed-equivalent
    }
}()
```

### IN-03: `Scorecard.Signals` uses `interface{}` instead of `any`

**File:** `internal/detect/server.go:26`
**Issue:** Go 1.25 module (`go.mod`: `go 1.25.1`); `interface{}` should be `any` per
modern style. Low-priority lint nit as flagged in scope.
**Fix:** `Signals map[string]any \`json:"signals"\``.

### IN-04: `evalDetect` substring match on the result prefix is brittle if the evaluated value itself contains the prefix text

**File:** `tests/detection_test.go:53-55`
**Issue:** `strings.Index(out, evalResultPrefix)` finds the first occurrence of
"Evaluate code successfully with result:". For the signals under test (UA, WebGL
strings, timezone) this is safe, but a value that happened to embed that literal
string would be mis-sliced. Robustness nit only; current signals can't trigger it.
**Fix:** Anchor on the last occurrence (`strings.LastIndex`) or split on the first
newline of a known single-line result format.

### IN-05: `waitForDetectReady` worst-case wait (9s) is tight relative to probe timeouts and could flake under a slow/cold CI runner

**File:** `tests/detection_test.go:64-69`
**Issue:** The loop is 30 × 300ms = 9s. The JS global-ready fallback alone is 3s
(`detect.js:229`) and the WebRTC safety net is 1.5s, and each poll spawns a fresh
`rod-cli` process (process start + daemon round-trip). On a cold CI runner this
leaves limited headroom before the 9s budget, and the failure surfaces as a fatal.
It is not a logic bug — `ready` is set deterministically — but it is the most
likely flake vector in CI.
**Fix:** Raise the iteration count (e.g. 50 → ~15s) or back off the sleep; the cost
is only paid on the unhappy path since it returns immediately once ready.

---

_Reviewed: 2026-06-24_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
