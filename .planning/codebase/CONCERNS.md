# Concerns

**Mapped:** 2026-06-18
**Refreshed:** 2026-06-25 (milestone v1.6 close)

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
   fingerprint layer only; TLS (JA3/JA4), IP reputation, and the CDP transport are
   out of scope (v2: CDP-01, TLS-01). See docs/stealth-validation.md +
   docs/cdp-footprint.md. (This phase corrected an overclaim in root
   ARCHITECTURE.md.)

6. **`Config.Proxy` is a deprecated shim.** It is bridged from `Stealth.Proxy` by
   `ResolveStealth`; prefer `Stealth.Proxy` everywhere. Removing the bridge needs
   the launchBrowser call site updated.

## Known honest constraints (documented limitations, not bugs)
- No dedicated "delay jitter" knob — it IS the `--typing-speed-min/max` spread.
- `--scroll-physics=false` cannot disable physics — godoll's `WithPhysics()` only
  enables, and physics is godoll's default. Needs a godoll signature change.

## Other
- **Concurrency:** a session daemon holds one shared `types.Context`/browser; the
  protocol serializes commands per session, so cross-talk is bounded — but the
  one-context-per-session model is worth keeping in mind for any parallel-within-
  session feature.
- **godoll is a local `replace` dep (`../godoll`).** Do NOT edit `../godoll` from
  this repo; it is a separate repository.
