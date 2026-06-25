# Requirements: rod-cli — v1.7 Complete Evasion Stack

**Defined:** 2026-06-26
**Core Value:** Native, token-efficient browser automation via standard I/O explicitly designed for LLM integration.

> **Milestone framing:** v1.6 proved and configured JS-layer stealth. v1.7 extends to the full stack — reducing CDP signals at the protocol layer, spoofing TLS fingerprints at the network layer, providing curated device profiles out of the box, and expanding hardening surfaces. This is the most architecturally significant milestone since the original godoll migration.

## v1 Requirements

### CDP Footprint Reduction

- [ ] **CDP-01**: The `Runtime.enable` signal is reduced or obfuscated — measured against detection harness and documented with an honest ceiling for what remains detectable.
- [ ] **CDP-02**: All CDP commands sent by rod-cli are inventoried; each has a documented mitigation or accepted visibility.
- [ ] **CDP-03**: CDP footprint reduction is measured against live detection targets; the harness asserts the reduction baseline.

### Network-Layer Identity (TLS)

- [ ] **TLS-01**: TLS/JA3-JA4 fingerprint matches the declared User-Agent/platform profile — the network-layer identity is coherent with the JS-layer identity.
- [ ] **TLS-02**: A TLS fingerprinting library (uTLS or equivalent) is integrated; HTTP/HTTPS requests from the browser carry spoofed TLS signatures.
- [ ] **TLS-03**: Proxy-aware TLS spoofing works correctly — TLS fingerprints do not leak through HTTP or SOCKS5 proxies.
- [ ] **TLS-04**: The TLS spoofing layer is configurable per session (enabled by default, can be disabled for debugging).

### Profile Library

- [ ] **PROF-01**: A built-in library of 5-10 vetted, coherent device profiles is shipped with the binary.
- [ ] **PROF-02**: Each profile is tested against the detection harness; profiles that fail key signals are not shipped.
- [ ] **PROF-03**: Users can list built-in profiles (`--profile=list`) and select by name without providing a path.
- [ ] **PROF-04**: Custom profiles (user-provided JSON files) continue to work alongside built-in profiles; CLI flags override both.

### Advanced Evasion

- [ ] **EVAD-01**: Additional fingerprint hardening surfaces are identified and prioritized based on detection harness results.
- [ ] **EVAD-02**: At least two new hardening surfaces (beyond v1.6 canvas/WebGL/WebRTC) are implemented and asserted by the harness.
- [ ] **EVAD-03**: Any new hardening toggles follow the v1.6 precedence chain (CLI > profile > default) and are documented in the stealth-config guide.

## v2 Requirements (Deferred)

### CDP Deepening

- **CDP-DEEP-01**: Full CDP protocol obfuscation — may require browser patching or MITM proxy.

### Extended Profile Ecosystem

- **PROF-ECO-01**: Remote profile update mechanism — fetch profiles from a curated repository.
- **PROF-ECO-02**: Profile versioning and deprecation — profiles older than N months warn on use.

### Mobile Fingerprints

- **MOB-01**: Mobile device profiles (smartphone/tablet) — currently out of scope for v1.7; requires separate investigation.

## Out of Scope

| Feature | Reason |
|---------|--------|
| "Bypasses Cloudflare/DataDome" guarantee or blocking CI gate against live WAFs | They score TLS (JA3/JA4) + IP reputation + behavior — layers this milestone addresses, but non-deterministic → flaky CI. rod-cli hardens the browser/JS + TLS layer; IP reputation is the operator's job. |
| CAPTCHA solving | Scope creep, ToS/legal exposure, external paid deps; breaks the zero-dependency single-binary constraint. |
| Mobile fingerprints from desktop headless | Desktop reporting mobile signals is exactly the inconsistency detectors flag. Ship vetted desktop profiles only. |
| Remote profile repository | Deferred to v2 (PROF-ECO-01) — built-in profiles are the v1.7 scope. |

## Traceability

Phase mapping will be assigned by the roadmapper (v1.7 = Phases 30–33).

| Requirement | Phase | Status |
|-------------|-------|--------|
| CDP-01 | Phase 30 | Pending |
| CDP-02 | Phase 30 | Pending |
| CDP-03 | Phase 30 | Pending |
| TLS-01 | Phase 31 | Pending |
| TLS-02 | Phase 31 | Pending |
| TLS-03 | Phase 31 | Pending |
| TLS-04 | Phase 31 | Pending |
| PROF-01 | Phase 32 | Pending |
| PROF-02 | Phase 32 | Pending |
| PROF-03 | Phase 32 | Pending |
| PROF-04 | Phase 32 | Pending |
| EVAD-01 | Phase 33 | Pending |
| EVAD-02 | Phase 33 | Pending |
| EVAD-03 | Phase 33 | Pending |

**Coverage:**

- v1 requirements: 13 total
- Mapped to phases: 13 ✓ (100% — every requirement maps to exactly one phase)
- Unmapped: 0

---
*Requirements defined: 2026-06-26*
*Last updated: 2026-06-26 at milestone start*