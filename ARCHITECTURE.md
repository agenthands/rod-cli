# rod-cli Architecture

This document provides a high-level overview of the `rod-cli` system architecture,
explaining how a simple CLI command translates into stealth-enabled browser
automation. For the deeper map the team keeps while working on the code, see
`.planning/codebase/`.

## 1. The Daemon Model

A CLI is ephemeral — it runs and exits — but a browser session must persist
between calls. `rod-cli` solves this with a **client/daemon split**:

1. You run a command (e.g. `rod-cli open https://example.com`). This process is
   the **client**.
2. The client (`runClientCommand`, `cmd.go`) checks whether a background daemon
   for the current `--session` is already up by pinging it
   (`daemon.ClientExecute` → `Request{Command: "ping"}`).
3. If no daemon answers, the client spawns a detached copy of itself in daemon
   mode (`daemon.EnsureDaemon` → `exec.Command(exePath, "--session", <s>, <flags…>, "daemon")`).
4. The daemon (`runDaemonServer`, `main.go`) loads config, **resolves the stealth
   surface once** (see §3), launches Chromium via `godoll`, and serves a small
   JSON-over-HTTP protocol on a per-session loopback socket
   (`daemon.StartServer`).
5. The client sends its request to the daemon (`daemon.ClientExecute`); the
   daemon runs the action against the live browser and returns the result, which
   the client prints (plain, `--raw`, or `--json`).

Browser state, cookies, and memory stay intact in the background daemon across
sequential client invocations.

### Multiplexing by session

`--session <name>` (default `default`) selects the daemon. Each named session is
its **own** daemon process with its own Chromium instance — so sessions are fully
isolated, and stealth config set on one session never bleeds into another.

## 2. Godoll Engine Integration

`rod-cli` is built on the [`godoll`](https://github.com/agenthands/godoll)
framework (a wrapper around `go-rod`). The daemon's `types.Context`
(`types/context.go`) owns the `rod.Browser`/`rod.Page` lifecycle and wires godoll
in at three points:

- **Stealth / evasion** — `godoll/stealth.EvasionManager`. `createPage` builds the
  active `stealth.Profile` from the resolved config (the single identity source),
  applies the evasion JS, threads the per-session noise seed, and conditionally
  wraps WebRTC. See §3.
- **Network interception** — `godoll/network.NewInterceptor`. The daemon holds a
  user-defined route table (`AddRoute`/`RemoveRoute`/`GetRoutes`) applied
  alongside the stealth layer so request mocking does not break evasion.
- **Resilient actions** — `click`, `type`, `fill`, `scroll`, navigation, etc. in
  `actions/actions.go` use godoll's retry/auto-wait and the humanize helpers
  rather than raw `go-rod` primitives.

Console (`Runtime.consoleAPICalled`) and request (`Network.requestWillBeSent`)
logging are wired here too — which is why the CDP Runtime/Network domains are
enabled, a footprint documented in [docs/cdp-footprint.md](docs/cdp-footprint.md).

## 3. The Stealth Config Spine (v1.6)

v1.6 turned stealth from "godoll defaults applied on every page" into a
**configurable, session-persistent, single-source-of-truth** surface. The spine
is deliberately funneled through one resolver so every knob shares the same
precedence and validation.

```
client flags (cmd.go)                  daemon (main.go)            types/context.go
────────────────────                   ─────────────────          ────────────────
--proxy --profile --user-agent …  ──▶  StealthFlags          ──▶  ResolveStealth(cfg, flags)
(forwarded into the daemon argv         (captured off the           │  precedence + validation,
 at spawn — non-secret only)            cli.Context)                │  resolved ONCE
--proxy-auth ───(out-of-band env)──▶  ROD_CLI_PROXY_AUTH           ▼
                                                              NewContext freezes cfg
                                                                    │
                                                              createPage → profileFromStealth
                                                                    │  → stealth.Profile (the
                                                                    ▼     single identity source)
                                                              EvasionManager.Apply() + EvadeWebRTC()
```

Key properties, all grounded in `types/config.go` and `types/context.go`:

- **Resolved once, at spawn.** `ResolveStealth` runs in `runDaemonServer` *before*
  `NewContext` freezes `Config`. A stealth flag only "sticks" for a session if it
  is present at spawn; passing one to an already-running session warns on stderr
  and is otherwise inert (close the session to re-apply).
- **Precedence: CLI flag > profile file > built-in default.** A `--profile`
  overlays a saved `stealth.Profile` JSON; explicit flags win over it; unset
  fields fall to godoll's `DefaultProfile()`.
- **Single identity source of truth.** `profileFromStealth` builds the active
  `stealth.Profile` from the resolved `cfg.Stealth`; the UA is the derivation
  anchor (its Chrome major version drives Client-Hints + `userAgentData`; its OS
  token drives `navigator.platform`). A randomly generated fingerprint is used
  *only* for the dimensions godoll still needs (e.g. WebGL VideoCard), never as
  the identity.
- **Fail loud, fail early.** `deriveAndValidateFingerprint` and
  `validateHumanizeTuning` run at the spawn seam: an incoherent identity (e.g.
  `--platform` contradicting the UA OS) or an out-of-range humanize value aborts
  the daemon with a clear error instead of shipping a mismatched lie or panicking
  per-keystroke deep inside a live session.
- **Hardened by default.** WebRTC leak protection and stable canvas/WebGL/audio
  noise are on by default (`*bool` fields so an explicit `false` in a config file
  survives resolution rather than being re-baselined to true).
- **Credentials never hit argv.** `--proxy-auth` is passed to the daemon via the
  `ROD_CLI_PROXY_AUTH` environment variable (argv is world-readable via
  `/proc/<pid>/cmdline`); URL-embedded proxy credentials are rejected loudly and
  proxy auth is performed via CDP `Fetch.continueWithAuth`, never URL-embedded.

The full flag catalog, the proxy model, the fingerprint consistency rules, the
hardening toggles, and the humanize-tuning knobs (with their honest constraints)
live in **[docs/stealth-config.md](docs/stealth-config.md)**.

## 4. Validating Stealth — and the honest ceiling

`rod-cli` does **not** claim to be "undetectable." It validates and hardens the
JavaScript / fingerprint layer it actually controls, and is explicit about the
layers (TLS/JA3-JA4, IP reputation, the CDP transport) it cannot. Validation has
two tiers:

- **Tier 1 — offline harness** (`internal/detect/`, `tests/detection_test.go`):
  deterministic, zero-egress, **blocking** in CI. Drives the real binary against
  an embedded `127.0.0.1` detection fixture and asserts each fingerprint signal
  by reading it back *from the live page*.
- **Tier 2 — live smoke check** (`tests/detection_live_test.go`, `//go:build
  detection_live`): opt-in, best-effort, **non-blocking**, excluded from CI by
  the build tag. Drives Cloudflare/DataDome/CreepJS and reports informationally.

The **`stealth-check`** command (`actions/stealth_check.go`) lets a user run the
same per-signal probe against any page, with `--raw` for a token-efficient
machine-readable verdict. Command and harness share one probe source
(`internal/detect/probe.js`, embedded as `detect.Probe`).

See **[docs/stealth-validation.md](docs/stealth-validation.md)** for what each
tier proves and the full honest-ceiling discussion, and
**[docs/cdp-footprint.md](docs/cdp-footprint.md)** for the CDP tell.

## 5. Invoking with no command

Running `rod-cli` with no arguments prints the banner, description, and full
command list (see `--help`). The tool is driven entirely through explicit
subcommands; the only long-running process is the per-session daemon spawned on
demand.
