---
author: engineer
responsible: engineer
phase: 30-cdp-footprint-reduction
plan: 01
type: execute-summary
status: complete
requirements: [CDP-01]
commit: 5ec045d
---

# Phase 30 Wave 1 — SUMMARY (CDP baseline reduction)

## Outcome
A plain session now enables **none** of {Runtime, Network, Fetch}. The three
always-on footprint sources in `types/context.go createPage()` were made
lazy/opt-in without regressing the v1.6 HTTP↔JS header-coherence invariant.

## WIRE-VERIFY verdict — Option C HOLDS (no fallback to B)
The one unproven assumption (does `Emulation.setUserAgentOverride` put the
spoofed `Sec-Ch-Ua` / `Accept-Language` on the OUTGOING wire WITHOUT
`Network.enable`?) was tested empirically by driving the shipped binary on a
plain `goto` against a loopback header-echo server and reading back the actual
request headers. Captured on the plain path (no capture flags, no routes):

```
User-Agent         = Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36
Accept-Language    = en-US,en;q=0.9;q=0.9
Sec-Ch-Ua          = "Not A(Brand";v="99", "Google Chrome";v="121", "Chromium";v="121"
Sec-Ch-Ua-Mobile   = ?0
Sec-Ch-Ua-Platform = "Windows"
X-Requested-With   = (absent)
```

All identity headers reach the wire via the Emulation override with **no**
`Network.enable`. **Option C is the implementation of record; Option B
(`NetworkSetUserAgentOverride` + `NetworkSetExtraHTTPHeaders`) was NOT needed.**
The existing `TestNetworkEvasionHeaders` (a plain `goto` asserting Sec-Ch-Ua /
UA / Accept-Language) stays green — it is the standing WIRE-VERIFY regression.

## What landed (per task)
- **Task 1 — Emulation identity.** `applyEmulationIdentity(page, prof)` calls
  `proto.EmulationSetUserAgentOverride{UserAgent, AcceptLanguage, Platform,
  UserAgentMetadata{Brands, Platform, Mobile:false}}`. Brands derived via
  `brandsForUA` (reuses `parseChromeMajor` + `defaultChromeMajor` "121"),
  Platform via `chPlatformFor` (Win32→Windows, MacIntel→macOS, else passthrough)
  — the exact tuple the old interceptor catch-all produced. UA-metadata attached
  only when `prof.SpoofClientHints` (mirrors prior gating). Log-and-continue on
  error. The catch-all identity rule was removed from `updateInterceptorRules`.
- **Task 2 — opt-in capture.** New `*bool` `ConsoleCapture` / `RequestCapture`
  on `StealthConfig` + `StealthFlags`, resolved in `ResolveStealth`
  (CLI>profile>default, default **false**). The two `EachEvent` subscriptions in
  createPage are gated behind `boolVal(..., false)` — when off they are never
  registered, so Runtime/Network are never enabled on a plain session.
- **Task 3 — lazy interceptor.** Removed the unconditional create+Enable from
  createPage. `ensureInterceptorEnabled(page)` helper lazily creates+Enables on
  the first `AddRoute`; `RemoveRoute` Disables+discards on the last route.
  `updateInterceptorRules` now carries only mock rules.
- **Task 4 — flags + docs.** `--console-capture` / `--request-capture` global
  flags (default off), forwarded into the daemon argv only on `IsSet`, captured
  into `StealthFlags` in main.go. `console`/`requests`/`request` Usage strings
  note the requirement. `docs/stealth-config.md` gained a "CDP footprint /
  capture" subsection (minimal-by-default, the two opt-in flags, precedence).

## Deviations from plan / code-review fixes
An `anvil-code-reviewer` gate on the diff found two MAJOR lifecycle holes from
moving interceptor creation out of createPage; both were fixed before commit:
1. **route-before-goto** — a `route` issued before the first `goto` (ctx.page
   nil) was stored but inert. Fix: createPage re-establishes the interceptor
   from `ctx.routes` when the page is created (`ensureInterceptorEnabled(page)`
   takes the page explicitly because `ctx.page` is assigned only after createPage
   returns). Added `TestRouteBeforeFirstGoto` regression.
2. **stale interceptor on page recreate** — closePage/closeBrowser never
   Disabled/nil'd the interceptor, leaking the router goroutine and binding a new
   page's routes to the closed page. Fix: `closePage` now Disables+nils it
   (routes preserved so createPage rebinds them to the fresh page). Note: in the
   daemon `close` exits the process, so this path is mainly latent/test-exercised
   today, but the fix is correct for the exported API.

Minors: X-Requested-With strip intentionally dropped (Chrome never emits it —
confirmed absent above; documented in code + cdp-footprint to follow in 30-03).
stateLock held across Enable/Disable accepted as pre-existing (no deadlock —
godoll's router uses its own mutex).

## Integration risks confirmed handled
- console/requests commands still work WHEN capture is on (TestNetworkAndDevCommands
  updated to spawn with the flags; green).
- mock routes work via lazy interceptor (TestRoutes, TestNetworkAndDevCommands,
  TestRouteBeforeFirstGoto green).
- Plugin OnRequest/OnResponse hooks unaffected — they enable their own Network/DOM
  subscription in `internal/plugin/lifecycle.go`, independent of the baseline path.

## Tests
- `go build ./...`, `go vet ./...` clean.
- WIRE-VERIFY: TestNetworkEvasionHeaders PASS; direct header capture confirms C.
- types: interceptor + close tests PASS. actions: route + network-request +
  console PASS. tests/: detection/stealth/network coherence suite PASS.

## Note for 30-02
The Fetch-enable site for the wave-2 ledger is `ensureInterceptorEnabled` (set
Fetch=true after a successful `Enable()`); Runtime/Network ledger points are the
two gated `EachEvent` branches in createPage.
