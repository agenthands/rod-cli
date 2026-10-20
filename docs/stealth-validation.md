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

**What it is:** `tests/detection_test.go` (with `tests/stealth_test.go` for
`navigator.webdriver`, plugins, and the UA) driving the **shipped `rod-cli` binary**
— the prebuilt `../rod-cli`, so run `go build -o rod-cli .` first — against the
offline fixture in `internal/detect/` (served on `127.0.0.1`, zero network egress).
It is the blocking gate in `.github/workflows/test.yml`, which builds the binary,
runs `./rod-cli install`, then `go test ./... -count=1`.

Every `tests/` run (both tiers) keeps its sessions' browser profiles in a fresh
`rod-cli-tests-cache-*` dir in `os.TempDir()` (`TestMain` points `XDG_CACHE_HOME`
at it) and removes it when the run ends; a run killed before then has its dir
removed by a later run once none of its daemons is left.

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
- Fingerprint dimensions (Phase 33): media devices, battery, and codecs read back
  plausible values; media devices and fonts read back the same within a session and
  revert when their toggle is off (`TestAdvancedEvasionDimensions`,
  `TestFontSpoof`). The font set is drawn for the profile's OS and the seeded
  generator reproduces the same dimensions for the same seed; this is checked
  without a browser at the generation layer (`TestSeededFingerprintDimensions`).

The CDP baseline is checked offline too: `TestCDPFootprintBaseline`
(`types/cdp_footprint_test.go`) reads rod-cli's per-session CDP domain ledger and
asserts a plain session enables none of `Runtime` / `Network` / `Fetch`.

**Why this tier is blocking:** it is fully offline and deterministic, so it can be a
CI gate without flakiness. It proves the layer rod-cli *owns* — the JS/fingerprint
surface — and nothing it cannot.

---

## Tier 2 — Live smoke check (opt-in, best-effort, NON-blocking)

**What it is:** `tests/detection_live_test.go`, guarded by
`//go:build detection_live`. It drives the real binary against **live third-party
anti-bot challenges** — Cloudflare, DataDome — the **CreepJS** fingerprint
scorer, and a CDP-tell probe on `example.com`, reporting each per-target verdict
**informationally**.

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
  (cipher suites, extensions, ordering). That handshake is the browser's, not
  something JavaScript can rewrite. **rod-cli takes this as a strength, not a gap:
  it drives a real Chrome/Chromium browser, so the TLS handshake is an *authentic
  Chrome* JA3/JA4 by construction — and rod-cli deliberately does NOT spoof it.**
  All shipped device profiles are Chrome variants (they vary the JS / header
  identity, never the TLS layer), so the TLS fingerprint and the spoofed UA always
  tell the same Chrome story — there is no UA-vs-TLS mismatch to close. A uTLS-style
  TLS-spoofing layer was evaluated for v1.7 and **deliberately rejected** (it adds a
  large network-layer surface and risk for no gain when the browser is already real
  Chrome). The residual caveat is version skew: if a profile claims a Chrome major
  far from the actual binary's, the TLS reflects the *real* binary — so keep the
  binary current.
- **IP reputation.** Datacenter / proxy / known-bad ASNs are scored at the network
  layer, independent of any browser signal. rod-cli does not launder your egress IP;
  a flagged IP is detected regardless of a perfect fingerprint.
- **CDP transport footprint.** rod-cli drives Chrome over the DevTools Protocol, and
  the CDP transport is observable independently of JS-property spoofing. As of v1.7
  the always-on footprint is **reduced**: a plain session enables **none** of
  `Runtime` / `Network` / `Fetch` — console/request logging are opt-in flags, the
  mock interceptor is lazy, and HTTP identity coherence moved to the zero-enable
  `Emulation` domain. `TestCDPFootprintBaseline` asserts this baseline
  deterministically (Tier 1). What remains is the irreducible `Page`/`Target`
  enablement needed to drive any CDP browser. The opt-in `--cdp-proxy` (v2.0,
  CDP-DEEP-01) strips getter values from `Runtime.getProperties` responses and,
  with `--cdp-jitter-ms`, jitters command timing, but the CDP transport itself
  stays observable. See
  [CDP Footprint](cdp-footprint.md) for the per-domain inventory and mitigation status.

**No "undetectable" guarantee.** rod-cli validates and hardens the JS/fingerprint
layer and is transparent about the TLS, IP, and CDP layers it does not. Use it with
that ceiling in mind: passing Cloudflare on one run, on one IP, at one moment is a
best-effort data point — not proof you are invisible.

---

## See also

- [CDP Footprint](cdp-footprint.md) — the per-domain CDP inventory + mitigation
  status: how the baseline Runtime/Network/Fetch footprint was reduced (v1.7), the
  opt-in CDP proxy (v2.0, CDP-DEEP-01), and the honest ceiling.
- [Stealth Configuration](stealth-config.md) — the config surface, fingerprint pins,
  proxy, hardening toggles, and humanize tuning that the harness validates.
- [Architecture](../ARCHITECTURE.md) — the daemon + stealth engine overview.
