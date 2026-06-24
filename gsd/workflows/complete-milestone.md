---
kind: workflow
name: complete-milestone
version: v0.1
transitions: []
---
## Context setup — run these first

Parse `$ARGUMENTS` for the optional version string, then gather context and end any active auto-chain:

```bash
roadmap=$(anvil-cc roadmap analyze --raw)
state=$(anvil-cc state json --raw)
anvil-cc config-set workflow._auto_chain_active false 2>/dev/null || true
```

Use `$roadmap` and `$state` below.

## Milestone Context (from the setup block above)

**Version to complete:** `$version`

**Roadmap:**
`$roadmap`

**State:**
`$state`

<process>

<step name="verify_readiness">

**Use `roadmap analyze` for comprehensive readiness check:**

Extract `branching_strategy`, `phase_branch_template`, `milestone_branch_template`, and `commit_docs` from init JSON.

**If "none":** Skip to git_tag.

**For "phase" strategy:**

```bash
BRANCH_PREFIX=$(echo "$PHASE_BRANCH_TEMPLATE" | sed 's/{.*//')
PHASE_BRANCHES=$(git branch --list "${BRANCH_PREFIX}*" 2>/dev/null | sed 's/^\*//' | tr -d ' ')
```

**For "milestone" strategy:**

```bash
BRANCH_PREFIX=$(echo "$MILESTONE_BRANCH_TEMPLATE" | sed 's/{.*//')
MILESTONE_BRANCH=$(git branch --list "${BRANCH_PREFIX}*" 2>/dev/null | sed 's/^\*//' | tr -d ' ' | head -1)
```

**If no branches found:** Skip to git_tag.

**If branches exist:**

```
## Git Branches Detected

Branching strategy: {phase/milestone}
Branches: {list}

Options:
1. **Merge to main** - Merge branch(es) to main
2. **Delete without merging** - Already merged or not needed
3. **Keep branches** - Leave for manual handling
```

AskUserQuestion with options: Squash merge (Recommended), Merge with history, Delete without merging, Keep branches.

**Squash merge:**

```bash
CURRENT_BRANCH=$(git branch --show-current)
git checkout main

if [ "$BRANCHING_STRATEGY" = "phase" ]; then
  for branch in $PHASE_BRANCHES; do
    git merge --squash "$branch"
    # Strip .planning/ from staging if commit_docs is false
    if [ "$COMMIT_DOCS" = "false" ]; then
      git reset HEAD .planning/ 2>/dev/null || true
    fi
    git commit -m "feat: $branch for v[X.Y]"
  done
fi

if [ "$BRANCHING_STRATEGY" = "milestone" ]; then
  git merge --squash "$MILESTONE_BRANCH"
  # Strip .planning/ from staging if commit_docs is false
  if [ "$COMMIT_DOCS" = "false" ]; then
    git reset HEAD .planning/ 2>/dev/null || true
  fi
  git commit -m "feat: $MILESTONE_BRANCH for v[X.Y]"
fi

git checkout "$CURRENT_BRANCH"
```

**Merge with history:**

```bash
CURRENT_BRANCH=$(git branch --show-current)
git checkout main

if [ "$BRANCHING_STRATEGY" = "phase" ]; then
  for branch in $PHASE_BRANCHES; do
    git merge --no-ff --no-commit "$branch"
    # Strip .planning/ from staging if commit_docs is false
    if [ "$COMMIT_DOCS" = "false" ]; then
      git reset HEAD .planning/ 2>/dev/null || true
    fi
    git commit -m "Merge branch '$branch' for v[X.Y]"
  done
fi

if [ "$BRANCHING_STRATEGY" = "milestone" ]; then
  git merge --no-ff --no-commit "$MILESTONE_BRANCH"
  # Strip .planning/ from staging if commit_docs is false
  if [ "$COMMIT_DOCS" = "false" ]; then
    git reset HEAD .planning/ 2>/dev/null || true
  fi
  git commit -m "Merge branch '$MILESTONE_BRANCH' for v[X.Y]"
fi

git checkout "$CURRENT_BRANCH"
```

**Delete without merging:**

```bash
if [ "$BRANCHING_STRATEGY" = "phase" ]; then
  for branch in $PHASE_BRANCHES; do
    git branch -d "$branch" 2>/dev/null || git branch -D "$branch"
  done
fi

if [ "$BRANCHING_STRATEGY" = "milestone" ]; then
  git branch -d "$MILESTONE_BRANCH" 2>/dev/null || git branch -D "$MILESTONE_BRANCH"
fi
```

**Keep branches:** Report "Branches preserved for manual handling"

</step>

<step name="document_milestone">

**Documentation is part of closing a milestone — do not tag/commit without it.** Per
`docs/05-team-workflow.md`, the **codebase-archaeologist owns documentation**, and that
is **not just `.planning/`** — it is the **product's contributor-facing `docs/`** (the
architecture / public surfaces / how-to that ship with the code) **and** the
`.planning/codebase/` map. The lead **delegates to the archaeologist** at every close:

1. The archaeologist **creates/refreshes `docs/`** to match HEAD (new/changed surfaces
   in, removed out, stale prose corrected) — synthesizing the whole record (ROADMAP,
   per-phase SUMMARY/VERIFICATION, the security review, the typed records) **and the
   actual code read via SMTC**, not one source.
2. It refreshes the `.planning/codebase/` map and runs **`self-audit`** (harness rot).
3. **Gate:** spawn **`anvil-doc-verifier`** — the independent drift check that every
   checkable claim in `docs/` still matches HEAD (a doc that lies about the code is
   worse than no doc). Drift → fix before close.

A milestone whose `docs/` don't match what was built is **not** complete. (Honesty
floor: if the planning record and the code disagree, surface the drift as a finding —
don't paper over it in the docs.)

</step>

<step name="git_tag">

Create git tag:

```bash
git tag -a v[X.Y] -m "v[X.Y] [Name]

Delivered: [One sentence]

Key accomplishments:
- [Item 1]
- [Item 2]
- [Item 3]

See .planning/MILESTONES.md for full details."
```

Confirm: "Tagged: v[X.Y]"

Ask: "Push tag to remote? (y/n)"

If yes:
```bash
git push origin v[X.Y]
```

</step>

<step name="git_commit_milestone">

Commit milestone completion.

```bash
anvil-cc commit "chore: complete v[X.Y] milestone" --files .planning/milestones/v[X.Y]-ROADMAP.md .planning/milestones/v[X.Y]-REQUIREMENTS.md .planning/milestones/v[X.Y]-MILESTONE-AUDIT.md .planning/MILESTONES.md .planning/PROJECT.md .planning/STATE.md
```
```

Confirm: "Committed: chore: complete v[X.Y] milestone"

</step>

<step name="offer_next">

```
✅ Milestone v[X.Y] [Name] complete

Shipped:
- [N] phases ([M] plans, [P] tasks)
- [One sentence of what shipped]

Archived:
- milestones/v[X.Y]-ROADMAP.md
- milestones/v[X.Y]-REQUIREMENTS.md

Summary: .planning/MILESTONES.md
Tag: v[X.Y]

---

## ▶ Next Up

**Start Next Milestone** - requirements → research → roadmap

`/anvil-new-milestone`

<sub>`/new` first → fresh context window</sub>

---

**Optional pre-step:** Capture scope intent before the planning session

`/anvil-discuss-milestone` — crystallize what to build next, then run `/anvil-new-milestone`

---
```

</step>

</process>

<milestone_naming>

**Version conventions:**
- **v1.0** - Initial MVP
- **v1.1, v1.2** - Minor updates, new features, fixes
- **v2.0, v3.0** - Major rewrites, breaking changes, new direction

**Names:** Short 1-2 words (v1.0 MVP, v1.1 Security, v1.2 Performance, v2.0 Redesign).

</milestone_naming>

<what_qualifies>

**Create milestones for:** Initial release, public releases, major feature sets shipped, before archiving planning.

**Don't create milestones for:** Every phase completion (too granular), work in progress, internal dev iterations (unless truly shipped).

Heuristic: "Is this deployed/usable/shipped?" If yes → milestone. If no → keep working.

</what_qualifies>

<success_criteria>

Milestone completion is successful when:

- [ ] MILESTONES.md entry created with stats and accomplishments
- [ ] PROJECT.md full evolution review completed
- [ ] All shipped requirements moved to Validated in PROJECT.md
- [ ] Key Decisions updated with outcomes
- [ ] ROADMAP.md reorganized with milestone grouping
- [ ] Roadmap archive created (milestones/v[X.Y]-ROADMAP.md)
- [ ] Requirements archive created (milestones/v[X.Y]-REQUIREMENTS.md)
- [ ] REQUIREMENTS.md deleted (fresh for next milestone)
- [ ] STATE.md updated with fresh project reference
- [ ] Git tag created (v[X.Y])
- [ ] Milestone commit made (includes archive files and deletion)
- [ ] Requirements completion checked against REQUIREMENTS.md traceability table
- [ ] Incomplete requirements surfaced with proceed/audit/abort options
- [ ] Known gaps recorded in MILESTONES.md if user proceeded with incomplete requirements
- [ ] RETROSPECTIVE.md updated with milestone section
- [ ] Cross-milestone trends updated
- [ ] User knows next step (/anvil-new-milestone)

</success_criteria>
