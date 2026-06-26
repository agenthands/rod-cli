# v1.6 stealth config spine (rod-cli) — map at milestone close

rod-cli is a **CLI + per-session daemon** (NOT an MCP server — old map was stale;
no mark3labs/mcp-go dep). Built on godoll (`replace => ../godoll`, separate repo —
do NOT edit). Go 1.25.1.

## The stealth spine (single funnel)
`cmd.go runClientCommand` forwards non-secret stealth flags into daemon spawn
argv (proxy-auth out-of-band via `ROD_CLI_PROXY_AUTH` env — argv is world-
readable) → `main.go runDaemonServer` captures into `types.StealthFlags` →
`types/config.go ResolveStealth(cfg,flags)` resolves ONCE (precedence: CLI flag >
profile file > default) + validates → `NewContext` freezes Config →
`types/context.go profileFromStealth` builds the single `stealth.Profile` →
`createPage` applies via godoll `EvasionManager` (`SetProfile`/`Apply`/`EvadeWebRTC`/
`SetNoiseSeed`).

## Invariants / landmines (verify before editing)
1. Stealth resolves ONCE at spawn. New knob = flag(cmd.go) + capture(main.go) +
   resolve(ResolveStealth), else it never "sticks" per-session.
2. Hardening (`WebRTCLeakProtection`,`CanvasNoise`) + ALL humanize fields are
   POINTERS: nil(unset)≠explicit false. Read toggles via `boolVal(p,default)`.
   Plain-bool reintroduces CR-02 (yaml false clobbered by baseline true).
3. Secrets never in argv. proxy-auth via env; parseProxyConfig rejects URL-
   embedded creds; CDP Fetch.continueWithAuth for auth.
4. Evasion failures = log-and-continue (stderr warn), NOT hard-abort, in
   createPage (em.Apply/EvadeWebRTC/fp gen). Deliberate (VALIDATE-03 nuance).
5. Validators run at spawn seam: `deriveAndValidateFingerprint` (UA=anchor;
   derive-when-unset/reject-when-user-conflicts) + `validateHumanizeTuning`
   (godoll rand panics on neg/min>max). UA Chrome major → CH; UA OS → platform.
6. `Config.Proxy` = deprecated shim bridged from `Stealth.Proxy`.

## Honest constraints (documented, not bugs)
- No separate "delay jitter" knob — it IS the --typing-speed-min/max spread.
- --scroll-physics=false canNOT disable physics (godoll WithPhysics only enables;
  physics is godoll default). Needs godoll sig change.
- Honest ceiling: rod-cli controls JS/fingerprint layer only. NOT TLS(JA3/JA4),
  IP rep, CDP transport (v2: CDP-01/TLS-01). Never claim "undetectable" /
  "bypasses Cloudflare". (Fixed an overclaim in root ARCHITECTURE.md this close.)

## Validation model (two tiers)
- Tier 1 offline blocking: tests/detection_test.go vs internal/detect/ fixture
  (127.0.0.1, embed). CI = `go test ./... -count=1` NO -tags.
- Tier 2 live non-blocking: tests/detection_live_test.go `//go:build
  detection_live`, excluded from CI.
- `stealth-check` cmd (actions/stealth_check.go) + harness share ONE probe:
  internal/detect/probe.js embedded as `detect.Probe`.

## Docs I own (current @ commit after 4311fa8)
- root ARCHITECTURE.md (daemon model + stealth spine + validation tiers)
- docs/stealth-config.md (NEW — config surface deep dive)
- docs/stealth-validation.md, docs/cdp-footprint.md (Phase 29/24, linked)
- .planning/codebase/ map refreshed to HEAD (was badly stale pre-v1.6).
