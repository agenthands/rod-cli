# Requirements — v1.9 godoll Hygiene & CDP-DEEP-01 Research

**Milestone goal:** Close the two remaining v1.7 security-review hygiene items (F2/F4) in godoll and rod-cli, and produce an evaluated, grounded design for CDP-DEEP-01 (deep CDP signal obfuscation) that a follow-on milestone can execute.

**Grounding:** rod-cli is a **pure CLI/daemon** (no MCP server — verified at HEAD). The v1.8 onboarding docs document shell-out only. v1.9 continues the v1.7→v1.8 carry-forward chain: F2/F4 are the last unaddressed v1.7 security-review findings; CDP-DEEP-01 is the honest-ceiling work deferred from v1.7 Phase 30.

---

## v1.9 Requirements

### Godoll Hygiene (HYGIENE) — v1.7 security-review F2/F4

- [ ] **HYGIENE-01 (F2):** rod-cli's `rejectUnsafeFingerprintValue` rejects backslash (`\`) in addition to the already-rejected `"` and control chars — defense-in-depth (godoll's `json.Marshal` already escapes it, so this is a belt-and-suspenders hardening).
- [ ] **HYGIENE-02 (F4):** godoll's `EnableRequestInterception` uses `json.Marshal` when interpolating `platform` into the JS literal, matching the pattern used for other profile-controlled strings — upstream correctness fix (note: rod-cli never calls this function; it carries identity via Emulation + its own interceptor).

### CDP-DEEP-01 Research & Design (CDP-DEEP)

- [ ] **CDP-DEEP-01:** Evaluate the three approaches named in the v1.7 honest ceiling — **browser-patching**, **MITM/alternate transport**, and **patched DevTools endpoint** — against Chrome's current detection surface (CDP WebSocket observable via `chrome.debugger`, protocol message sniffing, `Page`/`Target` domain enable signals). Produce a grounded recommendation with feasibility, risk, and effort estimates.
- [ ] **CDP-DEEP-02:** Produce a concrete, executable PLAN for the chosen approach — centerpiece symbols grounded against the real go-rod/rod-cli transport layer via SMTC, with a phased execution path — so a follow-on milestone can pick it up and build.
- [ ] **CDP-DEEP-03:** Update `docs/cdp-footprint.md` with the CDP-DEEP-01 research findings, the chosen approach, and an updated honest ceiling reflecting what the chosen approach can and cannot obfuscate.

---

## Future Requirements (deferred)

- Build the CDP-DEEP-01 approach (browser-patching, MITM, or patched endpoint) — gated on this milestone's research/design output.
- Heavier first-class Extensions that wrap rod-cli as an LLM-callable tool (Pi TypeScript extension, Gemini `gemini-extension.json`) — optional polish beyond shell-out onboarding.
- A genuine MCP server mode for rod-cli — only if a future milestone re-adds MCP.

## Out of Scope

- **TLS/JA3-JA4 spoofing** — rod-cli drives real Chrome; its TLS is authentic by construction. Permanent exclusion.
- Building the CDP-DEEP-01 approach — this milestone produces the evaluated design, not the implementation.

---

## Traceability

| REQ-ID | Phase | Status |
|--------|-------|--------|
| HYGIENE-01 | TBD | pending |
| HYGIENE-02 | TBD | pending |
| CDP-DEEP-01 | TBD | pending |
| CDP-DEEP-02 | TBD | pending |
| CDP-DEEP-03 | TBD | pending |

## Archived

<details>
<summary>v1.8 Debt Cleanup & Coding-Assistant Onboarding (shipped 2026-06-26)</summary>

All 16 requirements (BUILD-01/02, LEDGER-01/02, FONT-01/02/03, DOC-01..09) met and verified. See `.planning/v1.8-MILESTONE-AUDIT.md`.
</details>
