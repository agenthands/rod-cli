---
phase: 25-stealth-config-surface-per-session-proxy
plan: 02
subsystem: browser-launch
tags: [stealth, proxy, socks5, cdp-auth, godoll, relay, lifecycle]

# Dependency graph
requires:
  - phase: 25-stealth-config-surface-per-session-proxy
    plan: 01
    provides: StealthConfig (Proxy, ProxyAuth) on types.Config + ResolveStealth resolver frozen at daemon spawn
provides:
  - parseProxyConfig helper mapping cfg.Stealth.Proxy/ProxyAuth -> godoll browser.ProxyConfig (scheme->Protocol, host:port->Address, first-colon auth split, embedded-cred strip)
  - launchBrowser rewired to godoll ProxyConfig.ApplyToLauncher (replaces bare launcher.Proxy) with SetupBrowserAuth CDP handler for the auth case
  - per-session proxy relay lifecycle (proxyCleanup stored on Context, stopped in closeBrowser)
affects: [25-03 (end-to-end egress/isolation/auth-against-fixture asserts this wiring), 27 (WebRTC/DNS leak past proxy is HARDEN-01 territory)]

# Tech tracking
tech-stack:
  added: []  # no new third-party modules; godoll/browser already in go.mod
  patterns:
    - "Proxy parsing is pure + log-free: scheme->Protocol, host:port->Address, embedded creds stripped so they never reach --proxy-server (T-25-05)"
    - "Auth via CDP only: SetupBrowserAuth registers Browser.HandleAuth BEFORE first Page()/Navigate() so a 407 never hangs the headless daemon (T-25-06)"
    - "Relay lifecycle bound to session: ApplyToLauncher cleanup stored on Context, invoked once in closeBrowser (nil'd to guard double-call), plus a launch-failure cleanup so the listener never leaks (T-25-07)"

key-files:
  created:
    - types/context_test.go
  modified:
    - types/context.go
    - types/errpaths_test.go

key-decisions:
  - "Embedded URL credentials (http://user:pass@host) are STRIPPED, not rejected: url.Parse parks them in u.User and parseProxyConfig drops them, using only u.Host for Address. Auth flows exclusively through cfg.Stealth.ProxyAuth via CDP."
  - "A proxy URL without a scheme is a loud error (host:port is ambiguous for Protocol mapping); https normalizes to http."
  - "proxyAuth without a colon is a loud error rather than silently authing with an empty password; the password may contain colons so the split is first-colon-only (strings.Cut)."
  - "launchBrowser signature widened to (*rod.Browser, func(), error) to thread the relay cleanup out to Context.Close — the per-session daemon owns exactly one relay."

requirements-completed: [PROXY-01, PROXY-02]

# Metrics
duration: ~15min
completed: 2026-06-24
status: complete
---

# Phase 25 Plan 02: Per-Session HTTP/SOCKS5 Proxy with CDP Auth Summary

**The bare `launcher.Proxy(cfg.Proxy)` is replaced with godoll's `ProxyConfig.ApplyToLauncher` path — `cfg.Stealth.Proxy`/`ProxyAuth` parse into a credential-safe `browser.ProxyConfig`, authenticated proxies get a CDP `SetupBrowserAuth` handler before first navigation and a local relay whose cleanup is stored on `Context` and stopped on session close.**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-06-24
- **Tasks:** 2 of 2
- **Files modified:** 2 (+1 created)

## Accomplishments
- Added `parseProxyConfig(proxyURL, proxyAuth string) (*browser.ProxyConfig, error)`: scheme→Protocol (`http`/`https`→`http`, `socks5`, `socks4`), `host:port`→Address, first-colon split of `proxyAuth` into Username/Password, embedded-URL-cred strip, and loud errors for empty-scheme / unsupported-scheme / no-colon-auth / unparseable URL. Returns `(nil, nil)` for an empty proxy URL so the caller skips all wiring. Pure parsing — no logging.
- Rewired `launchBrowser` to the godoll proxy path: parse → `ProxyConfig.ApplyToLauncher(browserLauncher)` (sets `--proxy-server` for no-auth, starts a local CONNECT relay for the auth case) → `SetupBrowserAuth(browserInstance)` after `NewBrowserE` and before any `Page()`/`Navigate()` for the auth case. The bare `browserLauncher.Proxy(cfg.Proxy)` is gone; `cfg.Stealth.Proxy` is now the authoritative source.
- Threaded the relay cleanup out via a widened `launchBrowser` return signature, stored it as `Context.proxyCleanup`, and invoke it in `closeBrowser` (nil'd after, guarding double-call). Added a launch-failure cleanup so the relay listener is never leaked if `NewBrowserE` fails.

## Task Commits

Each task was committed atomically:

1. **Task 1: parse cfg.Stealth.Proxy/ProxyAuth into godoll browser.ProxyConfig** — `b061c1b` (feat, TDD)
2. **Task 2: ApplyToLauncher + SetupBrowserAuth + per-session relay cleanup** — `d0bd1b9` (feat)

_Note: Task 1 was TDD — the 10-subtest RED suite in `types/context_test.go` and the `parseProxyConfig` impl landed in one feat commit because the `types` package must compile to host the tests (same pattern as Plan 01). RED was confirmed first (undefined `parseProxyConfig` build failure) before the GREEN impl._

## Files Created/Modified
- `types/context_test.go` (created) — 10 subtests for `parseProxyConfig`: http/no-auth, socks5/auth, empty-url→nil, auth-without-url→nil, https→http normalization, embedded-cred strip (asserts `LauncherURL()` is credential-free), unparseable-URL error, missing-scheme error, first-colon auth split, no-colon auth error. Plus a compile-time type assertion that the helper returns `*browser.ProxyConfig`.
- `types/context.go` — Added `parseProxyConfig`; imported `net/url`; replaced the bare proxy block with `ApplyToLauncher` + `SetupBrowserAuth`; widened `launchBrowser` to return `(*rod.Browser, func(), error)`; added `proxyCleanup func()` to the `Context` struct; set it in `initial()`; invoke + nil it in `closeBrowser`.
- `types/errpaths_test.go` — Updated the two existing `launchBrowser` call sites for the new 3-value signature; moved `TestLaunchBrowser_ProxyAndBadBin` onto `cfg.Stealth.Proxy` (the new authoritative source); added `TestLaunchBrowser_BadProxyURL` asserting a scheme-less proxy fails loudly before launch.

## Decisions Made
- **Embedded creds stripped (not rejected):** `http://user:pass@host` parses, `u.User` is dropped, only `u.Host` reaches `Address`. The test accepts either rejection or strip, but the impl strips and verifies the resulting `LauncherURL()` carries no credentials.
- **Scheme is mandatory:** a bare `host:port` (no scheme) is a loud error since Protocol cannot be inferred; `https`→`http`.
- **`proxyAuth` must contain a colon:** no-colon is a loud error (avoids silent empty-password auth); split is first-colon-only via `strings.Cut` so passwords may contain colons.
- **Signature widened over a struct field on Context inside launch:** `launchBrowser` returns the cleanup explicitly so `initial()` stores it and `closeBrowser` owns the single stop point.

## Threat Mitigations Applied
- **T-25-05 (URL-embedded creds reaching `--proxy-server`):** `parseProxyConfig` drops `u.User`; `Address` is `u.Host` only. Test `embedded creds never reach launcher url` asserts `LauncherURL()` contains neither `user` nor `s3cret`.
- **T-25-06 (407 / native auth dialog hangs the daemon):** `SetupBrowserAuth(browserInstance)` is called after `NewBrowserE` and before the first `Page()`/`Navigate()`, registering the persistent CDP `Browser.HandleAuth` loop so the challenge is answered programmatically.
- **T-25-07 (per-session relay not stopped on close):** `proxyCleanup` is stored on `Context`, invoked once in `closeBrowser` and nil'd; a launch-failure path also calls it so the listener is never leaked.
- **T-25-08 (proxy creds in stdout/stderr/log):** `parseProxyConfig` and `launchBrowser` add no rod-cli-side logging of the proxy URL or credentials; only godoll's relay logs (upstream Address only, never credentials). Credential-leak grep of `types/context.go` matched comments and the parse code only — no log/print of the URL or auth.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Update existing launchBrowser test call sites for the new signature**
- **Found during:** Task 2
- **Issue:** `types/errpaths_test.go` called `launchBrowser` with the old 2-value signature (`go vet` assignment-mismatch), and `TestLaunchBrowser_ProxyAndBadBin` set the deprecated top-level `Config.Proxy`, which the new launch path no longer reads.
- **Fix:** Updated both call sites to the 3-value signature; moved the proxy test onto `cfg.Stealth.Proxy`; added `TestLaunchBrowser_BadProxyURL` to cover the new parse-error-through-launch path.
- **Files modified:** types/errpaths_test.go
- **Commit:** d0bd1b9

## Verification
- `go build ./...` and `go vet ./...` clean.
- `go test ./types/...` passes (full suite green, 18s; targeted `Proxy|LaunchBrowser|Close|ParseProxy` all PASS, 13/13).
- Bare `browserLauncher.Proxy(cfg.Proxy)` confirmed removed from the launch path (grep returns nothing).
- key_links satisfied: `ApplyToLauncher(browserLauncher)` (context.go:152), `SetupBrowserAuth(browserInstance)` (context.go:175), `proxyCleanup` stored on Context + invoked in `closeBrowser`.
- No proxy credential string is logged by rod-cli's own code (godoll relay logs upstream Address only).
- End-to-end egress + session-isolation + auth-against-a-fixture assertions are owned by Plan 03.

## Known Stubs
None. The proxy path is fully wired through godoll; the only deferred concern is the WebRTC/DNS real-IP leak past the proxy, which is explicitly Phase 27 (HARDEN-01) and cross-referenced in the threat model, not a stub in this plan.

## Threat Flags
None — no new security surface beyond the proxy egress already modeled in the plan's `<threat_model>`.

## Self-Check: PASSED

All created/modified files exist on disk and both task commits (`b061c1b`, `d0bd1b9`) are present in git history.
