# v1.7 Complete Evasion Stack — Security Review

**Reviewed:** 2026-06-26 (independent security-engineer, milestone-close gate)
**Verdict:** ✅ NO BLOCKER — cleared to close. No Critical/High findings; no v1.7-introduced exploitable defect.
**Scope:** v1.7 code diff (Phases 30, 32, 33; Phase 31 cancelled) across rod-cli + vendored godoll. Evidence tier HIGH (SMTC source→sink→sanitizer trace + `go vet` + `govulncheck`).

## Threat register (key)
| Threat | Source → Sink | Status |
|---|---|---|
| T1 Profile field → JS injection | custom `--profile` JSON → godoll injected-JS literals | **DEFENDED ×2** |
| T2 Profile field → CDP header injection | same → `EmulationSetUserAgentOverride` | **DEFENDED ×2** |
| T3 Phase-33 dimensions → JS | seeded generator (not profile JSON) → scriptMock* | N/A (not attacker-controlled) |
| T4 `--profile` path → file read | operator argv → os.ReadFile | operator's own CLI; not attacker-influenceable |
| T5 Proxy-cred leak | `--proxy-auth` | **INTACT** (env-only, `yaml:"-" json:"-"`, never logged) |
| T6 Embedded-profile integrity | `//go:embed` | **INTACT** (no runtime fetch) |

**T1/T2 load-bearing result:** two independent sanitizers, either alone sufficient — (1) rod-cli `rejectUnsafeFingerprintValue` (types/config.go:605-619) rejects `"`/control chars after the custom-profile overlay and aborts the spawn; (2) godoll `json.Marshal`s every profile-controlled string into JS literals (stealth/js.go). No sink relies on a single layer.

## Findings
- **F1 — Low (fix-later):** `govulncheck` flags 13 Go stdlib vulns on go1.26.0 (called paths via daemon http.Serve + URL parse); **all fixed in go1.26.1**. Mitigated: daemon binds `127.0.0.1:0`, plain HTTP. Pre-existing, not v1.7-introduced. `go.mod` declares `go 1.25.1` while building on 1.26.0. **Action: bump toolchain to go1.26.1.** Not a close blocker.
- **F2 — Low (accept):** rod-cli reject layer doesn't reject backslash, but godoll `json.Marshal` escapes it — intentional defense-in-depth, escape layer is load-bearing.
- **F3 — Info (accept):** `--request-capture`/`--console-capture` retain full URLs/console args in memory (opt-in, default OFF, in-memory only, returned to the operator's own session). Operator awareness.
- **F4 — Info (fix-later, godoll):** godoll `EnableRequestInterception` interpolates platform without `json.Marshal` — **not on rod-cli's path** (rod-cli carries identity via Emulation + its own interceptor; zero call sites). Upstream hygiene.
- **F5 — confirmations (no action):** proxy-auth invariant intact; embedded profiles no-fetch; CDP ledger exposes domain names only (no values); per-session seed uses crypto/rand with sound fallback.

## Disposition
Clear v1.7 to close. Route **F1** (toolchain bump) to the operator as fix-later. F2/F4 are upstream-godoll hygiene; F3 operator-awareness only.
