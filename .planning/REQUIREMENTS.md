# Requirements — v2.1 CDP Proxy Hardening & Diagnostics

**Milestone goal:** Close the v2.0 carry-forward items (live-browser proxy test, jitter validation, cdp-traffic sensitivity warning) and the long-standing v1.7 font-spoof no-op.

**Grounding:** rod-cli's CDP proxy (`internal/cdpproxy/`) is built and shipping but lacks an integration test. The `--font-spoof` toggle (v1.7 Phase 33) activates godoll's font injector but the injector is a documented no-op — make it real.

---

## v2.1 Requirements

### Proxy Hardening (PROXY)

- [ ] **PROXY-01:** A live-browser integration test starts a browser with `--cdp-proxy`, navigates a page, and asserts `Traffic()` contains expected CDP messages (Page.frameNavigated, etc.).
- [ ] **PROXY-02:** With `--cdp-proxy` and `--console-capture`, the `cdpTell` probe returns `"no-signal"` (normalization working).
- [ ] **PROXY-03:** `--cdp-jitter-ms` values above 1000ms produce a soft warning (stderr, not blocking).
- [ ] **PROXY-04:** The `cdp-traffic` help text includes a caveat about sensitive CDP payload data in output.

### Font Spoofing (FONT)

- [ ] **FONT-04:** With `--font-spoof` enabled, a font-probe on the live page reads a spoofed font set that differs from the host baseline (the no-op is gone).
- [ ] **FONT-05:** The spoofed set is coherent with the active profile's OS/locale and identical across re-reads on the same session.
- [ ] **FONT-06:** `--font-spoof=false` restores genuine host font behavior (harness-asserted).
- [ ] **FONT-07:** The offline detection harness asserts criteria FONT-04 through FONT-06.

## Traceability

| REQ-ID | Phase | Status |
|--------|-------|--------|
| PROXY-01 | TBD | pending |
| PROXY-02 | TBD | pending |
| PROXY-03 | TBD | pending |
| PROXY-04 | TBD | pending |
| FONT-04 | TBD | pending |
| FONT-05 | TBD | pending |
| FONT-06 | TBD | pending |
| FONT-07 | TBD | pending |

## Archived

<details>
<summary>v2.0 CDP-DEEP-01 Build (shipped 2026-06-26)</summary>

3 phases (40-42), 3 plans. MITM WebSocket CDP proxy built: pass-through + ring buffer, Runtime.getProperties normalization (7 tests), timing jitter, cdp-traffic command, bypass flag. See `.planning/v2.0-MILESTONE-AUDIT.md` and `.planning/v2.0-SECURITY.md`.
</details>

<details>
<summary>v1.9 godoll Hygiene & CDP-DEEP-01 Research (shipped 2026-06-26)</summary>

2 phases (38-39). Closed v1.7 F2 (backslash reject) and F4 (json.Marshal platform). Produced evaluated CDP-DEEP-01 design (MITM approach chosen). See `.planning/v1.9-MILESTONE-AUDIT.md`.
</details>

<details>
<summary>v1.8 Debt Cleanup & Coding-Assistant Onboarding (shipped 2026-06-26)</summary>

4 phases (34-37). Toolchain go1.26.4 + govulncheck CI, plugin-path CDP-ledger, real font spoofing stub, 5-assistant onboarding docs. See `.planning/v1.8-MILESTONE-AUDIT.md`.
</details>

<details>
<summary>v1.7 Complete Evasion Stack (shipped 2026-06-26)</summary>

3 phases (30, 32, 33; Phase 31/TLS cancelled). CDP footprint reduction, 6 Chrome-only profiles, 4 activated godoll dimensions. See `.planning/milestones/v1.7-MILESTONE-AUDIT.md`.
</details>
