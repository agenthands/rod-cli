---
author: architect
responsible: architect
phase: 54
status: ready
parent_artifacts:
  - .planning/phases/54/CONTEXT.md
  - .planning/REQUIREMENTS.md (DOCS-01, DOCS-02)
---

# Phase 54: Documentation & Discoverability — PLAN

## {P} Precondition
Three publish-ready packages (0.1.0, unpublished/404). Pi README substantial but 2-way + unlabelled npm install; core/omp READMEs are stubs; top-level README + onboarding/pi.md + SKILL.md mention only Pi and present the npm install unlabelled. No `--no-banner` in docs; banner flag is `--banner`.

## {Q} Postcondition
Pi + OMP READMEs have correct install (npm-install labelled "once published" + working local path), the 13-tool catalog, prerequisites, and a verify step; the core README frames it as the shared library; the top-level README + onboarding/pi.md + SKILL.md link both extensions and present the three-path (OMP/Pi/Skill) comparison; every command/example matches the shipped package names + current CLI surface (no `--no-banner`, no unpublished-install-as-verified).

## must_haves.truths
1. **No unpublished-install claim presented as verified.** Every `pi install npm:@agenthands/rod-cli-pi` / `omp plugin install npm:@agenthands/rod-cli-omp` is labelled "(once published)" and paired with the working local install (`pi install ./extensions/pi` / `omp plugin link ./extensions/omp`).
2. **No stale flags.** `grep -rn "\-\-no-banner" README.md docs skills extensions/*/README.md` → empty. Any banner mention uses `--banner` (opt-in, off by default).
3. **Accurate names + catalog.** Package names `@agenthands/rod-cli-{core,pi,omp}`; tool names match the 13 registered `browse_*`; the OMP catalog matches the Pi catalog (shared core).
4. **Three-path present.** Top-level README + onboarding/pi.md + SKILL.md each link both extensions and distinguish OMP / Pi / Skill.
5. **Links resolve.** Relative links between the docs (READMEs ↔ docs/extension-release.md ↔ SKILL.md ↔ top-level README) point at real files.

## §1 — Pi README (D-03) `extensions/pi/README.md`
- Replace the 2-way "Skill vs Extension" table with the CONTEXT D-01 **three-way** OMP/Pi/Skill table.
- Label the Install section (D-02): show `pi install ./extensions/pi` as the working path today and `pi install npm:@agenthands/rod-cli-pi` as "(available once published)".
- Keep the tool catalog + native-vs-eval guidance + the release-doc reference.

## §2 — OMP README (D-04) `extensions/omp/README.md`
Full README (mirror the Pi shape):
- **Title + purpose** (native OMP browser tools on the shared core).
- **Prerequisites:** rod-cli binary (`go install github.com/agenthands/rod-cli@latest`), Chromium (`rod-cli install`), an OMP host.
- **Install:** `omp plugin link ./extensions/omp` (works today) + `omp plugin install npm:@agenthands/rod-cli-omp` "(available once published)".
- **Tool catalog:** the same 13-tool table as Pi (verify names against `extensions/omp/src/tools/*.ts`).
- **Verify:** start an OMP session → "Browse to https://example.com and tell me the page title" → expect `browse_goto`+`browse_snapshot`.
- **Three-path table** (compact D-01 or link to the top-level README's).
- **OMP specifics:** injected `pi.zod` schemas, guidance folded into `description`, ungated idempotent daemon cleanup on shutdown.
- **Release process:** link `docs/extension-release.md`.

## §3 — core README (D-05) `extensions/core/README.md`
Expand the stub: state it is the **shared library** both adapters depend on (real `dependencies`), **not installed directly by end users**; brief public surface (`findRodCli`, the normalized `ExecFn`/`ExecResult` contract + `execRodCli`, the 13 pure argv-builders, `waitForSelector`/`sleep`); "consumed by `@agenthands/rod-cli-pi` and `@agenthands/rod-cli-omp`"; link `docs/extension-release.md`.

## §4 — Top-level README (D-06) `README.md`
- Add an **OMP Extension** entry to the Documentation list (link `extensions/omp/README.md`); update the Pi Extension line if needed.
- Add a short **"Integration paths"** section: the D-01 three-way table (this is the canonical copy) + links to both READMEs + `skills/rod-cli/SKILL.md`.

## §5 — onboarding/pi.md (D-07) `docs/onboarding/pi.md`
- Label the `pi install npm:` claim (D-02) + show the local path.
- Add a cross-link: "OMP users: use the native `@agenthands/rod-cli-omp` extension (see its README)."
- Keep the Skill path as the zero-install fallback.

## §6 — SKILL.md (D-08) `skills/rod-cli/SKILL.md`
- In the Pi/extension mention, add the **OMP** extension alongside Pi, and the three-path distinction (Skill = this file, for any Agent-Skills assistant; Pi/OMP = native extensions).
- Label any npm-install claim (D-02).

## §7 — Verification (the gate)
1. **No unpublished-install-as-verified (truth #1):** grep every `pi install npm:` / `omp plugin install npm:` occurrence → each has an adjacent "once published" label + a working local alternative. List them.
2. **No stale flags (truth #2):** `grep -rn "\-\-no-banner" README.md docs skills extensions/*/README.md` → empty.
3. **Names/catalog accurate (truth #3):** the OMP README's 13 tool names == the names in `extensions/omp/src/tools/*.ts` (diff the sets); package names match `extensions/*/package.json`.
4. **Three-path present (truth #4):** confirm the table/links in README.md + onboarding/pi.md + SKILL.md.
5. **Links resolve (truth #5):** every relative link added points at an existing file (check `docs/extension-release.md`, both READMEs, SKILL.md, extensions paths).
6. **No accidental code touch:** `git status` shows only doc files changed; re-run the three gates once (core 139 / pi 38 / omp 30) to confirm nothing broke (docs shouldn't affect them, but confirm since READMEs are in `files`).

## Execution order
§1 Pi README → §2 OMP README → §3 core README → §4 top-level README → §5 onboarding → §6 SKILL.md → §7 verification.

## Files changed
| File | § | Status |
|---|---|---|
| `extensions/pi/README.md` | §1 | MODIFIED (3-way table, labelled install) |
| `extensions/omp/README.md` | §2 | MODIFIED (stub → full) |
| `extensions/core/README.md` | §3 | MODIFIED (stub → library doc) |
| `README.md` | §4 | MODIFIED (OMP link + integration-paths section) |
| `docs/onboarding/pi.md` | §5 | MODIFIED (labelled install + OMP cross-link) |
| `skills/rod-cli/SKILL.md` | §6 | MODIFIED (both extensions + three paths) |
