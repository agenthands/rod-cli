---
author: architect
responsible: architect
phase: 54
phase_type: documentation
hard_bar: false
security_relevant: false
design_fork: false
status: locked
parent_artifacts:
  - .planning/REQUIREMENTS.md (DOCS-01, DOCS-02)
  - .planning/ROADMAP.md (Phase 54 Success Criteria)
  - .planning/phases/53/SUMMARY.md (final package names + install reality)
---

# Phase 54: Documentation & Discoverability — CONTEXT

**Gathered:** 2026-07-05
**Status:** Ready for planning

<domain>
## Phase Boundary

Correct and complete the consumer-facing docs so users discover, install, and use the
three integration paths accurately. Per-package READMEs (Pi + OMP + core) with correct
host-specific install + verify steps and tool catalog; the top-level `README.md`,
`docs/onboarding/pi.md`, and `skills/rod-cli/SKILL.md` cross-link both published
extensions and clearly distinguish the **three** paths (OMP extension / Pi extension /
Agent Skill). Every command/example verified against the shipped package names + current
CLI surface — no stale flags, no unpublished-install claim presented as verified.

**In scope:** DOCS-01, DOCS-02 — README content for all three packages + the cross-linking
doc pass. **Out of scope:** any code/config/workflow change (all shipped in 51–53); the
live npm publish; new tools.
</domain>

## {P} What the phase can assume (verified 2026-07-05)

- Three publish-ready packages exist: `@agenthands/rod-cli-core`, `-pi`, `-omp` (all `0.1.0`, MIT LICENSE, publish-ready package.json). **None are on npm yet** (`@agenthands/rod-cli-pi` → 404); the `ext-v*` pipeline publishes them once a maintainer token exists.
- `extensions/pi/README.md` is substantial (Phase 50 + a release-doc ref) but has a **2-way** Skill-vs-Extension table and presents `pi install npm:@agenthands/rod-cli-pi` as if it works (it 404s).
- `extensions/core/README.md` and `extensions/omp/README.md` are **stubs** (Phase 53).
- `docs/onboarding/pi.md` and `skills/rod-cli/SKILL.md` mention only the **Pi** extension (not OMP), and present the `pi install npm:` claim unlabelled.
- Top-level `README.md` "Documentation" list links the Pi Extension but not OMP, and has no 3-path comparison.
- **CLI-surface facts (verified):** the startup banner is OFF by default; the flag is `--banner` (opt-in) — there is **no `--no-banner`**. `--raw` is a documented no-op for the browse commands. 13 `browse_*` tools. rod-cli install: `go install github.com/agenthands/rod-cli@latest` + `rod-cli install` (Chromium). OMP local install: `omp plugin link ./extensions/omp`; Pi local: `pi install ./extensions/pi`.

## {Q} What the phase must establish

1. **DOCS-01** — Pi + OMP READMEs each carry correct host-specific install, the 13-tool catalog, prerequisites (rod-cli binary + Chromium), and a verify-on-install step. `pi install npm:` / `omp plugin install npm:` claims are **labelled as working only once a live publish exists** (currently: local install/link) — no unpublished-install claim presented as verified.
2. **DOCS-02** — the top-level `README.md`, `docs/onboarding/pi.md`, and `skills/rod-cli/SKILL.md` link both extensions and present a clear **three-path** (OMP / Pi / Skill) comparison.
3. **SC3** — every documented command/example is verified against the shipped package names + current CLI surface: **no stale flags** (no `--no-banner`), **no unpublished-install claim presented as verified**, correct package names, correct tool names.

<decisions>
## Implementation Decisions

### The canonical three-path framing (used everywhere)
- **D-01:** One comparison table, reused/linked across surfaces:

  | | OMP extension (`@agenthands/rod-cli-omp`) | Pi extension (`@agenthands/rod-cli-pi`) | Agent Skill (`skills/rod-cli/SKILL.md`) |
  |---|---|---|---|
  | Host | OMP (oh-my-pi) | Pi (earendil) | any Agent-Skills assistant (Claude Code, Codex, Gemini, opencode, …) |
  | Setup | `omp plugin install npm:@agenthands/rod-cli-omp` (once published) / `omp plugin link ./extensions/omp` (today) | `pi install npm:@agenthands/rod-cli-pi` (once published) / `pi install ./extensions/pi` (today) | copy `SKILL.md` to the assistant's skills dir |
  | Tools | 13 typed `browse_*` (injected `pi.zod`) | 13 typed `browse_*` (TypeBox) | LLM shells out to `rod-cli` subcommands |
  | Validation / lifecycle | schema + session_start/shutdown hooks | schema + session_start/shutdown hooks | none |
  | Best for | OMP users (native path) | Pi users | multi-assistant / zero-npm |

### Install labelling (DOCS-01 / SC3 — load-bearing)
- **D-02:** Every `npm`-registry install command is labelled: e.g. "**(available once published)** `pi install npm:@agenthands/rod-cli-pi`" with the **currently-working** local path shown alongside (`pi install ./extensions/pi` / `omp plugin link ./extensions/omp`). Never present the npm-registry install as verified/working today.

### Per-package READMEs (DOCS-01)
- **D-03:** **Pi README** — upgrade the 2-way table to the D-01 three-way; label the npm install (D-02); keep the (accurate) 13-tool catalog + native-vs-eval guidance + release-doc ref.
- **D-04:** **OMP README** — full content mirroring the Pi README's shape: purpose; prerequisites (rod-cli binary + `rod-cli install` Chromium + OMP host); install (`omp plugin link ./extensions/omp` today, `omp plugin install npm:@agenthands/rod-cli-omp` once published); the 13-tool catalog; verify step ("Browse to example.com and tell me the title"); the D-01 three-way table; release-doc ref. Note OMP-specifics: injected `pi.zod` schemas, guidance folded into `description`, ungated idempotent shutdown.
- **D-05:** **core README** — clarify it is the **shared library** consumed by the adapters (a real `dependencies` entry), **not installed directly by end users**; brief public-API surface (findRodCli, execRodCli/ExecFn contract, argv-builders) + a "used by rod-cli-pi and rod-cli-omp" pointer + release-doc ref.

### Cross-linking pass (DOCS-02)
- **D-06:** **Top-level `README.md`** — add the OMP Extension to the Documentation list; add a short "Integration paths" section with the D-01 three-way table (or a tight summary + links to both READMEs + SKILL.md).
- **D-07:** **`docs/onboarding/pi.md`** — label the `pi install npm:` claim (D-02); add a cross-link noting OMP users should use the native OMP extension (link its README); keep the Skill path as the zero-install fallback.
- **D-08:** **`skills/rod-cli/SKILL.md`** — mention BOTH extensions (currently only Pi) and the three-path distinction; label npm-install claims (D-02).

</decisions>

<disciplines>
## Lenses surfaced for this phase

~~~yaml
disciplines:
  - section: "inline"
    name: "docs-verified-against-reality"
    scope: ["README.md", "extensions/*/README.md", "docs/onboarding/pi.md", "skills/rod-cli/SKILL.md"]
    guidance: "Every command, flag, package name, and tool name in the docs must match what actually ships RIGHT NOW: run/grep it before writing it. An install that 404s is labelled 'once published', not shown as working; a removed flag (--no-banner) never appears; a tool name matches the registered name."
    rationale: "Docs are the consumer's contract; a doc that claims a working npm install that 404s, or a flag that no longer exists, actively breaks the user and erodes trust — the exact failure this milestone exists to end."
    override_signals:
      - "A forward-looking 'coming soon / once published' statement is fine WHEN explicitly labelled as not-yet-available (that is honest future intent, not a false present-tense claim)."
      - "Illustrative pseudo-output or a shortened example may elide detail as long as it is not presented as a verbatim, runnable command."
~~~

</disciplines>

<canonical_refs>
## Canonical References
- `.planning/REQUIREMENTS.md` (DOCS-01, DOCS-02) + `.planning/ROADMAP.md` §"Phase 54" (3 Success Criteria).
- `extensions/pi/README.md` (the shape to mirror + upgrade), `extensions/{core,omp}/README.md` (stubs to expand), `docs/extension-release.md` (to reference).
- `README.md` (top-level Documentation list), `docs/onboarding/pi.md`, `skills/rod-cli/SKILL.md` (the cross-link surfaces).
- The three `extensions/*/package.json` (authoritative package names + versions) and `extensions/*/src/tools/*.ts` (authoritative tool names + descriptions) — verify the catalog against these.
- CLI surface: `~/bin/rod-cli --help` (verify flags: `--banner` exists, `--no-banner` does NOT); `docs/stealth-config.md` §4 for stealth flags if referenced.

</canonical_refs>

<code_context>
## Existing Code Insights
- The Pi README's tool catalog is accurate (13 tools, correct params) — reuse it for the OMP README (same 13 tools, same argv via the shared core).
- The unpublished-install claim appears in 3 places: `extensions/pi/README.md`, `docs/onboarding/pi.md`, `skills/rod-cli/SKILL.md` — all must be labelled (D-02).
- `skills/rod-cli/SKILL.md` currently mentions only the Pi extension (line ~141) — add the OMP extension + the three-path distinction.
- No `--no-banner` currently in the docs (grep clean) — keep it that way; SC3 is a "confirm, don't introduce" check.

</code_context>

<specifics>
## Specific Ideas
- Keep ONE source of truth for the three-path table (author it once, e.g. in the top-level README, and have the per-package READMEs carry a compact version + link) to avoid drift.
- The honest present-tense: today the working install is local (`pi install ./extensions/pi`, `omp plugin link ./extensions/omp`); the npm-registry install is "once published." State both.

</specifics>

<deferred>
## Deferred Ideas
- None — this is the milestone's final phase; remaining work (live publish) is the maintainer-gated step tracked in Phase 53.

</deferred>

---

*Phase: 54-documentation-discoverability*
*Context gathered: 2026-07-05*
