# Testing

**Mapped:** 2026-06-18
**Refreshed:** 2026-06-25 (milestone v1.6 close)

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

## Framework & Execution
- Go's built-in `testing` package.
- CI: `.github/workflows/test.yml` runs `go test ./... -count=1` with **no
  `-tags`**, so the live tier never compiles in CI.
- Run the live tier manually:
  `go build -o rod-cli . && go test -tags detection_live ./tests/ -run TestLiveDetection -v`.
