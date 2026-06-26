---
author: architect-lead
---

# Phase 30: CDP Footprint Reduction - Context

**Gathered:** 2026-06-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Reduce/obfuscate the detectable Chrome DevTools Protocol (CDP) signals rod-cli
emits — chiefly the `Runtime.enable` family — by making the features that force
domain enablement opt-in rather than always-on; inventory **every** CDP command
rod-cli sends with a per-command mitigation status; and **measure** the reduction
deterministically (offline harness) and informationally (live targets), capped by
an honestly-documented ceiling.

**In scope (CDP-01/02/03):** baseline CDP-surface reduction, command inventory,
harness assertion of the reduced baseline, honest-ceiling documentation.

**Out of scope (own phases):** TLS/JA3 spoofing (Phase 31), profile library
(Phase 32), new JS-layer hardening surfaces (Phase 33). Deeper CDP obfuscation
requiring browser patching or a MITM/alternate transport is **deferred to v2
(CDP-DEEP-01)** — name it as the ceiling, do not attempt it here.

</domain>

<decisions>
## Implementation Decisions

### CDP reduction strategy — "footprint follows features"
- **D-01:** Drive the **baseline CDP surface to the minimum**. The three
  unconditional footprint sources in `types/context.go` `createPage()` become
  **lazy / opt-in**:
  - `Runtime.consoleAPICalled` subscription (currently `context.go:558`) — the
    thing that forces **`Runtime.enable`** today.
  - `Network.requestWillBeSent` subscription (currently `context.go:570`) — forces
    `Network.enable`.
  - The godoll network **interceptor** `.Enable()` (currently `context.go:577-580`)
    — forces `Fetch.enable`.
  A plain session (e.g. `open`/`goto` + `snapshot`) must trigger **no
  `Runtime.enable` and no `Fetch.enable`** — only what go-rod structurally
  requires (e.g. `Page.enable`).
- **D-02:** **Nothing stays always-on.** No capture (not even console-errors)
  remains unconditional — there must be **no `Runtime.enable` on a plain
  session**. Debugging is one flag away. (This also trims unsolicited output,
  consistent with the token-efficiency ethos — see [[output-noise-and-autonomy]].)
- **D-03:** **Features auto-enable the domain they need, on demand.** Footprint
  follows the feature: loading a plugin with `OnRequest`/`OnResponse` hooks
  auto-enables the interceptor (Fetch/Network); a console-capture flag enables
  `Runtime.consoleAPICalled`; a request-logging flag enables Network. The user
  never has to think about CDP domains — they ask for a *feature* and pay its
  footprint only then.

### Measurement / harness assertion (CDP-03)
- **D-04:** The deterministic assertion is an **instrumented CDP-command / enabled-
  domain count per session**. Instrument the daemon to record which CDP domains
  get enabled (and/or count commands) for a session; the **Tier-1 offline harness
  asserts a baseline session stays at/below threshold** — concretely, a plain
  `goto` enables **zero** of {Runtime, Fetch}. This is fully deterministic and
  offline, matching the Phase 24 harness ethos (read truth, gate on it).
- **D-05 (complementary, not the gate):** The harness **already exposes an
  informational JS leak probe** `window.__detect.cdpTell` (added Phase 24,
  alongside `window.__detect.webrtcIce`). Keep it as the realism signal; the
  command-count instrumentation (D-04) is the deterministic gate. We did **not**
  choose to make the JS leak probe the assertion (go-rod behavior can make it pass
  trivially and offline-determinism is harder).
- **D-06:** **Live-target measurement (CDP-03) reuses the existing Phase 29
  Tier-2 suite** (`tests/detection_live_test.go`, `//go:build detection_live`,
  non-blocking, informational). Do **not** add it to the CI gate — it stays
  best-effort. (See [[anvil-velocity-provenance-bug]] is unrelated; the relevant
  prior art is Phase 29's CONTEXT.)

### Configurability & default
- **D-07:** **Minimal-by-default, features opt-in.** **No dedicated `--cdp-stealth`
  flag.** Minimal footprint is just correct default behavior. The footprint-adding
  features (console logging, request logging, interceptor) get their **own opt-in
  flags**, each following the v1.6 precedence chain **CLI > profile > default**
  (see `docs/stealth-config.md`). One coherent surface; no overlapping master
  switch.

### Inventory documentation (CDP-02)
- **D-08:** **Expand `docs/cdp-footprint.md`** (it already exists as a Phase 24
  findings note — this is an *expand/finalize*, not a create). Deliver:
  - A **per-command / per-domain table**: command/domain → when it is sent →
    status one of `reduced` / `obfuscated` / `accepted-visible`.
  - An explicit **honest-ceiling section** naming what cannot be fixed without
    browser-patching / MITM / alternate transport → **defers to v2 CDP-DEEP-01**.
  - **Cross-links** to `docs/stealth-validation.md` (Tier-1 offline blocking vs
    Tier-2 live opt-in framing). No "undetectable" claim.

### Header coherence vs Fetch.enable — RESOLVED DESIGN FORK (added during planning)
- **D-09:** During planning the architect found the network interceptor is **not a
  pure opt-in feature** — its always-on catch-all rule (`updateInterceptorRules`,
  `URLPattern:"*"`, `network.Continue`) rewrites identity headers (UA,
  Accept-Language, Sec-Ch-Ua client hints) on every request to match the JS
  identity. That is the **v1.6 FINGERPRINT-01/02 triple-agreement invariant**, so
  naively making the interceptor opt-in (per D-01) would remove `Fetch.enable`
  **but regress HTTP↔JS header coherence**. Surfaced via `anvil-design-options`;
  **co-driver chose Option C+D (2026-06-26).**
- **Resolution (C+D hybrid):**
  - **Plain path:** set identity via `proto.EmulationSetUserAgentOverride{UserAgent,
    AcceptLanguage, Platform, UserAgentMetadata{Brands, FullVersionList, Platform,
    Mobile, ...}}` at page setup, built from the active `stealth.Profile` (reuse
    `parseChromeMajor` for the brand version — ONE derivation path). The **Emulation
    domain has no `enable` command**, so this enables **zero CDP domains** while
    Chrome natively emits coherent UA / `Sec-Ch-Ua*` / `navigator.userAgentData` —
    coherence becomes *stronger* than today's two-mechanism scheme.
  - **Lazy Fetch:** the interceptor is created + `.Enable()`d (→ `Fetch.enable`)
    **only** when a mock route is added (`AddRoute`) or a strict-header mode is
    opted in; disabled when the last route is removed. The catch-all `Continue`
    identity rule is **removed** (Emulation now carries identity).
  - **Fallback (Option B):** IF harness capture shows `Emulation.setUserAgentOverride`
    does NOT put `Sec-Ch-Ua`/`Accept-Language` on the **outgoing wire** without
    `Network.enable`, fall back to `proto.NetworkSetUserAgentOverride` +
    `proto.NetworkSetExtraHTTPHeaders` (accepts `Network.enable`, guarantees headers).
    **The engineer MUST verify wire propagation on the Phase 24/29 harness before
    committing**, and document whichever lands in `docs/cdp-footprint.md`.
  - Known minor gap to check: `X-Requested-With` deletion has no Fetch path on the
    plain page — verify Chrome doesn't emit it in this config, else note it.

### Claude's Discretion
- Exact flag names for the now-opt-in features (console capture, request logging,
  interceptor) — within the CLI>profile>default convention.
- The precise instrumentation shape for D-04 (where the per-session domain/command
  ledger lives; how the harness reads it) and the exact threshold encoding.
- Whether the lazy interceptor is gated by a single "needs-network" predicate or
  per-feature — engineer's call, as long as a plain session enables neither
  Runtime nor Fetch.
- Exact table columns/rows in `cdp-footprint.md` beyond the required
  command→when→status shape.

</decisions>

<disciplines>
## Lenses surfaced for this phase

> **Lenses, not rules.** Each entry below is a perspective the architect surfaced for this phase. Downstream agents engage with these lenses: adopting, refining, or reasoning past them. Divergence-with-reasoning is a successful interaction.

~~~yaml
disciplines:
  - section: "inline"
    name: "earn-accepted-visible"
    scope: ["docs/cdp-footprint.md", "the per-command mitigation table (CDP-02)"]
    guidance: "When classifying a CDP command's status, do not default to
      'accepted-visible' as a convenience. Treat each command as a real detection
      tell that deserves a reduction or obfuscation attempt first; only mark it
      'accepted-visible' once you can state WHY reduction is infeasible or
      pointless."
    rationale: "The whole phase exists because the Phase 24 note rationalized the
      footprint as an unavoidable ceiling ('Answer: No'). The risk now is
      rubber-stamping every remaining command as 'accepted-visible' and calling it
      done — that would re-create the very gap this phase closes."
    override_signals:
      - "The command is provably indistinguishable from traffic a normal,
        non-automated Chrome sends anyway (e.g. a domain Chrome always enables) —
        then 'accepted-visible' is the honest classification, not a dodge."
      - "Reduction would require browser-patching/MITM, which is explicitly
        deferred to v2 (CDP-DEEP-01) — mark 'accepted-visible' and point at the
        v2 ceiling."

  - section: "inline"
    name: "lazy-enable-but-not-too-late"
    scope: ["types/context.go createPage()", "the opt-in feature wiring (D-01..D-03)"]
    guidance: "Prefer enabling a CDP domain lazily — only when the feature that
      needs it is actually requested — rather than eagerly at page creation."
    rationale: "Eager enablement at createPage() is exactly what produces the
      always-on Runtime/Fetch footprint this phase removes."
    override_signals:
      - "A feature needs events emitted from the very first navigation (e.g.
        request logging must catch the initial document request). Lazily enabling
        Network AFTER navigation would silently miss those events — so once that
        feature is opted in, enable its domain EAGERLY (before navigation) for
        that session. Lazy across sessions, eager within an opted-in one."
~~~

</disciplines>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & roadmap
- `.planning/REQUIREMENTS.md` — CDP-01 (reduce/obfuscate Runtime.enable, measured, honest ceiling), CDP-02 (inventory all CDP commands w/ mitigation status), CDP-03 (measure vs live targets, harness asserts baseline). Also the v2-deferred `CDP-DEEP-01` (full obfuscation via browser-patching/MITM) and the "no live-WAF guarantee" Out-of-Scope row.
- `.planning/ROADMAP.md` §"Phase 30: CDP Footprint Reduction" — goal + 3 success criteria.

### The footprint sources (what to make lazy)
- `types/context.go` `createPage()` — the always-on subscriptions to make opt-in:
  - `Runtime.consoleAPICalled` (~line 558) → forces `Runtime.enable`.
  - `Network.requestWillBeSent` (~line 570) → forces `Network.enable`.
  - network interceptor `.Enable()` (~lines 577-580) → forces `Fetch.enable`.
  - `updateInterceptorRules()` / `GetConsoleLogs()` consumers (~603-690) — what depends on these domains.

### Measurement infrastructure
- `internal/detect/` — the offline Tier-1 harness (`server.go`, `embed.go`, `detect.html`, `detect.js`, `probe.js`). The informational CDP-tell probe is surfaced as `window.__detect.cdpTell`. `probe.js` is the single source of truth shared by harness + `stealth-check`.
- `internal/detect/server_test.go` — harness test pattern to extend with the D-04 baseline assertion.
- `tests/detection_live_test.go` — Phase 29 Tier-2 live suite (`//go:build detection_live`, non-blocking) to reuse for CDP-03 live measurement.

### Documentation deliverable + ethos
- `docs/cdp-footprint.md` — **already exists** as a Phase 24 findings note; CDP-02 expands it into the per-command table + honest-ceiling. Read it first — it already documents WHY the footprint exists and names the opt-in fix.
- `docs/stealth-validation.md` — Tier-1 (offline, blocking) vs Tier-2 (live, opt-in, non-blocking) honesty framing; cross-link target. No-overclaim ethos.
- `docs/stealth-config.md` — the v1.6 config surface + CLI>profile>default precedence chain that any new opt-in feature flags must follow.
- `.planning/phases/29-best-effort-live-validation/29-CONTEXT.md` — the non-blocking live-suite discipline and honest-ceiling precedent.

### go-rod CDP mechanism (research/grounding)
- go-rod v0.116.2 `states.go` (`EnableDomain`), `page_eval.go` (`getJSCtxID` uses `Runtime.evaluate{window}`, no global `Runtime.enable`), `browser.go` (`EachEvent`/`WaitEvent` lazily enables a domain when an event type is subscribed). Confirms: domain enablement is driven by event subscriptions, so removing the subscriptions removes the enablement.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`internal/detect/` harness + `probe.js`** — already has an informational
  `window.__detect.cdpTell` CDP probe; extend with the D-04 command-count
  assertion rather than building new measurement from scratch.
- **`tests/detection_live_test.go`** — Phase 29's non-blocking Tier-2 harness;
  reuse for the CDP-03 live measurement (do not add to CI gate).
- **`docs/cdp-footprint.md`** — Phase 24 stub to expand, not a blank create.
- **v1.6 stealth-config precedence resolver** (`types/config.go` `ResolveStealth`,
  `docs/stealth-config.md`) — the CLI>profile>default machinery any new opt-in
  flag plugs into.

### Established Patterns
- `createPage()` wires browser/page lifecycle and currently enables CDP domains
  **eagerly + unconditionally** — the pattern this phase inverts to lazy/on-demand.
- `EachEvent(...)` subscriptions are what *cause* domain enablement in go-rod;
  the lever is which subscriptions run, and when.
- Stealth resolved **once** at daemon spawn (`ResolveStealth`) — opt-in feature
  state should resolve through the same single path, not ad hoc per command.

### Integration Points
- The daemon spawn argv / `types.Context` build is where opt-in feature flags must
  thread through (same place v1.6 stealth flags flow).
- Plugin engine (`internal/plugin/`) hooks (`OnRequest`/`OnResponse`) are a
  feature that must **auto-enable** the interceptor (D-03) — the dependency the
  planner must wire so loading such a plugin still works.

</code_context>

<specifics>
## Specific Ideas

- The headline, falsifiable win: **a plain `goto` session enables neither
  `Runtime` nor `Fetch`** — asserted by the harness (D-04). This is the single
  sentence the phase should be judged against.
- The existing `docs/cdp-footprint.md` literally predicts this phase: *"opt-in
  logging so Runtime/Network domains are only enabled on demand … deferred to v2
  (requirement CDP-01)."* Phase 30 makes that real and re-scopes the ceiling to
  CDP-DEEP-01.

</specifics>

<deferred>
## Deferred Ideas

- **Full CDP protocol obfuscation** (browser-patching / MITM / alternate
  transport) → **v2 CDP-DEEP-01**. Name it as the honest ceiling in
  `cdp-footprint.md`; do not attempt in Phase 30.
- **JS leak probe as a blocking gate** — kept informational (`__detect.cdpTell`);
  the deterministic gate is the command-count instrumentation. Revisit only if the
  instrumentation proves insufficient.
- **Live-WAF pass/fail guarantee** — explicitly Out of Scope (non-deterministic);
  CDP-03 live measurement stays informational/non-blocking.

None of the above are in-scope for Phase 30.

</deferred>

---

*Phase: 30-cdp-footprint-reduction*
*Context gathered: 2026-06-26*
