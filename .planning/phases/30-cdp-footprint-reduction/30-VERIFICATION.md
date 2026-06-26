# Phase 30: CDP Footprint Reduction — Verification

**Verdict:** ✅ PASSED
**Verified:** 2026-06-26 (independent qa, goal-backward against HEAD)
**Requirements:** CDP-01, CDP-02, CDP-03 — all delivered in code.

## Success criteria (ROADMAP Phase 30)

1. **Harness asserts reduced CDP visibility vs baseline** — ✅ `TestCDPFootprintBaseline`
   (`types/cdp_footprint_test.go`) asserts a plain `goto` enables ZERO of
   {Runtime, Network, Fetch}; three positive controls (`--console-capture`→Runtime,
   `--request-capture`→Network, mock route→Fetch) prove the gate is falsifiable, not
   vacuous. All pass on real headless Chrome.
2. **All CDP commands inventoried with mitigation status** — ✅ `docs/cdp-footprint.md`
   per-domain table (Runtime/Network/Fetch=reduced, Emulation=zero-enable, DOM=reduced,
   Page/Target=accepted-visible with earned justification); true against HEAD, no overclaim.
3. **Honest ceiling documented + live measurement** — ✅ Honest-ceiling section names
   v2 CDP-DEEP-01; informational non-blocking CDP observation in the `detection_live`
   (Tier-2) suite, excluded from the CI gate.

## Key code facts verified
- Lazy/opt-in wiring in `types/context.go createPage()`: console (`:661-674`),
  request (`:679-686`), lazy interceptor via `ensureInterceptorEnabled` (`:791-819`).
- Header coherence via `applyEmulationIdentity` (`:590-606`) — `proto.EmulationSetUserAgentOverride`,
  zero CDP footprint; **WIRE-VERIFY: Option C held** (Sec-Ch-Ua/Accept-Language reach the
  wire without Network.enable); `TestNetworkEvasionHeaders` is the standing regression.
- Per-session ledger `ctx.cdpDomains` + `GetEnabledCDPDomains`.

## Evidence
`go build ./...`, `go vet ./...`, `go build -tags detection_live ./...` all clean.
`TestCDPFootprintBaseline` + 3 controls PASS; `TestNetworkEvasionHeaders` PASS.

## Routed UP (no action this phase)
- Plugin lifecycle binder (`internal/plugin/lifecycle.go:31,34`) enables Network/DOM
  WITHOUT the ledger — reachable only via `actions.PluginLoad`, so the plain baseline is
  unaffected. Honestly disclosed in the SUMMARY + doc. v2 candidate: an SMTC dominance
  spec (or routing the binder through the ledger) to make the invariant mechanically
  re-checked.
