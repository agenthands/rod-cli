---
slug: architect-v1-6-security-review
created_at: 2026-06-25T20:58:11Z
modified_at: 2026-06-25T20:58:11Z
---
---
name: architect-v1-6-security-review
description: v1.6 security review findings and disposition
metadata:
  type: reference
  milestone: v1.6
---

# v1.6 Security Review — rod-cli

## Attack Surface

v1.6 introduces four key security-relevant surfaces:

1. **Per-session proxy with CDP auth** (`--proxy`, `--proxy-auth`)
2. **JSON profile-file loading** (`--profile`)
3. **CLI flag → Config forwarding** (ResolveStealth)
4. **JS injection for fingerprint/canvas noise** (godoll integration)

## Findings

### F1: Profile Path Resolution — No Path Traversal Guard (Low)

**Location:** `types/config.go:243-259` (`resolveProfilePath`)

**Issue:** A user-provided `--profile` value containing path separators (`/` or `os.PathSeparator`) or `.json` suffix is used verbatim. An attacker with CLI access could potentially load arbitrary JSON files:
- `--profile ../../../etc/passwd` (if it exists and parses as JSON)
- `--profile /etc/some-secret.json`

**Mitigating factors:**
- CLI-only access required (not exposed over network)
- Profile files must be valid JSON to be parsed by `stealth.LoadProfile`
- The loaded content only affects `Config.Stealth` fields — no arbitrary code execution
- Missing file is a LOUD failure (daemon aborts, not silent fallback)

**Verdict:** **ACCEPTED RISK** — The attack requires local CLI access and the impact is limited to loading stealth config from arbitrary paths. No credential exposure. Could be hardened by restricting to `$HOME/.rod-cli/profiles/` or `./profiles/` only, but current design trades flexibility for security in a local CLI context.

**Recommendation:** Document in security guidance that `--profile` accepts arbitrary paths and should not be used with untrusted input.

---

### F2: Proxy Credential Stripping — Correctly Handled (PASS)

**Location:** `types/context.go:68-74`, `types/config.go:46-50`

**Issue:** Proxy credentials could leak through URL embedding or config serialization.

**Controls found:**
1. **URL-embedded credentials rejected** (line 72-74): `if u.User != nil { return nil, errors.Errorf("embedded proxy credentials...") }`
2. **ProxyAuth excluded from serialization**: `ProxyAuth string \`yaml:"-" json:"-"\`` — credentials never written to disk
3. **Credentials passed via CDP** (`Fetch.continueWithAuth`), never to `--proxy-server` Chrome flag
4. **Test coverage** (`types/config_test.go:186-208`): Tests explicitly verify credential non-persistence

**Verdict:** **PASS** — Correctly implemented. Credential handling follows security best practices.

---

### F3: SOCKS5 Auth Relay — Known Gap (Tech Debt)

**Location:** Tech debt from Phase 25 milestone audit

**Issue:** Authenticated SOCKS5 (`socks5://` + `--proxy-auth`) is accepted but godoll's auth relay speaks HTTP CONNECT to the SOCKS upstream — may mishandle authentication.

**Status:** Documented as tech debt. Root cause is in godoll dependency.

**Verdict:** **TECH DEBT** — Accepted with documentation. Not a v1.6 blocking issue since SOCKS5+auth is not a primary use case.

---

### F4: Stealth Config Injection — Boundary Validated (PASS)

**Location:** `types/config.go:269-580` (`ResolveStealth`)

**Issue:** CLI flags and profile values could inject malicious content into fingerprint fields.

**Controls found:**
1. **Type safety**: Values are typed (string, bool, int) — no arbitrary code execution
2. **Consistency validation**: `ValidateStealth()` checks UA ↔ Client-Hints ↔ platform coherence
3. **Loud failure mode**: Invalid profiles abort daemon startup, not silent fallback
4. **No shell command injection**: Values passed to godoll via Go struct, not shell args

**Verdict:** **PASS** — Injection boundary is correctly typed and validated.

---

### F5: JS Injection Boundaries — godoll Trust Boundary (PASS)

**Issue:** Canvas/WebGL/Audio noise and fingerprint spoofing inject JS into browser pages.

**Controls:**
- Injection occurs in controlled godoll library context
- Values derive from resolved Config, not raw user input (after precedence resolution)
- Noise seeds are per-session stable, preventing timing-based detection

**Verdict:** **PASS** — JS injection is within the designed trust boundary (user controls the browser session).

---

## Threat Register

| Threat | Boundary | Asset | Disposition |
|--------|----------|-------|-------------|
| Path traversal via `--profile` | CLI → filesystem | Config file content | ACCEPTED — limited impact, CLI-only |
| Credential leak via URL embed | CLI → proxy config | Proxy credentials | MITIGATED — rejected at parse |
| Credential persistence | Config serialization | Proxy auth | MITIGATED — yaml:"-" tag |
| SOCKS5 auth mishandling | godoll relay | Proxy credentials | TECH DEBT — documented |
| Config injection | CLI → godoll | Fingerprint values | PASS — typed, validated |
| JS injection | godoll → browser | Page content | PASS — trust boundary |

## Verdict

**PASSED WITH TECH DEBT.**

All primary security controls are correctly implemented:
- Credential stripping (F2) — PASS
- Config injection boundary (F4) — PASS
- JS trust boundary (F5) — PASS

One accepted risk:
- Profile path traversal (F1) — LOW, CLI-only, documented

One documented tech debt:
- SOCKS5 auth relay gap (F3) — deferred to godoll fix

**Recommendation:** No blocking issues for v1.6 close. Document F1 in security guidance for users.