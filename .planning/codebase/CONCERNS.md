# Concerns

**Mapped:** 2026-06-18
**Refreshed:** 2026-06-25 (milestone v1.6 close)
**Refreshed:** 2026-06-26 (milestone v1.7 close)

> The original concerns ("no automated testing", "no CI") are **resolved/stale**:
> HEAD has ~40 test files and a blocking `test.yml` gate. Below are the real,
> current items, grounded in HEAD.

## Landmines & invariants (verify before editing the stealth spine)

1. **Stealth is resolved ONCE, at daemon spawn.** `ResolveStealth` runs in
   `runDaemonServer` before `NewContext` freezes `Config`. Any new stealth knob
   MUST be (a) forwarded in `cmd.go` at spawn, (b) captured in `main.go` into
   `StealthFlags`, (c) resolved in `ResolveStealth`. Adding a knob that's read
   per-command instead will silently never "stick" for a session.

2. **`*bool` / pointer "unset" semantics are load-bearing.** Hardening toggles
   (`WebRTCLeakProtection`, `CanvasNoise`) and all humanize fields are pointers so
   `nil` ("unset") is distinct from an explicit `false`/zero. Changing one to a
   plain value reintroduces the CR-02 bug: a yaml-persisted explicit value gets
   clobbered by the baseline default. Read toggles via `boolVal(p, default)`.

3. **Secrets must never reach argv.** `--proxy-auth` is intentionally passed via
   `ROD_CLI_PROXY_AUTH` (env), not the daemon argv (argv is world-readable via
   `/proc/<pid>/cmdline`). Any future credential-bearing flag must follow the same
   out-of-band path; `parseProxyConfig` also rejects URL-embedded credentials.

4. **Evasion failures are log-and-continue by design (VALIDATE-03 nuance).**
   `em.Apply()`, `EvadeWebRTC()`, and fingerprint generation in `createPage`
   warn to stderr and continue rather than aborting the daemon. This is
   deliberate (a single page's evasion hiccup shouldn't kill a session) — do not
   "fix" it into a hard failure without revisiting the requirement.

5. **The honest ceiling must stay honest.** Docs and code must NOT claim rod-cli
   is "undetectable" or "bypasses Cloudflare/DataDome." It controls the JS/
   fingerprint layer only. v1.7 status: the baseline **CDP footprint is reduced**
   (a plain session enables none of Runtime/Network/Fetch); deeper CDP-transport
   obfuscation is deferred to v2 (CDP-DEEP-01). **TLS (JA3/JA4) is deliberately
   out of scope** — rod-cli drives real Chrome so the handshake is authentic
   Chrome by construction; Phase 31 TLS spoofing was CANCELLED (it lives in a
   separate project, "munch"). IP reputation is out of scope. See
   docs/stealth-validation.md + docs/cdp-footprint.md.

6. **`Config.Proxy` is a deprecated shim.** It is bridged from `Stealth.Proxy` by
   `ResolveStealth`; prefer `Stealth.Proxy` everywhere. Removing the bridge needs
   the launchBrowser call site updated.

7. **`em.SetFingerprint` MUST run BEFORE `em.SetProfile` in createPage (v1.7).**
   `SetFingerprint` derives a profile from the fp (`FromFingerprint`) and
   overwrites `em.profile`, so `SetProfile` must run AFTER it to restore the
   config-pinned identity as the single source of truth. Reordering them silently
   makes the randomly-generated fingerprint the identity (the exact incoherence the
   pinned-identity design exists to prevent). The `em.fingerprint` stays set, which
   is what activates the dormant `applyFingerprintDimensions` path.

8. **Bare `--profile=<name>` resolves a built-in FIRST (v1.7, Phase 32).** Built-in
   names (`types/profiles/*.json` stems) are RESERVED for bare-name selection; a
   user file with a colliding name under `~/.rod-cli/profiles/` is shadowed. To use
   such a custom file pass an explicit path (separator or `.json` suffix), which
   `profileLooksLikePath` routes around the built-in lookup. `list` is reserved for
   discovery and never names a profile.

## v1.7 follow-ups (open, route UP to the architect)

9. **godoll font spoof is an observable no-op.** `stealth.scriptMockFonts`
   (`../godoll/stealth/fingerprint_bridge.go`) overrides
   `CanvasRenderingContext2D.prototype.measureText` but returns the **original**
   width in every branch, so `--font-spoof` gates injection yet does not change
   the live `measureText` readout. rod-cli enforces OS-coherence of the font *set*
   at generation, and the docs (docs/stealth-config.md font caveat) state this
   honestly. Fix requires a godoll change (out of tree — do NOT edit ../godoll
   from here). Confidence: HIGH (read the godoll source at HEAD).

10. **The plugin lifecycle binder bypasses the CDP-domain ledger.** Loading any
   plugin calls `proto.DOMEnable{}.Call(page)` and subscribes to
   `NetworkRequestWillBeSent` in `internal/plugin/lifecycle.go` `BindLifecycle`
   (unconditionally, even if the plugin defines no `onRequest`/`onDOMNodeInserted`)
   — enabling DOM + Network on the wire **without** calling `recordCDPDomainLocked`.
   So `GetEnabledCDPDomains()` under-reports for a plugin-loaded session, and the
   `TestCDPFootprintBaseline` gate only covers the instrumented enable-points, not
   raw CDP wire traffic. Candidate strengthening (v2): an `.smtc/analyzers`
   dominance spec asserting every CDP `*.enable`/subscription is dominated by a
   `recordCDPDomainLocked`, route the plugin binder through the ledger, or sniff
   real CDP commands. Confidence: HIGH (verified in lifecycle.go + context.go).

11. **Go toolchain version skew + CVE (v1.7 security F1).** `go.mod` declares
   `go 1.25.1` with no `toolchain` directive; CI `test.yml` pins `1.25.x`; the
   release workflows pin `1.23`/`1.23.7`; the dev machine builds on `go1.26.0`.
   The v1.7 security review flags bumping to the CVE-patched **`go1.26.1`** and
   aligning the declared/CI/release toolchains. This is a build/release-hygiene
   item, not a code defect. Confidence: HIGH on the skew (read go.mod + workflows);
   the specific CVE attribution is the security review's (F1).

## Known honest constraints (documented limitations, not bugs)
- No dedicated "delay jitter" knob — it IS the `--typing-speed-min/max` spread.
- `--scroll-physics=false` cannot disable physics — godoll's `WithPhysics()` only
  enables, and physics is godoll's default. Needs a godoll signature change.
- Mobile profiles are unsupported: `osForPlatform` falls back Android→linux,
  iPadOS→macos because their `navigator.platform` carries no mobile token. The
  shipped profile library is desktop-Chrome-only, so this is in-scope-correct.

## Other
- **Concurrency:** a session daemon holds one shared `types.Context`/browser; the
  protocol serializes commands per session, so cross-talk is bounded — but the
  one-context-per-session model is worth keeping in mind for any parallel-within-
  session feature.
- **godoll is a local `replace` dep (`../godoll`).** Do NOT edit `../godoll` from
  this repo; it is a separate repository.
