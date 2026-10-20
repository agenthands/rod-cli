# Stealth Configuration Surface (v1.7)

This is the contributor-facing reference for `rod-cli`'s configurable stealth
surface: the config spine, the flag catalog, the per-session proxy, the
fingerprint consistency model, the anti-fingerprint hardening toggles, the
advanced fingerprint-dimension toggles, the built-in profile library, the
opt-in CDP-capture flags, and the human-behavior tuning knobs — including the
honest constraints on what each can and cannot do.

For the system-level picture see [../ARCHITECTURE.md](../ARCHITECTURE.md); for
what the validation actually proves see
[stealth-validation.md](stealth-validation.md); for the per-domain CDP footprint
behind the opt-in capture flags (§2) see [cdp-footprint.md](cdp-footprint.md).

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
| Apply | `types/context.go` `preparePage` | `EvasionManager.SetProfile` + `Apply` + `EvadeWebRTC` + `applyEmulationIdentity` per page |

**Resolved once, at spawn.** `ResolveStealth` runs before `NewContext` freezes
`Config`. A stealth flag therefore only takes effect for a session if it is
present when that session's daemon spawns. Passing a stealth flag to an
already-running session prints a stderr warning (`stealth flags apply at session
spawn — run close first to re-apply`) and is otherwise inert.

**Precedence: CLI flag > profile file > built-in default** (`ResolveStealth`):

- *Tier 3* — `stealth.DefaultProfile()` (godoll), plus the hardened defaults
  rod-cli adds (see §4).
- *Tier 2* — a `--profile`, overlaid onto `cfg.Stealth`. A **bare name** resolves
  to an embedded **built-in profile first** (Phase 32, see §8), then to
  `~/.rod-cli/profiles/<name>.json`; a value that looks like a path (contains a
  separator or ends in `.json`) loads verbatim. A missing/malformed profile is a
  **loud** failure: the daemon aborts rather than silently shipping a default identity.
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
| `--profile` | a built-in name, a custom name, or a path to a `stealth.Profile` JSON | bare name → **built-in first** (§8), then `~/.rod-cli/profiles/<name>.json`; `--profile=list` lists the built-ins |

### Hardening toggles (Phase 27, default ON)

| Flag | Meaning |
|---|---|
| `--webrtc-protection` | prevent WebRTC local-IP leaks; `--webrtc-protection=false` to disable |
| `--canvas-noise` | stable-per-session canvas/WebGL/audio noise; `--canvas-noise=false` to disable |

### Advanced fingerprint-dimension toggles (Phase 33, default ON)

These four activate godoll's per-vector fingerprint-dimension injectors (fonts,
media devices, battery, codecs; `plugins` rides the same path). Each is a `*bool`
on the same **CLI > profile > default** precedence (§5), default ON; pass
`=false` to disable. They follow the exact `--canvas-noise` pattern.

| Flag | Meaning |
|---|---|
| `--font-spoof` | spoof OS-coherent font availability; `--font-spoof=false` to disable |
| `--media-devices-spoof` | spoof `navigator.mediaDevices.enumerateDevices()`; `--media-devices-spoof=false` to disable |
| `--battery-spoof` | spoof `navigator.getBattery()` (level/charging); `--battery-spoof=false` to disable |
| `--codec-spoof` | spoof media `canPlayType()` / codec support; `--codec-spoof=false` to disable |

**Coherent per profile OS.** The dimensions are generated **constrained to the
active profile's OS + locale** (`FPWithOS`/`FPWithLocales`, OS mapped from
`navigator.platform`), so a Windows profile gets Windows-family fonts/devices and
a macOS profile gets macOS-family — never an unconstrained random draw on a pinned
identity (that would be a *new* cross-layer tell). Generation is seeded with the
per-session noise seed, so a re-read (or a recreated page) within one session
reads the **same** values (stability).

**On/off guidance.** Leave ON (the default) for the most complete, coherent
surface. Turn an individual vector OFF only to diagnose a site that reacts to that
specific surface, or to fall back to the browser default for one dimension while
keeping the others hardened. A toggle OFF *fully* suppresses that vector's
injection (it reverts to the un-hardened browser default — e.g. with
`--battery-spoof=false`, headless `getBattery` reads back `level: 1, charging:
true`, vs a spoofed laptop-like `level: 0.38, charging: false` when ON). The
`media-devices-spoof` toggle-off is the harness-proven leg (ON vs OFF device
signatures differ), and so is `font-spoof` (`TestFontSpoof`); battery and codecs
revert by the same per-dimension gate.

**Performance.** Each is a one-time JS injection at page setup
(`EvalOnNewDocument`); cost is negligible and there is no per-call overhead.

**Real Chrome, no TLS.** Like the rest of the stack these are real-Chrome JS
overrides only — there is **no** TLS/JA3 spoofing (see `stealth-validation.md`).

> **Font proof:** godoll's font injector offsets canvas `measureText` widths.
> `TestFontSpoof` (`tests/detection_test.go`) asserts on the live page that ON
> differs from `--font-spoof=false`, stays stable within a session, and reverts
> to the baseline when OFF. OS-coherence of the font *set* is enforced at
> generation (the OS-keyed font list).

### Humanize tuning (Phase 28, all opt-in)

| Flag | godoll option | Notes |
|---|---|---|
| `--typing-speed-min` / `--typing-speed-max` | `WithTypingSpeed(min,max)` | **both** required; the spread *is* the "delay jitter" |
| `--typo-rate` | `WithTypoRate` | probability 0.0–1.0 per keystroke |
| `--mouse-tremor` | `WithMouseTremor` | `--mouse-tremor=false` to disable |
| `--mouse-steps` | `WithMouseSteps` | interpolation steps along the path |
| `--mouse-speed-min` / `--mouse-speed-max` | `WithMouseSpeed(min,max)` | **both** required |
| `--mouse-deviation` | `WithMouseDeviation` | path randomness 0.0–1.0 |
| `--scroll-duration` | none (rod-cli) | ms each `mousewheel` step spreads across its ~100 px notches; unset = random 20–60 ms gap per notch |
| `--scroll-physics` | none | ignored; accepted for config compatibility (see §6) |

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

The opt-in CDP proxy (v2.0, `internal/cdpproxy/`) is also default OFF:

| Flag | Meaning |
|---|---|
| `--cdp-proxy` | route CDP through the in-process WebSocket proxy: logs traffic for the `cdp-traffic` command and strips getter values from `Runtime.getProperties` responses. Default OFF. |
| `--cdp-jitter-ms` | max random delay in ms before each CDP command; `0` = off; requires `--cdp-proxy`; above 1000 prints a stderr warning. |
| `--no-cdp-proxy` | bypass the CDP proxy even when `--cdp-proxy` is set. |

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
- Auth is handled **out-of-band** — the primary carrier is a local CONNECT relay
  (godoll `StartProxyRelay`, which injects `Proxy-Authorization` on the upstream
  CONNECT and points Chrome at the relay's credential-free URL); a CDP auth
  handler (`SetupBrowserAuth` → go-rod `HandleAuth` → `Fetch.continueWithAuth`) is
  also registered as a fallback. Either way the credential never reaches
  `--proxy-server` and never appears in argv.

The proxy is bound **per named session**, not as one global egress IP.

> The deprecated `Config.Proxy` field is bridged from `Stealth.Proxy` by
> `ResolveStealth` for backward compatibility; prefer `Stealth.Proxy` everywhere.

---

## 4. Fingerprint: single source of truth + consistency

`profileFromStealth` overlays the resolved `cfg.Stealth` onto
`stealth.DefaultProfile()` and returns the **one** `stealth.Profile` that drives
both godoll's evasion JS and the HTTP identity (`applyEmulationIdentity` →
`Emulation.setUserAgentOverride`). Notable behavior:

- `SpoofClientHints` is forced **on** so `Sec-Ch-Ua`, `navigator.userAgentData`,
  and the UA all tell one Chrome-version story (FINGERPRINT-02/03). Shipping the
  default identity with CH off would itself be a tell.
- Client-Hints are **derived from the active UA** (`parseChromeMajor`), not the
  old hardcoded version `121`.
- `--canvas-noise` gates **both** `SpoofCanvas` and `SpoofAudioContext`.
- The fingerprint generated in `preparePage` (seeded, constrained to the
  profile's OS and locale) is used *only* for dimensions godoll still needs (WebGL
  VideoCard and the §5 dimension injectors) — **never** as the identity.

**Consistency gate** (`deriveAndValidateFingerprint`, run at the spawn seam):

- *Derive-when-unset, reject-when-user-conflicts.* Platform is derived from the
  UA OS token; a user-set `--platform` that contradicts the UA is rejected by
  name. Locale is derived from `languages[0]`; a contradicting `--locale` is
  rejected.
- If the UA carries no recognized OS token, a user-pinned `--platform` must be one
  of the known values (`Win32`, `MacIntel`, `Linux`) — it does **not** fail open.
- Hardware/screen values are range-checked (`hardwareConcurrency` 1–256,
  `deviceMemory` 1–8 GB — `navigator.deviceMemory` is **capped at 8** by the W3C
  Device Memory API, so a higher value is a synthetic tell and is rejected; screen
  width/height both-or-neither & positive).
- String fields carrying control chars or double-quotes are rejected before they
  can reach the JS injection boundary.
- `timezone` ↔ proxy-geo is **warn-only** (a stderr line), never a hard failure —
  geo-IP resolution needs network access and would break the offline harness.


**Zero / empty = unset (no spoofing for that field).** The identity pins
(`Screen`, `UserAgent`, `HardwareConcurrency`, `DeviceMemory`, `Vendor`,
`Languages`, `AcceptLanguage`) all follow one rule: a zero value or an empty
string means *"derive from the actual browser"*, not "default to a hardened
value." A `rod-cli.yaml` with `screen: {width: 0, height: 0}` and no
`userAgent` produces a **raw fingerprint** — the browser reports real host
screen geometry, real OS, and the stock Chromium UA. This is deliberate: zero
is the absence of spoofing, not a default identity.

To ship a hardened fingerprint you MUST set the identity pins explicitly —
either in `rod-cli.yaml`, via `--user-agent`/`--profile` CLI flags, or through
a built-in profile (§8). Start from a built-in (`--profile=windows-11-chrome`)
for a vetted, coherent baseline.

All of these fail at daemon spawn (a returned error `main.go` surfaces), not
mid-session.

---

## 5. Hardening toggles and the `*bool` pattern

`WebRTCLeakProtection`, `CanvasNoise`, and the Phase-33 dimension toggles
(`FontSpoof`, `MediaDevicesSpoof`, `BatterySpoof`, `CodecSpoof`) are `*bool`, not
`bool`, on purpose: `nil` ("unset") must be distinguishable from a deliberate
`false`. A config file that persisted `canvasNoise: false` (or
`mediaDevicesSpoof: false`) arrives as a non-nil `*bool(false)` and **survives**
`ResolveStealth` — only a `nil` (omitted key, no flag) resolves to the hardened
default `true`. Consumers read them through `boolVal(p, true)`.

- **WebRTC (HARDEN-01):** when on, `launchBrowser` sets the
  disable-non-proxied-UDP browser preference *and* `preparePage` injects godoll's
  `EvadeWebRTC` JS wrapper, so a real local IP cannot leak past a proxy.
- **Canvas/WebGL/Audio noise (HARDEN-02):** seeded by a per-session
  `noiseSeed` (generated once from `crypto/rand` in `NewContext`, threaded via
  `EvasionManager.SetNoiseSeed`). The seed makes the noise **stable within a
  session** — re-reads return an identical hash — because an unstable hash across
  reads is itself a tell.
- **Fingerprint dimensions (Phase 33 / EVAD-01..03):** `preparePage` calls
  `em.SetFingerprint(fp)` (before `SetProfile`, so the config identity stays the
  source of truth) with an **OS/locale-constrained, seeded** fingerprint, then
  `em.SetDimensionOptions(...)` from the four toggles. godoll's
  `applyFingerprintDimensions` injects only the enabled vectors. The same
  `noiseSeed` seeds generation, so the dimensions are stable per session; a toggle
  OFF skips that vector's injection entirely. See §2 "Advanced fingerprint-
  dimension toggles" for the per-flag table and the font proof.

---

## 6. Human-behavior tuning and the zero-regression invariant

The humanize knobs are consumed in `actions/actions.go` by two builders —
`typingOpts` and `mouseOpts` — that read `ctx.HumanizeTuning()` (the resolved
`StealthConfig`) at the `click`/`fill`/`type`/`hover`/`mousemove`/`mousewheel`/`drag`
call sites; `wheelScroll` reads `ScrollDuration` directly.

Every tunable is a **pointer**: `nil` means "emit no godoll option" (for
`ScrollDuration`, keep the random per-notch gap), so godoll's own default
applies and behavior is byte-for-byte the pre-v1.6 default. A bad value is
rejected at spawn by `validateHumanizeTuning` (out-of-range, inverted, or
half-set min/max ranges) — godoll's `rand` would otherwise *panic per-keystroke*
deep inside a frozen session.

**Honest constraints (documented, not bugs):**

- **No separate "delay jitter" knob.** "Delay jitter" is the variance produced by
  the `--typing-speed-min`/`--typing-speed-max` spread; a wider spread = more
  jitter. There is deliberately no dedicated jitter field.
- **`--scroll-physics` is ignored.** `mousewheel` sends real ~100 px wheel
  notches and Chrome does the scrolling, so there is no physics model to switch.
  The flag and the `scrollPhysics` config key are accepted only so existing
  configs still load.

---

## 7. Persistence and isolation summary

- A stealth profile, saved as JSON, is **set once and inherited** by every later
  command hitting the same daemon session (PROFILE-01).
- Resolution is deterministic and happens **once at spawn** (PROFILE-02); there is
  no per-command stealth state, and no bleed across named sessions.
- Secrets (`--proxy-auth`) never enter argv; the daemon log
  (`os.TempDir()/rod-cli-daemon.log`, shared by all sessions, started over once
  it passes 1 MiB) is `0600`.
- Each session's Chrome runs on its own browser profile dir under
  `types.DefaultProfileRoot()` (`~/.cache/rod-cli/profiles/` on Linux; the config
  key `browserTempDir` overrides it, and the legacy `./rod/browser` counts as
  unset) — not the stealth profile JSON in `~/.rod-cli/profiles/`. The profile dir
  holds `owner.pid` and `chrome-tmp`, the path of the session Chrome's own short
  temp dir (`os.TempDir()/chrome-XXXX`, or `/tmp` when `TMPDIR` is longer than 40
  chars), which Chrome uses as `TMPDIR`. Both are removed on `close`, the
  15-minute idle timeout, and SIGINT/SIGTERM/SIGHUP. A daemon killed
  with `kill -9` leaves them; the next session Chrome launch under the default root
  removes profiles whose owner is dead or whose PID now belongs to a non-rod-cli
  process, and the next daemon start removes port files
  (`os.TempDir()/rod-cli-<session>.port`) of dead daemons.

---

## 8. Built-in profile library (Phase 32, PROF-01..04)

rod-cli ships a vetted library of **Chrome-only desktop** identity profiles
embedded in the binary (`//go:embed`, `types/profiles/*.json`). Pick one by name —
no JSON authoring, no external file:

```
rod-cli --profile=list                       # discover (honors --raw / --json)
rod-cli --profile=windows-11-chrome goto …   # select by name
```

| Name | OS (UA token) | `navigator.platform` | Screen | HW cores / mem |
|---|---|---|---|---|
| `windows-11-chrome` | Windows NT 10.0 | `Win32` | 1920×1080 @1.0 | 8 / 8 GB |
| `windows-11-desktop-1440p` | Windows NT 10.0 | `Win32` | 2560×1440 @1.0 | 16 / 8 GB |
| `windows-10-chrome` | Windows NT 10.0 | `Win32` | 1920×1080 @1.0 | 12 / 8 GB |
| `windows-10-laptop` | Windows NT 10.0 | `Win32` | 1366×768 @1.0 | 4 / 8 GB |
| `macos-applesilicon-chrome` | Intel Mac OS X 10_15_7 | `MacIntel` | 2560×1600 @2.0 | 8 / 8 GB |
| `macos-intel-chrome` | Intel Mac OS X 10_15_7 | `MacIntel` | 1920×1080 @1.0 | 8 / 8 GB |

Notes:

- **Real Chrome, no spoofed TLS.** Every profile drives a real Chrome whose
  TLS/JA3 is authentic by construction — there is no TLS-spoofing knob in the
  profile schema or anywhere in rod-cli (see `docs/stealth-validation.md`).
- **Chrome can't tell Win 10 from Win 11** in the UA (both report `Windows NT 10.0`)
  and reports `MacIntel` for **both** Apple-Silicon and Intel Macs — so the Win
  variants share a UA and differ in screen/hardware, and the two Mac profiles
  differ in screen/scale, not `platform`. This matches genuine Chrome.
- **Vetted gate (PROF-02).** `types.TestBuiltinProfilesAreVetted` runs EVERY
  built-in through the v1.6 consistency validator offline (UA↔platform, locale↔
  languages, plausible screen/hardware, a parseable Chrome major), and the
  detection harness drives a representative subset live; an incoherent profile
  **fails the build** and is not shipped.
- **Precedence unchanged (PROF-04).** Built-ins overlay at the same Tier 2 as a
  custom profile file, so CLI flags still win (`--user-agent`/`--platform`/… over
  the profile) and a custom profile file or path still works exactly as before.
- **Reserved names.** A built-in name resolves to the embedded profile *before* the
  user-dir path. To use a custom profile whose name collides with a built-in, pass
  an explicit path (e.g. `--profile=./windows-11-chrome.json` or any path ending in
  `.json`). `list` is reserved for discovery and never names a profile.
