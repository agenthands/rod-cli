# Testing

**Mapped:** 2026-06-18
**Refreshed:** 2026-06-25 (milestone v1.6 close)
**Refreshed:** 2026-06-26 (milestone v1.7 close)
**Refreshed:** 2026-06-27 (milestone v2.0 close)
**Refreshed:** 2026-06-27 (milestone v2.1 close)

> The original map claimed "lack of test files / no CI." That is **stale and
> wrong** — HEAD has ~40 `*_test.go` files and a blocking CI gate.

## Testing Strategy
- **Unit tests** colocated with packages (`types/`, `actions/`, `daemon/`,
  `internal/plugin/`, `utils/`, `banner/`) — e.g. `config_test.go`,
  `context_*_test.go`, `actions_*_test.go`.
- **End-to-end / integration** under `/tests/` drive the **built `rod-cli`
  binary** against local fixtures (proxy, navigation, interaction, stealth,
  humanize, network-evasion, …).

## The two-tier detection validation (v1.6)
- **Tier 1 — offline, blocking** (`tests/detection_test.go`): drives the real
  binary against the embedded `internal/detect/` fixture on `127.0.0.1`, zero
  egress, deterministic. Asserts each fingerprint signal by reading it back from
  the live page. This is the CI gate.
- **Tier 2 — live, non-blocking** (`tests/detection_live_test.go`, `//go:build
  detection_live`): opt-in Cloudflare/DataDome/CreepJS smoke check; excluded from
  CI by the build tag; informational only (`t.Logf`/`t.Skip`, never `t.Fatal`).

## v1.7 deterministic gates (offline, blocking)
- **CDP footprint baseline** (`types/cdp_footprint_test.go`,
  `TestCDPFootprintBaseline`): asserts a plain session records ZERO of
  {Runtime, Network, Fetch} via the `GetEnabledCDPDomains` ledger, with positive
  controls proving each opt-in feature records exactly its domain. Falsifiable
  CDP-01 gate. SCOPE CAVEAT: covers rod-cli's instrumented enable-points, not raw
  CDP wire traffic — the plugin binder path is uninstrumented (see CONCERNS #10).
- **Wire-level identity** (`tests/network_evasion_test.go`,
  `TestNetworkEvasionHeaders`): drives a plain `goto` and confirms the spoofed
  `Sec-Ch-Ua`/UA/`Accept-Language` reach the outgoing wire under the Emulation
  override (WIRE-VERIFY of the zero-footprint design).
- **Built-in profile vetting** (`types/profiles_test.go`,
  `TestBuiltinProfilesAreVetted`): runs EVERY embedded profile through the v1.6
  consistency validator; an incoherent profile fails the build (PROF-02).

## v2.0–v2.1 CDP proxy integration tests (offline, blocking)

- **Proxy traffic log** (`tests/proxy_integration_test.go`, `TestProxyTraffic`):
  launches a headless browser with `--cdp-proxy`, navigates, and asserts
  `cdp-traffic --json` returns a non-empty JSON array containing CDP messages
  with recv direction and JSON-RPC structure (PROXY-01).
- **cdpTell normalization** (`tests/proxy_integration_test.go`, `TestProxyCdpTell`):
  launches with `--cdp-proxy --console-capture`, injects a self-contained cdpTell
  probe (Error with getter on `stack`, `console.debug`), and asserts `"no-signal"`
  — confirming Runtime.getProperties normalization strips accessor getters (PROXY-02).
- **Font spoofing** (`tests/detection_test.go:1140`, `TestFontSpoof`): asserts
  `--font-spoof` ON produces a different font hash than OFF, OFF restores baseline,
  and hash is stable within session (FONT-04..07).

## Framework & Execution
- Go's built-in `testing` package.
- CI: `.github/workflows/test.yml` runs `go test ./... -count=1` with **no
  `-tags`**, so the live tier never compiles in CI.
- Run the live tier manually:
  `go build -o rod-cli . && go test -tags detection_live ./tests/ -run TestLiveDetection -v`.
