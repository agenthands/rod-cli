# Stealth Configuration Surface (v1.6)

This is the contributor-facing reference for `rod-cli`'s configurable stealth
surface: the config spine, the flag catalog, the per-session proxy, the
fingerprint consistency model, the anti-fingerprint hardening toggles, and the
human-behavior tuning knobs — including the honest constraints on what each can
and cannot do.

For the system-level picture see [../ARCHITECTURE.md](../ARCHITECTURE.md); for
what the validation actually proves see
[stealth-validation.md](stealth-validation.md).

---

## 1. The spine: one resolver, one identity

Every stealth knob flows through a single funnel so they all share the same
precedence and the same validation seam:

| Stage | Code | Role |
|---|---|---|
| Client flag capture | `cmd.go` `runClientCommand` | forwards non-secret flags into the daemon spawn argv; `--proxy-auth` goes out-of-band via `ROD_CLI_PROXY_AUTH` |
| Daemon flag capture | `main.go` `runDaemonServer` | reads the flags off its `cli.Context` into a `types.StealthFlags` |
| Resolution | `types/config.go` `ResolveStealth(cfg, flags)` | applies precedence, derives + validates, freezes into `cfg.Stealth` |
| Identity build | `types/context.go` `profileFromStealth` | turns the resolved `cfg.Stealth` into the active `stealth.Profile` (the single identity source) |
| Apply | `types/context.go` `createPage` | `EvasionManager.SetProfile` + `Apply` + `EvadeWebRTC` per page |

**Resolved once, at spawn.** `ResolveStealth` runs before `NewContext` freezes
`Config`. A stealth flag therefore only takes effect for a session if it is
present when that session's daemon spawns. Passing a stealth flag to an
already-running session prints a stderr warning (`stealth flags apply at session
spawn — run close first to re-apply`) and is otherwise inert.

**Precedence: CLI flag > profile file > built-in default** (`ResolveStealth`):

- *Tier 3* — `stealth.DefaultProfile()` (godoll), plus the hardened defaults
  rod-cli adds (see §4).
- *Tier 2* — a `--profile` JSON file (`stealth.LoadProfile`), overlaid onto
  `cfg.Stealth`. A missing/malformed profile is a **loud** failure: the daemon
  aborts rather than silently shipping a default identity.
- *Tier 1* — explicit CLI flags win over both.

---

## 2. Flag catalog

All flags are global (defined in `cmd.go`) and forwarded at daemon spawn.

### Proxy (Phase 25)

| Flag | Meaning |
|---|---|
| `--proxy` | egress proxy URL with scheme: `http://host:port`, `https://…`, `socks5://…`, `socks4://…` |
| `--proxy-auth` | credentials as `user:pass`; handled via CDP, **never** URL-embedded |

### Fingerprint identity pins (Phase 26)

| Flag | Maps to | Notes |
|---|---|---|
| `--user-agent` | `navigator.userAgent` / HTTP UA | the **derivation anchor** (drives Client-Hints + platform) |
| `--locale` | BCP-47 locale | derived from `languages[0]` when unset |
| `--timezone` | IANA timezone ID | e.g. `America/New_York` |
| `--platform` | `navigator.platform` | auto-derived from the UA OS token when unset |
| `--profile` | a saved `stealth.Profile` JSON (name or path) | name resolves under `~/.rod-cli/profiles/<name>.json` |

### Hardening toggles (Phase 27, default ON)

| Flag | Meaning |
|---|---|
| `--webrtc-protection` | prevent WebRTC local-IP leaks; `--webrtc-protection=false` to disable |
| `--canvas-noise` | stable-per-session canvas/WebGL/audio noise; `--canvas-noise=false` to disable |

### Humanize tuning (Phase 28, all opt-in)

| Flag | godoll option | Notes |
|---|---|---|
| `--typing-speed-min` / `--typing-speed-max` | `WithTypingSpeed(min,max)` | **both** required; the spread *is* the "delay jitter" |
| `--typo-rate` | `WithTypoRate` | probability 0.0–1.0 per keystroke |
| `--mouse-tremor` | `WithMouseTremor` | `--mouse-tremor=false` to disable |
| `--mouse-steps` | `WithMouseSteps` | interpolation steps along the path |
| `--mouse-speed-min` / `--mouse-speed-max` | `WithMouseSpeed(min,max)` | **both** required |
| `--mouse-deviation` | `WithMouseDeviation` | path randomness 0.0–1.0 |
| `--scroll-duration` | `WithDuration` | base scroll animation in ms |
| `--scroll-physics` | `WithPhysics` | see the constraint in §6 |

### CDP footprint / capture (Phase 30, default OFF)

Minimal CDP footprint is just **correct default behavior** — there is no
`--cdp-stealth` master switch. A plain session (`goto` + `snapshot`, no capture
flags, no mock routes, no plugins) enables **none** of the `Runtime`, `Network`,
or `Fetch` CDP domains. HTTP↔JS identity coherence (UA / `Accept-Language` /
`Sec-Ch-Ua*` / `navigator.userAgentData`) is carried by Chrome's
`Emulation.setUserAgentOverride`, which has **no `enable` command** and therefore
adds zero CDP footprint. The footprint-adding features are opt-in, each following
the same **CLI > profile > default** precedence as every other flag:

| Flag | Meaning |
|---|---|
| `--console-capture` | capture console messages for the `console` command; installs the `RuntimeConsoleAPICalled` subscription (enables the **Runtime** domain). Default OFF. |
| `--request-capture` | capture network requests for the `requests`/`request` commands; installs the `NetworkRequestWillBeSent` subscription (enables the **Network** domain). Default OFF. |

The network interceptor (godoll **Fetch** domain) is created + enabled **lazily**
on the first mock `route` and torn down when the last route is removed — so a
session that never adds a route never enables Fetch. Plugin `onRequest`/`onResponse`
hooks enable their own Network subscription on demand, independent of the baseline.

See [`cdp-footprint.md`](cdp-footprint.md) for the per-command inventory and the
offline harness baseline assertion.

---

## 3. Per-session proxy

`parseProxyConfig` (`types/context.go`) turns `--proxy` + `--proxy-auth` into a
`godoll/browser.ProxyConfig`:

- Scheme is required and validated (`http`/`https` → `http`, `socks5`, `socks4`).
- **URL-embedded credentials are rejected loudly** (`user:pass@host` in the URL)
  — they would surface as a confusing 407, and Chrome removed them. Use
  `--proxy-auth`.
- Auth is applied via CDP `Fetch.continueWithAuth`, so the credential never
  reaches `--proxy-server` and never appears in argv.

The proxy is bound **per named session**, not as one global egress IP.

> The deprecated `Config.Proxy` field is bridged from `Stealth.Proxy` by
> `ResolveStealth` for backward compatibility; prefer `Stealth.Proxy` everywhere.

---

## 4. Fingerprint: single source of truth + consistency

`profileFromStealth` overlays the resolved `cfg.Stealth` onto
`stealth.DefaultProfile()` and returns the **one** `stealth.Profile` that drives
both godoll's evasion JS and rod-cli's interceptor. Notable behavior:

- `SpoofClientHints` is forced **on** so `Sec-Ch-Ua`, `navigator.userAgentData`,
  and the UA all tell one Chrome-version story (FINGERPRINT-02/03). Shipping the
  default identity with CH off would itself be a tell.
- Client-Hints are **derived from the active UA** (`parseChromeMajor`), not the
  old hardcoded version `121`.
- `--canvas-noise` gates **both** `SpoofCanvas` and `SpoofAudioContext`.
- The randomly generated fingerprint in `createPage` is used *only* for
  dimensions godoll still needs (e.g. WebGL VideoCard) — **never** as the
  identity.

**Consistency gate** (`deriveAndValidateFingerprint`, run at the spawn seam):

- *Derive-when-unset, reject-when-user-conflicts.* Platform is derived from the
  UA OS token; a user-set `--platform` that contradicts the UA is rejected by
  name. Locale is derived from `languages[0]`; a contradicting `--locale` is
  rejected.
- If the UA carries no recognized OS token, a user-pinned `--platform` must be one
  of the known values (`Win32`, `MacIntel`, `Linux`) — it does **not** fail open.
- Hardware/screen values are range-checked (`hardwareConcurrency` 1–256,
  `deviceMemory` 1–64 GB, screen width/height both-or-neither & positive).
- String fields carrying control chars or double-quotes are rejected before they
  can reach the JS injection boundary.
- `timezone` ↔ proxy-geo is **warn-only** (a stderr line), never a hard failure —
  geo-IP resolution needs network access and would break the offline harness.

All of these fail at daemon spawn (a returned error `main.go` surfaces), not
mid-session.

---

## 5. Hardening toggles and the `*bool` pattern

`WebRTCLeakProtection` and `CanvasNoise` are `*bool`, not `bool`, on purpose:
`nil` ("unset") must be distinguishable from a deliberate `false`. A config file
that persisted `canvasNoise: false` arrives as a non-nil `*bool(false)` and
**survives** `ResolveStealth` — only a `nil` (omitted key, no flag) resolves to
the hardened default `true`. Consumers read them through `boolVal(p, true)`.

- **WebRTC (HARDEN-01):** when on, `launchBrowser` sets the
  disable-non-proxied-UDP browser preference *and* `createPage` injects godoll's
  `EvadeWebRTC` JS wrapper, so a real local IP cannot leak past a proxy.
- **Canvas/WebGL/Audio noise (HARDEN-02):** seeded by a per-session
  `noiseSeed` (generated once from `crypto/rand` in `NewContext`, threaded via
  `EvasionManager.SetNoiseSeed`). The seed makes the noise **stable within a
  session** — re-reads return an identical hash — because an unstable hash across
  reads is itself a tell.

---

## 6. Human-behavior tuning and the zero-regression invariant

The humanize knobs are consumed in `actions/actions.go` by three builders —
`typingOpts`, `mouseOpts`, `scrollOpts` — that read `ctx.HumanizeTuning()` (the
resolved `StealthConfig`) at the `click`/`fill`/`type`/`scroll` call sites.

Every tunable is a **pointer**: `nil` means "emit no godoll option," so godoll's
own default applies and behavior is byte-for-byte the pre-v1.6 default. A bad
value is rejected at spawn by `validateHumanizeTuning` (out-of-range, inverted, or
half-set min/max ranges) — godoll's `rand` would otherwise *panic per-keystroke*
deep inside a frozen session.

**Honest constraints (documented, not bugs):**

- **No separate "delay jitter" knob.** "Delay jitter" is the variance produced by
  the `--typing-speed-min`/`--typing-speed-max` spread; a wider spread = more
  jitter. There is deliberately no dedicated jitter field.
- **`--scroll-physics=false` cannot disable physics.** godoll's `WithPhysics()`
  can only *enable* physics, and physics is godoll's own default. So `true` emits
  `WithPhysics()` and `false`/unset emits nothing — godoll's default physics still
  applies. Disabling it would need a godoll signature change (out of scope for
  v1.6). The flag is kept for round-trip symmetry and forward-compat.

---

## 7. Persistence and isolation summary

- A stealth profile, saved as JSON, is **set once and inherited** by every later
  command hitting the same daemon session (PROFILE-01).
- Resolution is deterministic and happens **once at spawn** (PROFILE-02); there is
  no per-command stealth state, and no bleed across named sessions.
- Secrets (`--proxy-auth`) never enter argv; the daemon log is `0600`.
