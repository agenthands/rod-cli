# Pitfalls Research

**Domain:** Proving & extending browser stealth in a Go CLI (rod-cli / godoll) — CI-able detection harness, proxy support, configurable fingerprints, canvas/WebGL/WebRTC hardening, humanize tuning, on a persistent daemon-shared browser.
**Researched:** 2026-06-24
**Confidence:** HIGH (grounded in current detection research + direct read of `types/context.go`, `types/config.go`; April 2026 benchmark data confirms the failure modes)

> **Reality anchor for the roadmap.** As of April 2026, independent benchmarks show *most* commercial "stealth browser" APIs still fail CreepJS / fingerprint-consistency tests, and the Runtime.enable CDP tell was only partially neutered by V8 changes in mid-2025. This milestone's value is **proving** what rod-cli actually evades and being **honest about the ceiling** — not winning an unwinnable arms race. Every pitfall below is framed around that.

---

## Critical Pitfalls

### Pitfall 1: Flaky live third-party sites in blocking CI

**What goes wrong:**
The detection harness asserts against live CreepJS / bot.sannysoft / fingerprintjs / Cloudflare / DataDome over the network in CI. Tests go red because the remote site changed its markup/scoring, rate-limited the CI IP, rotated a challenge, or was simply down — not because rod-cli regressed. The team starts ignoring red CI, which defeats the entire point of a "proven stealth" milestone.

**Why it happens:**
Live suites *feel* like the most authoritative validation, so they get wired directly into the blocking test path. The retrospective already records being burned by flaky validation (`v1.5`: verbosity/round-trips), and v1.3 needed `retry.Fetch` to kill "dozens of flaky failure modes on page loads" — network-dependent assertions are this project's known weak spot.

**How to avoid:**
- **Two-tier harness.** Tier 1 (blocking/CI): a **vendored, offline** detection page bundled in-repo (extend the existing `internal/plugin/scanner/testserver` pattern) that runs the *same JS probes* CreepJS/sannysoft use — `navigator.webdriver`, `navigator.plugins`, UA string, WebGL vendor/renderer, permissions, `navigator.userAgentData`, CDP-stack probes — and asserts on the **evaluated values in the live browser**. No external network. Deterministic. Tier 2 (non-blocking/manual/nightly): real CreepJS/Cloudflare/DataDome behind a build tag or `-tags=live`, allowed to fail, reported as informational.
- Pin the vendored probe set to a commit and document its provenance so "the suite changed" becomes an explicit, reviewed update rather than a silent CI break.
- Never gate merge on a network call to a domain you don't control.

**Warning signs:**
Tests reference `https://` URLs to third-party domains; CI failures that "go away on re-run"; test names like `TestCreepJSScore` hitting the real site.

**Phase to address:** **Phase 1 (Detection Harness)** — establishes the offline Tier-1 backbone before any evasion feature is built. This is the deterministic foundation everything else is validated against.

---

### Pitfall 2: Asserting on source/config instead of live browser behavior (the `onDOMNodeInserted` lesson)

**What goes wrong:**
A stealth evasion is "wired" — the code calls `em.Apply()`, sets a header, or injects a script — and a test asserts that the *code path ran* or the *config field is set*, so it passes. But the evasion is silently inert in the running browser: the CDP domain wasn't enabled, the script ran too late (after the detector already read the property), or `EvalOnNewDocument` didn't apply to the right execution context. The fingerprint *looks* spoofed in Go but `navigator.webdriver` is still `true` in the page.

**Why it happens:**
This is the project's single most expensive recurring lesson. In v1.5, `onDOMNodeInserted` was wired-but-silent because the CDP DOM domain wasn't enabled — caught *only* by driving the live binary, not by source/automated checks. The retrospective elevates this to a Key Lesson: "Documenting/validating against the *live binary* catches wired-but-silent behavior that source inspection misses." Stealth has the identical shape — `createPage()` calls `em.Apply()` and ignores its error (`_ = em.Apply()`), and the fingerprint generate error is swallowed (`if err == nil`). Both can no-op without any test noticing.

**How to avoid:**
- **Every stealth assertion reads back from the page**, not from Go state. Pattern: navigate to the offline probe page → `page.Eval("() => navigator.webdriver")` → assert `false`. Never assert "we set the UA in the Config struct."
- Stop swallowing the two error paths in `createPage()`: surface or at minimum log `em.Apply()` and fingerprint-generation failures so a silent no-op is observable.
- For each evasion (webdriver, plugins, UA, WebGL, canvas, permissions, WebRTC), write a probe that *reproduces what the detector measures* and assert the post-spoof value.
- Validate in the **daemon context** the way real callers hit it (a second command against the already-running browser), not just a fresh single-shot launch — config that applies on first launch may not apply on a reused browser (see Pitfall 8).

**Warning signs:**
Test assertions reference Config/Profile/Fingerprint Go fields; no `page.Eval`/`MustEval` reading a navigator/WebGL property; green tests but manual `rod-cli goto <probe>` shows tells; `_ =` discarding evasion return values.

**Phase to address:** **Phase 1 (Detection Harness)** sets the read-from-live-page convention; **every subsequent evasion phase** must use it as its definition-of-done.

---

### Pitfall 3: CDP itself is detectable regardless of fingerprint spoofing

**What goes wrong:**
The team perfects `navigator.webdriver`, UA, WebGL, and canvas — then a modern detector (DataDome, Cloudflare, CreepJS) flags the session anyway because the *Chrome DevTools Protocol connection* is observable. The `Runtime.enable` call rod/go-rod issues to receive console/exception events creates a detectable side-effect; serialization of certain objects over the CDP websocket triggers observable behavior. No amount of JS-property spoofing hides the transport. Stealth "passes sannysoft" but fails the suites that actually matter.

**Why it happens:**
go-rod, like Puppeteer/Playwright/Selenium, drives Chrome over CDP and enables the Runtime domain. rod-cli's `createPage()` *relies* on Runtime events (`RuntimeConsoleAPICalled` for console logs, `NetworkRequestWillBeSent` for the request log). Those subscriptions are exactly the CDP tells detectors probe. A mid-2025 V8 change reduced one specific Runtime.enable side-channel, but CDP-presence detection as a class is alive and is the frontier signal in 2025–2026.

**How to avoid:**
- **Add explicit CDP-tell probes to the Tier-1 harness** (e.g. the `Runtime.enable` error-serialization side-effect, stack-depth/`console.debug` getter tricks) so rod-cli *knows* its CDP exposure rather than assuming fingerprint spoofing covers it.
- Investigate whether the request/console logging in `createPage()` can be made **opt-in** — it forces Runtime/Network domain enablement on every page. If a caller doesn't need console/request logs, not enabling those domains shrinks the CDP surface.
- Evaluate godoll/rod options for the Runtime.enable→Disable context-acquisition trick or a flagged "low-CDP-footprint" mode; if godoll can't do it, document the limitation explicitly.
- **Set scope honestly:** treat "fully hide CDP" as a research spike with an explicit YES/NO outcome, not a guaranteed deliverable. CDP-hiding is the part most likely to be partially-or-not achievable.

**Warning signs:**
Stealth passes sannysoft/CreepJS-property checks but real Cloudflare/DataDome still challenges; assuming "fingerprint spoofed = undetectable"; no probe in the harness that targets the protocol layer rather than JS properties.

**Phase to address:** **Phase 1 (Harness)** adds CDP-tell probes; a dedicated **CDP-footprint spike** (likely its own small phase) decides feasibility and sets the honest ceiling. Flag this phase as needing deeper research.

---

### Pitfall 4: "Passes headful, fails headless" (and vice-versa)

**What goes wrong:**
Validation is done with `Headless: false` (rod-cli's current default) and looks clean, but the agent's real usage is headless, where extra tells appear: missing/zero-size screen and `outerWidth`/`outerHeight`, absent plugins, different WebGL/ANGLE renderer, no audio devices, `HeadlessChrome` token leaking through some surface, and timing differences. Stealth is declared "proven" against the wrong mode.

**Why it happens:**
`DefaultConfig.Headless = false`, so the easiest local validation runs headful. New-headless (`--headless=new`) closes much of the gap but not all, and the launcher flags rod-cli already sets (`no-gpu`, `--disable-dev-shm-usage`, `--disable-features=HttpsUpgrades`) interact with rendering/fingerprint surfaces differently across modes.

**How to avoid:**
- **Run the Tier-1 harness in BOTH modes** (`Headless: true` and `false`) as separate, matrixed CI cases. A regression that only shows headless must turn CI red.
- Prefer `--headless=new` over legacy headless and verify the harness confirms it (legacy headless leaks `HeadlessChrome` in more places).
- Audit launcher flags for mode-dependent fingerprint impact: `no-gpu` can change WebGL renderer (a spoof-vs-real mismatch risk, see Pitfall 5); decide per-mode.

**Warning signs:**
All validation screenshots/logs are headful; no headless row in the CI matrix; `--headless=new` not explicitly set; `no-gpu` left on while spoofing a GPU-backed WebGL renderer.

**Phase to address:** **Phase 1 (Harness)** must matrix headless/headful from day one; revisit in the canvas/WebGL phase.

---

### Pitfall 5: Spoofing one signal while leaking another (fingerprint inconsistency)

**What goes wrong:**
The UA says "Chrome on Windows" but `navigator.platform`/WebGL renderer/canvas font-smoothing still says Linux (the host) — and 2026 detectors specifically cross-check that "the same device and the same browser" story holds. A single inconsistency tanks the CreepJS trust score and is *more* damning than an honest default. rod-cli is acutely exposed: it generates a **random** fingerprint per page (`fg.Generate()`), derives a `stealth.Profile` from it for headers, **but hardcodes the Client-Hints version to `121`** in `updateInterceptorRules()` regardless of the actual UA/Chrome version, and runs on Linux while potentially spoofing Windows. UA ≠ UA-CH ≠ `navigator.userAgentData` ≠ WebGL ≠ platform.

**Why it happens:**
Spoofing is done signal-by-signal in different code locations (evasion manager sets JS properties; the network interceptor sets HTTP headers with a separately-hardcoded CH version), so they drift. The host OS (Linux CI/dev) bleeds through WebGL/canvas/fonts that JS spoofing doesn't touch. Client Hints is now weighted *higher* than the classic UA string, so a UA/UA-CH mismatch is a top-tier red flag.

**How to avoid:**
- **Define a single source of truth** for a fingerprint identity (UA, UA-CH version, platform, WebGL vendor/renderer, locale, timezone, screen) and derive *all* surfaces — JS properties AND HTTP headers AND Client Hints — from it. Kill the hardcoded `"121"` literal; the CH version must equal the UA's Chrome major version.
- **Add a consistency invariant test to Tier-1**: navigate the probe page and assert UA, `navigator.userAgentData`, `Sec-CH-UA`, `navigator.platform`, and WebGL all tell the *same* OS+version story. This is a blocking gate, not a nicety.
- Don't spoof an OS the host can't physically back convincingly unless WebGL/canvas are *also* spoofed to match; a Linux host claiming Windows must spoof the WebGL renderer string consistently or stay Linux.

**Warning signs:**
Different OS/version literals in `context.go` header code vs evasion code; hardcoded version strings (`121`); CreepJS "lies" / low trust score; UA-CH absent while UA present.

**Phase to address:** **Phase 3 (Configurable Fingerprints)** — must build the single-source-of-truth identity and the consistency invariant test. Canvas/WebGL phase extends the same invariant.

---

### Pitfall 6: Over-spoofing — entropy spikes & impossible fingerprints more detectable than defaults

**What goes wrong:**
Aggressive canvas/WebGL noise injection produces a fingerprint that is *unique on every page load* (high-entropy outlier), or an impossible combination (a GPU/renderer that doesn't exist, screen dimensions no real device ships, fonts that can't coexist). Detectors flag "this changes every time" or "this can't be real" — the hardening makes the bot *more* identifiable, not less. Canvas noise that follows a detectable pattern (uniform per-pixel perturbation) is itself a signature.

**Why it happens:**
"More spoofing = more stealth" intuition. Random-per-page generation (rod-cli's current `fg.Generate()` on every `createPage`) maximizes entropy/instability — the opposite of blending in. Canvas-noise libraries often add statistically detectable noise.

**How to avoid:**
- **Stability over uniqueness:** a session should present a *consistent, plausible, common* fingerprint, not a fresh random one per page. Pin the identity per session (this is also why "configurable, not random-only" is a milestone goal).
- Prefer **plausible real-device profiles** (common GPU/screen/OS combos) over synthetic randomness. Validate any canvas/WebGL noise against CreepJS's "lies"/entropy detection — if noise *raises* detectability, it's worse than no noise.
- Add a Tier-1 check that the same session yields the *same* canvas/WebGL hash across loads (stability), and that values fall in plausible ranges.

**Warning signs:**
Fingerprint changes every navigation; CreepJS entropy/uniqueness score worsens after enabling hardening; canvas hash differs per pixel-read in a detectable pattern; renderer strings that don't map to a real GPU.

**Phase to address:** **Phase 4 (Canvas/WebGL/WebRTC hardening)** with the stability invariant; **Phase 3** establishes per-session pinning so hardening has a stable base.

---

### Pitfall 7: Proxy auth, DNS/WebRTC leaks bypassing the proxy

**What goes wrong:**
Three distinct failures: (a) **Auth** — Chromium has no programmatic proxy-auth dialog; setting `--proxy-server=user:pass@host` silently drops credentials and the proxy 407s, or pops a native dialog that hangs a headless/daemon session. (b) **WebRTC leak** — even with a working proxy, WebRTC STUN/ICE exposes the host's *real* public/local IP via JS, so the site sees a proxy IP in HTTP but the true IP in WebRTC — a glaring inconsistency. (c) **DNS leak** — DNS resolves outside the proxy, leaking the real resolver/geo. rod-cli today does `browserLauncher.Proxy(cfg.Proxy)` only — **no auth handling, no WebRTC mitigation, no DNS-through-proxy guarantee.**

**Why it happens:**
`launcher.Proxy()` maps to `--proxy-server`, which handles routing but not authentication and does nothing about WebRTC/DNS. Auth in Chromium requires intercepting `Fetch.authRequired` / `Network.requestWillBeSentExtraInfo` over CDP and answering with credentials — a separate mechanism. WebRTC is a parallel network stack the proxy flag never touches.

**How to avoid:**
- **Auth:** implement CDP-level proxy auth (handle the auth-required event with the supplied credentials) rather than relying on the URL or a dialog. Verify against an authenticated proxy in a test (a local auth proxy fixture, offline).
- **WebRTC:** force WebRTC policy so it cannot leak local/host IPs (`--force-webrtc-ip-handling-policy=disable_non_proxied_udp` / disable, or spoof via the evasion layer), and **add a WebRTC-leak probe to Tier-1** that asserts no host IP is exposed.
- **DNS:** ensure DNS resolves through the proxy (proxy-side resolution); probe that the resolved geo/IP matches the proxy, not the host.
- **Consistency:** add a leak-matrix test — HTTP egress IP, WebRTC-reported IP, and timezone/locale must all agree with the proxy's geo (ties into Pitfall 9).

**Warning signs:**
`--proxy-server=...@...` with embedded creds; 407 responses; a Tier-1 WebRTC probe showing a non-proxy IP; DNS/geo mismatch; proxy works but site still geo-blocks.

**Phase to address:** **Phase 2 (Proxy support)** for auth+DNS; **Phase 4 (WebRTC hardening)** for the leak prevention; both add probes to the **Phase 1 harness**.

---

### Pitfall 8: Per-session config baked at launch in a SHARED daemon browser

**What goes wrong:**
The milestone promises **per-session** proxy and fingerprint. But rod-cli's architecture is a **persistent daemon** that boots one `*rod.Browser` on the first command and reuses it (`initial()` only launches if `ctx.browser == nil`). Proxy is a **launch-time** flag (`browserLauncher.Proxy`), and the fingerprint is generated inside `createPage()`. So: a second command with a *different* `--proxy` either is silently ignored (browser already running with the old proxy) or — worse — the new fingerprint/headers apply to a browser still routing through the old proxy. Config **leaks between sessions** or requires a full relaunch that the daemon model is explicitly designed to avoid.

**Why it happens:**
Proxy is fundamentally a browser-launch (or per-context) property in Chromium, while fingerprint/headers are per-page in rod-cli's code. The daemon's whole value prop is *not relaunching*. These collide: you cannot change a launch-time proxy on a running browser without either a new browser, a new BrowserContext, or per-request routing.

**How to avoid:**
- **Decide and document the config model explicitly** before building: which knobs are per-page (UA, headers, JS evasions, WebRTC, canvas — all already per-page via `createPage`), which require a new **BrowserContext** (proxy, ideally), and which truly require relaunch.
- Prefer **per-incognito-BrowserContext proxy** (`CreateBrowserContext` with proxy) so a named session (`-s`) gets its own proxy without killing the shared browser — this matches the existing `-s` named-session model.
- If a knob can't be changed on a live daemon, **fail loudly** ("proxy change requires `kill-all` / new session") rather than silently using the old value.
- **Add a session-isolation test:** start session A with proxy/fingerprint X, session B with Y, assert each page reports its own — no bleed. This is the daemon-specific integration gate.

**Warning signs:**
`--proxy` on a 2nd command has no effect; fingerprint identical across `-s` sessions when it shouldn't be; proxy set via launcher flag with no per-context path; no test that runs two concurrent sessions with different config.

**Phase to address:** **Phase 2 (Proxy)** and **Phase 3 (Fingerprints)** must each resolve the daemon-shared-state question; a **session-isolation integration test** spans both. Flag as architectural — likely needs design discussion in discuss-phase.

---

## Moderate Pitfalls

### Pitfall 9: Timezone / locale vs IP/geo mismatch

**What goes wrong:** UA-CH/`navigator.language` says `en-US`, JS `Intl` timezone says `America/New_York`, but the proxy egress IP geolocates to Frankfurt. Detectors cross-reference IP-geo against browser-reported timezone/locale; the mismatch is a strong bot signal.
**How to avoid:** When a proxy is set, derive (or require) timezone+locale consistent with the proxy's geo, and set them per-page (CDP `Emulation.setTimezoneOverride`, locale via launch/headers). Add a geo-consistency probe to Tier-1 using a stubbed geo.
**Warning signs:** Timezone hardcoded to host; locale fixed while proxy geo varies.
**Phase to address:** **Phase 2 (Proxy)** + **Phase 3 (Fingerprint)** jointly.

### Pitfall 10: Detection suites drift over time and silently rot the harness

**What goes wrong:** CreepJS/sannysoft/fingerprintjs evolve; vendored probes become stale, so Tier-1 passes while real-world detection has moved on — false confidence.
**How to avoid:** Version-pin vendored probes with provenance notes; schedule the Tier-2 (live, non-blocking) job to surface drift; treat a Tier-1/Tier-2 divergence as a signal to refresh probes via reviewed PR.
**Warning signs:** Probe set untouched for months while Tier-2 live results worsen.
**Phase to address:** **Phase 1 (Harness)** designs for refreshability.

### Pitfall 11: Token-efficient output regression (carried-forward retrospective lesson)

**What goes wrong:** Adding proxy/fingerprint/validation features re-introduces chatty default output (status lines, "stealth applied", proxy banners) — exactly the verbosity the v1.5 retrospective spent round-trips removing (the startup banner). An LLM/pipe caller gets noise.
**How to avoid:** Default to **quiet**; gate any decorative/diagnostic stealth output behind `--verbose`/TTY check or `--json`. Validate with the actual pipe path (`--raw`/`--json`), not a TTY. Make "no new default stdout noise" a definition-of-done line on every phase.
**Warning signs:** New `fmt.Println` in hot paths; banners on proxy/fingerprint set; output diff vs prior version in `--raw` mode.
**Phase to address:** **Every phase** (cross-cutting DoD); explicitly re-checked at milestone verify.

### Pitfall 12: Automation-flag tells in the launcher

**What goes wrong:** Existing launch flags are themselves signals: `ignore-certificate-errors`, `--disable-features=HttpsUpgrades`, `disable-xss-auditor`, `--remote-allow-origins=*`, `no-gpu`. Some correlate with automation/insecure configs and can be probed (e.g. accepting bad certs, downgraded HTTPS, missing GPU/WebGL). `no-gpu` in particular can force a software-renderer WebGL string that contradicts a spoofed GPU.
**How to avoid:** Audit each flag for fingerprint/security impact; drop or gate the ones not strictly required for stealth use; reconcile `no-gpu` with the WebGL spoof story (Pitfall 5).
**Warning signs:** WebGL renderer = SwiftShader/software while UA claims a real GPU; sites behaving differently due to accepted bad certs.
**Phase to address:** **Phase 3/4** (fingerprint + WebGL), audited against the harness.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Assert on Go Config/Profile fields instead of live `page.Eval` | Fast, no browser in test | Wired-but-silent evasions ship green (the `onDOMNodeInserted` trap) | **Never** for stealth assertions |
| Live third-party suite in blocking CI | "Most authoritative" feeling | Flaky red CI, ignored signals, the exact retrospective burn | **Never** blocking; OK as Tier-2 non-blocking |
| Random fingerprint per page (current `fg.Generate()`) | Zero config | Entropy spike, per-session inconsistency, un-pinnable identity | Only until Phase 3 pins per-session identity |
| Hardcoded Client-Hints version `"121"` | Quick header | UA/UA-CH mismatch = top-tier bot flag | **Never** — derive from UA |
| `--proxy-server` with embedded creds | One-line proxy | No auth, 407s, dialog hangs in daemon | **Never** for authed proxies |
| Validate headful only | Easy local run | Headless tells ship undetected | **Never** — matrix both |
| `_ = em.Apply()` / swallow fingerprint err | Compiles clean | Silent stealth no-op | **Never** — surface/log |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Chromium proxy auth | Embed creds in `--proxy-server` URL | Handle CDP auth-required event with credentials |
| WebRTC + proxy | Assume proxy flag covers WebRTC | Force WebRTC IP-handling policy; probe for host-IP leak |
| DNS + proxy | Let DNS resolve on host | Resolve through proxy; probe geo match |
| Daemon-shared browser | Set proxy as launch flag, reuse browser | Per-BrowserContext proxy keyed to `-s` session |
| CDP (go-rod) | Assume JS spoofing hides automation | Probe CDP/Runtime.enable tells separately; accept partial ceiling |
| UA-CH headers | Hardcode version, set in different file than JS spoof | Single fingerprint source of truth → all surfaces |
| Canvas/WebGL noise | Uniform random noise per read | Stable per-session, plausible-range, validate vs CreepJS "lies" |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Per-page fingerprint regen | CPU/latency on every navigation; instability | Generate once per session, reuse | As soon as a session opens many pages |
| Re-launching browser to change proxy | Slow commands, defeats daemon | Per-context proxy, no relaunch | First multi-proxy / `-s` workflow |
| Always-on Runtime/Network domains for logging | Larger CDP surface + overhead every page | Make console/request logging opt-in | Every page, even when logs unused |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Proxy creds logged or echoed | Credential leak in CLI output/logs | Redact creds; never print proxy URL with auth |
| WebRTC real-IP leak | De-anonymizes the user behind the proxy | Force WebRTC policy + leak probe in CI |
| `ignore-certificate-errors` always on | MITM/bad-cert acceptance + automation tell | Gate behind explicit opt-in flag |
| Session config bleed in shared daemon | Session A's proxy/identity leaks to session B | Per-context isolation + isolation test |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Silently ignoring a `--proxy`/fingerprint change on a running daemon | Agent thinks config applied; it didn't | Fail loudly with remediation (`kill-all`/new `-s`) |
| Chatty stealth/proxy banners by default | Token waste, pollutes `--raw`/pipe output | Quiet default; diagnostics behind `--verbose`/TTY |
| Claiming "undetectable" in help/docs | False guarantee; user gets blocked, loses trust | State best-effort + what's proven (Tier-1) vs not (CDP/WAF) |

## "Looks Done But Isn't" Checklist

- [ ] **Fingerprint spoof:** asserted via live `page.Eval` of `navigator.*`/WebGL, not Go fields — verify webdriver/plugins/UA/WebGL/permissions read clean *in the page*.
- [ ] **Client Hints:** UA-CH version == UA Chrome major version (no hardcoded `121`) — verify `Sec-CH-UA` vs UA agree.
- [ ] **Headless parity:** harness green in BOTH `Headless: true` and `false` — verify the CI matrix has both rows.
- [ ] **Proxy auth:** authenticated proxy actually connects (no 407/dialog) — verify against a local auth-proxy fixture.
- [ ] **WebRTC:** no host IP leaks with proxy on — verify the leak probe.
- [ ] **DNS/geo:** resolved geo matches proxy; timezone/locale agree — verify geo-consistency probe.
- [ ] **Per-session isolation:** two `-s` sessions with different proxy/fingerprint don't bleed — verify concurrent isolation test.
- [ ] **CDP tells:** harness includes a protocol-layer probe; ceiling documented — verify it's measured, not assumed.
- [ ] **Output:** no new default stdout noise in `--raw`/`--json` — verify output diff.
- [ ] **Consistency invariant:** UA/UA-CH/platform/WebGL tell one OS story — verify the invariant test is blocking.

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Flaky live-site CI (P1) | LOW | Move live suites behind `-tags=live`/nightly; build offline Tier-1 |
| Source-asserted stealth (P2) | MEDIUM | Rewrite assertions to read from live page; re-validate every evasion |
| Fingerprint inconsistency (P5) | MEDIUM | Introduce single source of truth; regenerate all surfaces; add invariant test |
| Proxy can't change on daemon (P8) | HIGH | Re-architect to per-BrowserContext proxy; may touch session lifecycle |
| CDP detectable (P3) | HIGH / maybe unwinnable | Spike feasibility; if no fix, document ceiling, scope down claims |
| Over-spoof entropy (P6) | LOW | Pin per-session identity; remove/replace detectable noise |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| P1 Flaky live CI | Phase 1 Harness | No 3rd-party network in blocking tests; offline probe server runs |
| P2 Source-not-live asserts | Phase 1 + all evasion phases | Every stealth test reads via `page.Eval` |
| P3 CDP tells | Phase 1 probes + CDP spike phase | Protocol-layer probe present; ceiling documented |
| P4 Headful/headless gap | Phase 1 Harness | CI matrix has headless + headful rows |
| P5 Fingerprint inconsistency | Phase 3 Fingerprints | Consistency invariant test blocks merge; no hardcoded version |
| P6 Over-spoof entropy | Phase 4 Canvas/WebGL | Per-session hash stability + plausibility checks |
| P7 Proxy auth/WebRTC/DNS leak | Phase 2 Proxy + Phase 4 WebRTC | Auth-proxy fixture passes; WebRTC/DNS leak probes clean |
| P8 Daemon-shared config bleed | Phase 2 + Phase 3 | Concurrent `-s` session-isolation test |
| P9 Timezone/geo mismatch | Phase 2 + Phase 3 | Geo-consistency probe |
| P10 Suite drift | Phase 1 Harness | Pinned probes + Tier-2 drift job |
| P11 Output verbosity regression | Every phase (DoD) | `--raw`/`--json` output diff clean |
| P12 Launcher flag tells | Phase 3/4 | Flags audited vs harness; `no-gpu`/WebGL reconciled |

## Sources

- DataDome — *How New Headless Chrome & the CDP Signal Are Impacting Bot Detection* (https://datadome.co/threat-research/how-new-headless-chrome-the-cdp-signal-are-impacting-bot-detection/) — HIGH
- Rebrowser — *How to fix Runtime.Enable CDP detection* (https://rebrowser.net/blog/how-to-fix-runtime-enable-cdp-detection-of-puppeteer-playwright-and-other-automation-libraries) — HIGH
- Castle.io — *Why a classic CDP bot detection signal suddenly stopped working* (https://blog.castle.io/why-a-classic-cdp-bot-detection-signal-suddenly-stopped-working-and-nobody-noticed/) — HIGH
- Castle.io — *How to detect Headless Chrome bots instrumented with Puppeteer* (https://blog.castle.io/how-to-detect-headless-chrome-bots-instrumented-with-puppeteer-2/) — HIGH
- svebaa — *How V8 Leaks Your Headless Browser's Identity* (https://svebaa.github.io/personal/blog/cdp-fingerprinting/) — MEDIUM
- Wilico — *Bot Detection Triggered by Fingerprint Mismatches (UA vs UA-CH)* (https://wilico.co.jp/en/blog/browser-fingerprint-inconsistency-detection-consistency-check) — HIGH
- ScrapeOps — *Most "Stealth Browser" APIs Fail Browser Fingerprinting Tests [April 2026]* (https://scrapeops.io/proxy-providers/stealth-browser-fingerprint-benchmark/) — HIGH (current benchmark; sets the realistic ceiling)
- Undetectable.io — *CreepJS: Browser Fingerprinting Tests* (https://undetectable.io/blog/creepjs-browser-fingerprint-test/) — MEDIUM
- Sendwin — *WebGL Renderer Fingerprinting Protection (2026)* (https://blog.send.win/webgl-renderer-fingerprinting-protection-complete-guide-2026/) — MEDIUM
- Direct code read: `types/context.go` (launchBrowser/createPage/updateInterceptorRules), `types/config.go` (Config: launch-time Proxy, no auth/WebRTC fields) — HIGH
- Project retrospective: v1.5 wired-but-silent `onDOMNodeInserted`, validate-live-not-source, quiet-output lessons; v1.3 flaky page-load failures fixed by `retry.Fetch` — HIGH

---
*Pitfalls research for: proving & extending browser stealth in a Go daemon CLI (rod-cli/godoll)*
*Researched: 2026-06-24*
