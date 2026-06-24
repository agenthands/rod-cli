---
kind: workflow
name: autonomous
version: v0.1
transitions: []
---
<purpose>
Drive milestone phases autonomously in the **anvil 5-peer model** — all remaining
phases, a range (`--from N`/`--to N`), or one (`--only N`). The **architect (lead)**
runs the loop, **delegating each stage to its owning peer via handshake** (not GSD's
flat single-orchestrator Skill chain). Re-reads ROADMAP after each phase. Pauses only
for genuine design forks, blockers/refusals, and the milestone-close gate.
</purpose>

<anvil_role_map>
This is the lead's orchestration loop; it **delegates**, it does not execute spans
itself. Per `docs/05-team-workflow.md`: the lead owns discuss + plan; the **engineer**
owns execute, **qa** owns verify, **security-engineer** owns secure, the
**codebase-archaeologist** owns map/document. Each span's specialist gate is the owning
peer's responsibility (the lead does not spawn `anvil-executor`/`anvil-verifier`
itself). Where this references a stage workflow, that workflow's own
`<anvil_delegation_directive>` governs how it delegates.
</anvil_role_map>

<process>

<step name="initialize">
Parse `$ARGUMENTS` for `--from N` / `--to N` / `--only N` / `--interactive`.
```bash
state=$(anvil-cc state json --raw); roadmap=$(anvil-cc roadmap analyze --raw)
```
If no ROADMAP → error: "Run new-milestone first." Announce the run mode + range.
</step>

<step name="discover_phases">
From `roadmap analyze`, keep **incomplete** phases (disk_status ≠ complete) in range.
This list is re-derived after every phase (to catch phases inserted mid-run).
</step>

<step name="phase_loop">
For each incomplete phase, in order:

1. **Discuss → CONTEXT** (architect, in-band). Use the `anvil-decomposition` lens for
   scoping. On a **genuine design fork (>1 defensible answer)**, spawn
   **`anvil-design-options`** and **PAUSE for the co-driver** to choose. Verify
   CONTEXT exists; if not → `handle_blocker`.

2. **Plan → PLAN** (architect, in-band on opus; spawn **`anvil-planner`** only if
   large/parallel; **`anvil-researcher`** for an external-knowledge gap). Verify PLAN
   exists; if not → `handle_blocker`.

3. **PLAN handshake → engineer.** Hand the PLAN to the engineer; it **accepts or
   refuses** (the cross-role plan review — this replaces GSD's plan-checker). On
   `PLAN REFUSED` → revise the plan and re-hand (bounded; 3 refusals on the same point
   → `handle_blocker`).

4. **Execute (engineer span).** The engineer runs its execute discipline
   (`agents/engineer.md`) — incl. the **`anvil-code-reviewer` gate before handoff**,
   `anvil-adversarial-tester` for the hard bar, the regression / post-merge / drift
   gates — and returns `EXECUTE COMPLETE` + SUMMARY. On `EXECUTE BLOCKED` (plan flaw)
   → back to step 2. (The lead delegates + waits; it does not execute.)

5. **VERIFY handshake → qa.** qa runs the verifier gate → VERIFICATION (passed /
   gaps_found / human_needed). **Gaps route UP to you (the architect)** — never
   lateral to the engineer.
   - `gaps_found` → plan gap-closure (step 2, `--gaps`) → re-hand to engineer (step 3).
   - `human_needed` → **PAUSE for the co-driver**.

6. **Re-read ROADMAP** (`roadmap analyze`) and continue to the next incomplete phase.
</step>

<step name="milestone_close">
When no incomplete phases remain:
1. **PAUSE — ask the co-driver:** "security review for v{X.Y}?" On yes, **SECURITY
   handshake → security-engineer** (threat-modeler → security-auditor; cert engine on
   the keystone). Findings route up.
2. **Document** — **codebase-archaeologist** refreshes `docs/` + map (doc-verifier
   gate); run `self-audit`.
3. Architect runs `complete-milestone` (MILESTONE-AUDIT + measurability:
   `debt query`, `measure refusal-rate|kb`). Present the close summary; **PAUSE**.
</step>

<step name="handle_blocker">
Stop the autonomous run, preserve state, and present the blocker + options to the
co-driver (fix-and-resume / skip-phase / stop). Autonomous mode **never** forces past
a refusal, a design fork, or a human-needed verdict — those are the explicit pause
points by contract.
</step>

</process>
