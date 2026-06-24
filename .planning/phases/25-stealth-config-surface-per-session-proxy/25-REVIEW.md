---
phase: 25-stealth-config-surface-per-session-proxy
reviewed: 2026-06-24T06:25:04Z
depth: deep
files_reviewed: 7
files_reviewed_list:
  - types/config.go
  - types/context.go
  - cmd.go
  - main.go
  - tests/proxyfixture/server.go
  - tests/proxy_test.go
  - daemon/daemon.go
findings:
  critical: 1
  warning: 5
  info: 4
  total: 10
status: findings
---

# Phase 25: Code Review Report — Stealth Config Surface & Per-Session Proxy

**Reviewed:** 2026-06-24T06:25:04Z
**Depth:** deep (cross-file: rod-cli → godoll/browser proxy relay, daemon spawn boundary)
**Files Reviewed:** 7
**Status:** findings

## Summary

The phase delivers a clean, well-documented precedence resolver (`ResolveStealth`),
a correct URL-credential-stripping parser (`parseProxyConfig`), and disciplined
per-session relay cleanup. Most of the credential-safety design is genuinely
sound: the `.port` file only ever receives `fmt.Sprint(port)`, `parseProxyConfig`
does no logging, the URL-embedded-cred stripping is correct (Go's `url.Parse`
parks userinfo in `u.User`, leaving `u.Host` credential-free), `--proxy-auth` is
suppressed from the already-running stderr warning, and the precedence (CLI >
profile > default, resolved once) is implemented exactly as claimed with a loud
failure on a bad profile.

However, the central credential-safety claim — "credentials never leak anywhere"
— has a real hole the tests do not cover: **`--proxy-auth` is forwarded verbatim
as a daemon process argument**, making the cleartext `user:pass` visible to any
local user via `/proc/<pid>/cmdline` / `ps` for the daemon's lifetime. This is
the highest-priority finding. Several secondary credential-exposure and
correctness gaps follow.

The structural pre-pass was not provided; all findings below are narrative.

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01: Proxy credentials leak into the daemon process argv (visible to other local users)

**File:** `cmd.go:37`, `daemon/daemon.go:99-102`
**Issue:** `--proxy-auth <user:pass>` is appended to the spawn `flags` slice in
`runClientCommand` and forwarded verbatim by `EnsureDaemon` into
`exec.Command(exePath, args...)`. The daemon therefore runs with
`rod-cli --session <s> --proxy <url> --proxy-auth user:pass daemon` as its
command line. On Linux with the default `hidepid=0`, `/proc/<pid>/cmdline` (and
hence `ps -ef`, `ps aux`) exposes that cleartext credential to **every other user
on the host** for the entire daemon lifetime (up to the 15-minute idle window, or
indefinitely while in use). This directly contradicts the field doc
(`types/config.go:43-46`: "never logged or persisted ... Credential-sensitive")
and the phase's stated credential-safety guarantee. It is the one credential
vector the `TestProxyCredentialsNotLeaked` test does **not** check (it scans
folded stdout/stderr and the `.port` file, never `/proc/self/cmdline`), so it
passes vacuously with respect to this leak.

**Fix:** Do not pass the secret on the command line. Forward `--proxy-auth` to
the daemon out-of-band — e.g. via an environment variable on the spawned process
(`cmd.Env = append(os.Environ(), "ROD_CLI_PROXY_AUTH="+auth)`) which
`runDaemonServer` reads instead of `c.String("proxy-auth")`, or write it to a
0600 file the daemon reads-then-unlinks. Environment is still readable via
`/proc/<pid>/environ` but only by the owning user (and root), unlike `cmdline`
which is world-readable by default. Example sketch:

```go
// cmd.go — stop forwarding the secret as an argv flag:
//   if c.String("proxy-auth") != "" { flags = append(flags, "--proxy-auth", c.String("proxy-auth")) }   // REMOVE
// Instead thread it through EnsureDaemon's env (new param) so it lands in cmd.Env, not cmd.Args.

// main.go runDaemonServer:
proxyAuth := c.String("proxy-auth")
if proxyAuth == "" {
    proxyAuth = os.Getenv("ROD_CLI_PROXY_AUTH")
}
stealthFlags := types.StealthFlags{Proxy: c.String("proxy"), ProxyAuth: proxyAuth, Profile: c.String("profile")}
```

Add a regression test that reads `/proc/<daemon-pid>/cmdline` (Linux) and asserts
the password byte-sequence is absent.

## Warnings

### WR-01: `proxyAuth` is a first-class persisted YAML field — invites cleartext credentials on disk

**File:** `types/config.go:46` (`ProxyAuth string \`yaml:"proxyAuth" json:"proxyAuth"\``), `types/config_test.go:108-135`
**Issue:** The field comment says ProxyAuth is "never logged or persisted to
state/port files," but it carries a `yaml:"proxyAuth"` tag and
`TestLoadConfig_StealthBlockRoundTrips` explicitly asserts it round-trips through
`rod-cli.yaml`. That makes persisting the secret in a plaintext config file a
fully-supported, tested path. Worse, `InitDefaultConfig` encodes the whole
`DefaultConfig` (including an empty `proxyAuth:` key) into a generated
`rod-cli.yaml` in the user's CWD, normalizing the idea that credentials belong in
that file. The config file is written 0644 (`InitDefaultConfig` via `os.Create`
default), so any populated value is world-readable.
**Fix:** Either drop the YAML serialization of the secret
(`yaml:"-" json:"-"` on `ProxyAuth`, keeping only the in-memory CLI/env path), or
keep it loadable but (a) document loudly that it is plaintext, (b) refuse to write
it back out in `InitDefaultConfig`, and (c) chmod any config containing it to
0600. Given the field doc's own "never persisted" promise, `yaml:"-"` is the
consistent choice.

### WR-02: `socks5`/`socks4` + `--proxy-auth` is silently mishandled (relay speaks HTTP CONNECT to a SOCKS upstream)

**File:** `types/context.go:65-92` (parse accepts socks5/4 with auth), cross-ref `godoll/browser/proxy.go:47-61, 92-117, 139-156`
**Issue:** `parseProxyConfig` produces a `ProxyConfig{Protocol:"socks5", ...}`
with `Username/Password` set, and `--proxy` help text advertises socks5. But in
the authenticated path `ProxyConfig.ApplyToLauncher` ignores `Protocol` entirely:
it starts `StartProxyRelay`, which unconditionally dials the upstream and writes
an **HTTP** `CONNECT ... Proxy-Authorization: Basic ...` request
(`proxy.go:148-150`). Pointed at a real SOCKS5 proxy this is protocol garbage —
the upstream will reject or hang, and the relay's "contains 200" check
(`proxy.go:166`) fails with an opaque "upstream rejected" 502. The user gets a
confusing failure for a flag combination the CLI advertises as supported. (Root
cause is in godoll, but rod-cli is the layer that accepts and forwards the
unsupported combination.)
**Fix:** In `parseProxyConfig` (or before `ApplyToLauncher`), reject
`socks5`/`socks4` together with non-empty `proxyAuth` with a clear error
("authenticated SOCKS proxies are not supported; use an HTTP proxy for
--proxy-auth"), until godoll's relay learns the SOCKS auth handshake. At minimum,
fix the `--proxy` help text to state auth is HTTP-only.

### WR-03: `parseProxyConfig` silently drops URL-embedded credentials instead of failing loud

**File:** `types/context.go:77-83`
**Issue:** When a user passes `--proxy http://user:pass@host:port`, the parser
deliberately discards `u.User` and proceeds with no proxy auth at all (the
comment documents this as intentional). Stripping the creds out of
`--proxy-server` is correct and necessary, but *silently* dropping them means a
user who supplied working credentials this way gets an **unauthenticated**
connection that will 407 against the proxy, with no hint why. The credential was
also briefly present in the daemon argv (see CR-01) yet had no effect — worst of
both worlds.
**Fix:** When `u.User != nil`, either (a) lift the embedded creds into
`Username/Password` if `proxyAuth` was not separately supplied (so the URL form
"just works" and still never reaches `--proxy-server`), or (b) return a loud
error directing the user to `--proxy-auth`. Do not silently no-op.

### WR-04: Daemon log file is world-readable/writable (0666) and captures relay logging

**File:** `daemon/daemon.go:103`
**Issue:** `os.OpenFile(.../rod-cli-daemon.log, ..., 0666)` creates a
world-writable, world-readable log in a shared temp dir, and the daemon's
stdout+stderr are redirected into it. godoll's proxy relay logs into that stream
(`godoll/browser/proxy.go:55,140,143,168`: relay/upstream addresses and CONNECT
target hosts). While the current relay lines do **not** print the password, this
is a shared sink for daemon diagnostics that an attacker can both read (browsing
history via CONNECT hosts, proxy addresses) and tamper with (0666 write). It also
sits one careless `log.Printf("auth=%s")` away from a credential leak. Tightening
this is cheap insurance for the credential-handling story.
**Fix:** Create the log `0600`. Consider whether the daemon should inherit
godoll's default-logger output at all, or route it through a logger the daemon
controls and can scrub.

### WR-05: Credential-leak test does not exercise the real leak surfaces (argv) and can pass vacuously

**File:** `tests/proxy_test.go:264-300`
**Issue:** `TestProxyCredentialsNotLeaked` points `--proxy` at
`http://127.0.0.1:59999` (an assumed-dead fixed port) and scans only the folded
client stdout/stderr and the `.port` file. It never inspects the daemon's
`/proc/<pid>/cmdline` (the actual leak, CR-01) nor the shared
`rod-cli-daemon.log` (WR-04). Because the parser never logs and the `.port` file
is port-only by construction, the assertions it does make can essentially never
fail — the test "proves" safety on channels that were never at risk while
ignoring the channel that is. If port 59999 happens to be bound by something
else, behavior is also nondeterministic.
**Fix:** Add assertions against `/proc/<daemon-pid>/cmdline` (gate on
`runtime.GOOS == "linux"`) and against the contents of `rod-cli-daemon.log` for
the password substring. Bind the dead-proxy fixture on `127.0.0.1:0` and close it
to get a guaranteed-unused port rather than hardcoding 59999.

## Info

### IN-01: `ResolveStealth` loads and discards the profile — no validation, dead `DefaultProfile()` call

**File:** `types/config.go:124, 129-132`
**Issue:** `_ = stealth.DefaultProfile()` (line 124) is a pure no-op kept only as
intent signaling, and `stealth.LoadProfile(path)` is called for its error side
effect but the returned `*Profile` is discarded. `LoadProfile` only does
`json.Unmarshal`, so an empty `{}` or a JSON file with garbage-but-valid fields
"loads" successfully and `cfg.Stealth.ProfilePath` is recorded as if valid. The
"loud failure on bad profile" guarantee therefore only covers missing files and
malformed JSON, not semantically empty/invalid profiles. Acceptable for Phase 25
(identity overlay is deferred to Phase 26) but the gap should be a known TODO.
**Fix:** Drop the dead `DefaultProfile()` call, or add a comment that profile
*content* validation lands in Phase 26. Consider a minimal non-empty check (e.g.
require a UserAgent) so a blank profile fails at selection time.

### IN-02: `resolveProfilePath` treats any CWD file matching the bare name as a profile

**File:** `types/config.go:97-99`
**Issue:** A bare `--profile foo` does an `os.Stat("foo")`; if a file literally
named `foo` exists in the current directory it is used verbatim, shadowing the
intended `~/.rod-cli/profiles/foo.json`. Low-likelihood surprise, but it makes
profile resolution CWD-dependent.
**Fix:** Resolve bare names from the profiles dir first and only fall back to a
CWD/relative stat if that miss, or document the CWD precedence explicitly.

### IN-03: `RandomString` uses time-seeded `math/rand` for per-session temp-dir isolation

**File:** `utils/str.go:10-18` (used at `types/context.go:113`)
**Issue:** Per-session browser temp dirs rely on `RandomString(10)` for
uniqueness/isolation, but it calls `rand.Seed(time.Now().UnixNano())` on every
call and uses the global `math/rand`. Two daemons started within the same
nanosecond-resolution window (or after a re-seed) could collide, and the value is
predictable. Pre-existing (not introduced by Phase 25) and not a direct
credential issue, but the session-isolation claim leans on it.
**Fix:** Use `crypto/rand` for the temp-dir suffix, or seed once at package init.
Low priority for this phase.

### IN-04: Minor lint nits (LOW, as scoped)

**File:** `types/context.go:11` interface usage elsewhere; `cmd.go:128` `map[string]interface{}`
**Issue:** `interface{}` could be `any`; a couple of single-statement `if`
bodies on one line in `runClientCommand` (`cmd.go:28-38`) hurt readability. These
are the known low-severity nits called out in the review brief.
**Fix:** `gofmt`/`golangci-lint` sweep; `interface{}` → `any`. Non-blocking.

---

_Reviewed: 2026-06-24T06:25:04Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
