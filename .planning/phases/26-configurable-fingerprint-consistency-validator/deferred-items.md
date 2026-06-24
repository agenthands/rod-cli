# Phase 26 — Deferred / Out-of-Scope Items

Items discovered during Plan 26-05 execution that are outside this plan's declared
scope (`tests/detection_test.go` only). Logged per the executor scope-boundary rule;
NOT fixed here.

## BLOCKING coherence gap (Plan 03 regression) — Client-Hints OFF by default

**Discovered:** Plan 26-05, Task 4 (full-suite regression run).

**Symptom:** `tests/network_evasion_test.go::TestNetworkEvasionHeaders` FAILS — the
default identity (no `--profile`, no flags) navigates and the response request
headers contain **no `Sec-Ch-Ua`** (only User-Agent / Accept-Language). The test
expects `Sec-Ch-Ua` to be present.

**Root cause:** `stealth.DefaultProfile()` ships `SpoofClientHints: false`
(`../godoll/stealth/profile.go:79`). Plan 03's `updateInterceptorRules`
(`types/context.go:576`) gates the entire `Sec-Ch-Ua` / `Sec-Ch-Ua-Mobile` /
`Sec-Ch-Ua-Platform` injection behind `if prof.SpoofClientHints`. So on the
default identity the header is never emitted. On the default identity godoll's
`navigator.userAgentData.brands` is also empty (`[]`) for the same reason. Both
Client-Hints surfaces are therefore EMPTY (not coherent) unless the user opts in.

There is also **no `--spoof-client-hints` CLI flag**; the field is only settable
via a `--profile` JSON or config file (`types/config.go:93`, `:203`). So a user who
pins `--user-agent ...Chrome/130...` gets a UA that is NOT backed by matching
Client-Hints unless they ALSO supply a profile enabling `spoofClientHints`.

**Why this is the phase's coherence gap:** The Phase-26 intent (26-CONTEXT.md) is
"the hardcoded Client-Hints `121` literal is eliminated ... so `Sec-CH-UA`,
`navigator.userAgentData`, and the UA all tell one OS+version story." The wiring
WORKS when activated (proven live in Plan 05's `consistency_invariant` /
`pinned_identity_macos` subtests, which enable it via a profile), but it is
**opt-in, not the default**, and the `--user-agent` anchor does not auto-derive it.

**Plan 05 handling (per Task 4 "report it, do not paper over it"):** The new
blocking subtests ACTIVATE Client-Hints (temp `--profile` with
`spoofClientHints:true`) so they prove real cross-surface coherence rather than
passing vacuously on empty CH. The default-off gap is reported here and in the
26-05 SUMMARY for the phase verifier.

**Suggested fix (out of Plan 05 scope — touches production code, not
`detection_test.go`):** Either (a) default `SpoofClientHints` to `true` in the
resolved active profile / `DefaultProfile()`, or (b) auto-enable
`SpoofClientHints` whenever a UA is pinned (the CONTEXT "derive-when-unset"
policy), plus add a `--spoof-client-hints` (or `--no-client-hints`) flag. This is
a `types/context.go` / godoll change and should be a follow-up plan or a Phase-26
verifier remediation.

## Pre-existing / unrelated

- `daemon/daemon_more_test.go::TestStartServerWithPpid` — FAILED ("server with
  ppid did not become ready") under the heavily-loaded full `go test ./...` run
  (~217s in the tests package alone). This is a daemon-readiness/ppid test with no
  relation to stealth/fingerprint or to `detection_test.go`. The file also carried
  an UNCOMMITTED pre-session signature edit (adding a 4th `nil` arg to
  `EnsureDaemon`) that is not part of Plan 05. Treated as pre-existing/flaky and
  left untouched.
