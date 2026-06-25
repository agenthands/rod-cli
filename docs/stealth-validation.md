# Stealth Validation — What rod-cli Proves, and What It Cannot (LIVEWAF-01)

**TL;DR — rod-cli does NOT claim to be "undetectable."** It validates the stealth
layer it actually controls (the JavaScript / fingerprint surface), deterministically
and offline, and is honest about the layers it cannot control (TLS fingerprint, IP
reputation, and the CDP transport). A passing check — offline or live — is **not** a
guarantee that any given anti-bot system will let you through.

rod-cli's stealth is validated in **two tiers**. The split is deliberate: one tier
is a deterministic, blocking proof of the layer rod-cli controls; the other is a
best-effort, non-blocking smoke signal against the real world, which is flaky by
nature and subject to layers a JS-injecting CLI cannot touch.

---

## Tier 1 — Offline harness (deterministic, blocking CI)

**What it is:** `tests/detection_test.go` driving the **shipped `rod-cli` binary**
against the offline fixture in `internal/detect/` (served on `127.0.0.1`, zero
network egress). It is the blocking gate in `.github/workflows/test.yml`
(`go test ./... -count=1`).

**What it deterministically proves** — the **JavaScript-layer fingerprint signals**,
each read back **from the live page** (via the `eval` command, never from a Go
config field — the project's *validate-live-not-source* rule):

- `navigator.userAgent`, Client-Hints (`Sec-CH-UA`), and `navigator.userAgentData`
  tell one coherent Chrome-version story.
- `navigator.platform` + `Sec-CH-UA-Platform` + the UA OS token tell one coherent
  OS story.
- WebGL vendor/renderer is not a software rasterizer (no SwiftShader/llvmpipe tell).
- `navigator.webdriver`, `window.chrome`, plugins, `navigator.languages`, screen
  geometry, timezone, and permissions consistency carry no headless tell.
- WebRTC ICE leaks no real host IP (Phase 27 HARDEN-01).
- Canvas / WebGL / audio readback noise is **stable within a session** (Phase 27
  HARDEN-02).

**Why this tier is blocking:** it is fully offline and deterministic, so it can be a
CI gate without flakiness. It proves the layer rod-cli *owns* — the JS/fingerprint
surface — and nothing it cannot.

---

## Tier 2 — Live smoke check (opt-in, best-effort, NON-blocking)

**What it is:** `tests/detection_live_test.go`, guarded by
`//go:build detection_live`. It drives the real binary against **live third-party
anti-bot challenges** — Cloudflare, DataDome — and the **CreepJS** fingerprint
scorer, reporting each per-target verdict **informationally**.

**It is non-blocking in BOTH senses:**

1. **Excluded from CI by construction.** The build tag means a plain
   `go build ./...` / `go test ./...` — and the CI gate, which runs
   `go test ./... -count=1` with **no `-tags`** — never compiles or runs it. There
   is an explanatory comment in `test.yml` so this is not "helpfully" re-enabled.
2. **Informational within the suite.** Every outcome is logged with `t.Logf`; an
   unreachable target or an environment with no egress is `t.Skip`-ped. The suite
   **never** fails (`t.Fatal`/`t.Errorf`) on a live detection or a network error —
   a flaky third-party challenge must not produce a red build. **A live "green" is
   not the bar**, and a "no challenge observed" result is *not* a pass guarantee.

### How to run it

```sh
go build -o rod-cli .          # the suite execs the prebuilt ../rod-cli
go test -tags detection_live ./tests/ -run TestLiveDetection -v
```

Outcomes are informational. The target URLs are third-party and may change,
rate-limit, or disappear at any time — treat unreachability as "best-effort skip,"
never as a defect.

---

## The honest ceiling — layers a JS-injecting CLI CANNOT control

rod-cli operates by injecting JavaScript and tuning the browser launch. That layer
is real and worth validating (Tier 1), but it is **not the whole detection
surface**. The following layers live *below* or *outside* JS injection, and rod-cli
makes **no claim** to control them — which is exactly why even a clean live run does
**not** mean "undetectable":

- **TLS fingerprint (JA3 / JA4).** Anti-bot systems fingerprint the TLS ClientHello
  (cipher suites, extensions, ordering). That handshake is the browser's / Go
  runtime's, not something JavaScript can rewrite. A mismatch between the spoofed UA
  and the real TLS stack is a tell rod-cli cannot close.
- **IP reputation.** Datacenter / proxy / known-bad ASNs are scored at the network
  layer, independent of any browser signal. rod-cli does not launder your egress IP;
  a flagged IP is detected regardless of a perfect fingerprint.
- **CDP transport footprint.** The daemon relies on the Chrome DevTools Protocol
  (`Runtime.enable` / `Network.enable`) for console and request logging, and that
  domain-enablement is observable independently of JS-property spoofing. See
  [CDP Footprint](cdp-footprint.md) for the full analysis — reducing this footprint
  is deferred to v2 (CDP-01), not attempted in v1.6.

**No "undetectable" guarantee.** rod-cli validates and hardens the JS/fingerprint
layer and is transparent about the TLS, IP, and CDP layers it does not. Use it with
that ceiling in mind: passing Cloudflare on one run, on one IP, at one moment is a
best-effort data point — not proof you are invisible.

---

## See also

- [CDP Footprint](cdp-footprint.md) — why the CDP transport tell cannot be closed in
  the JS layer (one of the uncontrollable layers above).
- [Architecture](../ARCHITECTURE.md) — the daemon + stealth engine overview.
