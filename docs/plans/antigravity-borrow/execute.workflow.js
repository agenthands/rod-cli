export const meta = {
  name: 'rodcli-antigravity-execute',
  description: 'Execute docs/plans/antigravity-borrow/PLAN.md task by task (sf-tdd), verify independently, checkpoint-commit; resumes from PLAN.md checkboxes',
  phases: [
    { title: 'Status', detail: 'read task checkboxes from PLAN.md' },
    { title: 'Implement', detail: 'one task, test-first, wip commits' },
    { title: 'Verify', detail: 'independent done-check + diff review, tick [x]' },
    { title: 'Fix', detail: 'one repair round on verify failure' },
  ],
}

const REPO = '/home/john/go/src/github.com/agenthands/rod-cli'
const PLAN = `${REPO}/docs/plans/antigravity-borrow/PLAN.md`
const EVIDENCE = `${REPO}/docs/plans/antigravity-borrow/evidence.md`
const BRANCH = 'feat/antigravity-borrow'
const COAUTH = 'Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>'

const STATUS = { type: 'object', properties: {
  branch: { type: 'string' },
  tasks: { type: 'array', items: { type: 'object', properties: {
    id: { type: 'string' }, status: { type: 'string', enum: [' ', '~', 'x', '!'] } }, required: ['id', 'status'] } } },
  required: ['branch', 'tasks'] }
const IMPL = { type: 'object', properties: {
  id: { type: 'string' }, outcome: { type: 'string', enum: ['ready_for_verify', 'blocked'] },
  done_check_output: { type: 'string' }, notes: { type: 'string' } },
  required: ['id', 'outcome', 'done_check_output', 'notes'] }
const VERDICT = { type: 'object', properties: {
  id: { type: 'string' }, pass: { type: 'boolean' }, reason: { type: 'string' }, commit: { type: 'string' } },
  required: ['id', 'pass', 'reason', 'commit'] }

const section = id => `the section of ${PLAN} from the heading "### ${id} " up to the next "### " heading, plus the "## Conventions (every task)" section`

const implPrompt = (id, resume) => `You are executing task ${id} of the plan ${PLAN} in the repo ${REPO} on branch ${BRANCH}.
1. Read ${section(id)}. That text is your specification.
2. ${resume
  ? `The heading is [~]: a previous run was interrupted, most likely by a reboot. Run: git -C ${REPO} status --short ; git -C ${REPO} log --oneline -10 . Uncommitted changes and wip(${id}) commits are your partial work: continue from them, and re-run the task's tests to see where you are.`
  : `Change the heading "### ${id} [ ]" to "### ${id} [~]" and commit only the plan: git -C ${REPO} add docs/plans/antigravity-borrow/PLAN.md && git -C ${REPO} commit -m "wip(${id}): start" -m "${COAUTH}"`}
3. If the task's Seams line says "none", the task changes no code: skip to step 5, and if it turns out to need a code change, return "blocked". For every other task, call the Skill tool with skill "sf-tdd" before writing any test or code, and follow it for all coding. The Seams in the task are confirmed by the user: write tests only at those seams, one red→green cycle at a time, using the test names in the task.
4. After each green cycle: append its "tdd: ..." line to ${EVIDENCE} under "## ${id}" (create the heading if missing), then commit the touched files by explicit path: git -C ${REPO} add <paths> ${EVIDENCE} && git -C ${REPO} commit -m "wip(${id}): <test name> green" -m "${COAUTH}"
5. Run every "Done when" command exactly as written and compare with its expected output. Commit any remaining task changes the same way. Leave the heading at [~]; the verifier ticks it.
6. Return outcome "ready_for_verify" with the done-check output pasted verbatim. Return outcome "blocked" with the exact reason and what you tried only when the task cannot be done as written (a premise in the plan is false, or a required tool fails).`

const verifyPrompt = id => `You are the independent verifier for task ${id} of ${PLAN} in ${REPO}. You did not write this code; treat it as wrong until the checks prove otherwise.
1. Read ${section(id)}, and the "## Baseline" section of ${EVIDENCE}.
2. Run: git -C ${REPO} status --short . Uncommitted changes to files this task names are a failure ("implementer left uncommitted work"). Read this task's diff: git -C ${REPO} diff "$(git -C ${REPO} log --fixed-strings --grep="wip(${id}): start" --format=%H -1)~1" HEAD -- . ':!docs/plans'
3. Run every "Done when" command exactly as written. Each must give its expected output. A failure listed under Baseline is pre-existing; any other failure fails this task.
4. Check the diff against the task: every item under "Change" is implemented; tests sit only at the listed Seams and use the listed names; each assertion compares against a literal expected value, not one recomputed the way the code computes it; ${EVIDENCE} has a "tdd:" line with a copied red failure for every cycle; no file outside the ones the task names changed, except tests and docs it names.
5. If all checks hold: change "### ${id} [~]" to "### ${id} [x]" in ${PLAN}, append "verify: ${id} — pass — <one-line summary of the done-check output>" under "## ${id}" in ${EVIDENCE}, then run: git -C ${REPO} add docs/plans/antigravity-borrow && git -C ${REPO} commit -m "checkpoint(${id}): verified" -m "${COAUTH}" . Return pass=true with the new commit hash.
6. If any check fails: change nothing, and return pass=false with the failing command, its output, and the gap.`

const fixPrompt = (id, reason) => `Verification of task ${id} of ${PLAN} in ${REPO} failed:
${reason}
Read ${section(id)}. Call the Skill tool with skill "sf-tdd". Close the gap at the task's listed Seams; when the gap is missing behaviour, write the failing test first. Commit by explicit path: git -C ${REPO} add <paths> && git -C ${REPO} commit -m "wip(${id}): fix <short>" -m "${COAUTH}" . Re-run every "Done when" command and return outcome "ready_for_verify" with the output, or "blocked" with the reason.`

const markBlocked = (id, reason) => agent(`In ${PLAN}, change the heading for task ${id} ("### ${id} [~]" or "### ${id} [ ]") to "### ${id} [!]" and insert this line directly under the heading: "Blocked: ${String(reason).replace(/"/g, "'").slice(0, 600)}". Then run: git -C ${REPO} add docs/plans/antigravity-borrow/PLAN.md && git -C ${REPO} commit -m "blocked(${id})" -m "${COAUTH}"`, { label: `block:${id}`, phase: 'Verify', effort: 'low' })

phase('Status')
const st = await agent(`Run exactly: git -C ${REPO} branch --show-current
Then read ${PLAN}. Every task heading has the form "### Tnn [s] title", where s is one of: space, ~, x, !. Return the branch name and, in file order, every task id (for example "T04") with its status character.`, { label: 'status', phase: 'Status', schema: STATUS, effort: 'low' })

if (!st || st.branch.trim() !== BRANCH) { log(`Wrong branch "${st && st.branch}"; expected ${BRANCH}. Stopping.`); return { stopped: 'wrong-branch' } }
const blocked = st.tasks.find(t => t.status === '!')
if (blocked) { log(`${blocked.id} is [!] blocked. Resolve it in PLAN.md (set it back to [ ] or [~]), then re-run.`); return { stopped: 'blocked', id: blocked.id } }
const todo = st.tasks.filter(t => t.status !== 'x')
log(`${st.tasks.length - todo.length}/${st.tasks.length} done; remaining: ${todo.map(t => t.id).join(' ') || 'none'}`)

for (const t of todo) {
  const impl = await agent(implPrompt(t.id, t.status === '~'), { label: `impl:${t.id}`, phase: 'Implement', schema: IMPL })
  if (!impl || impl.outcome === 'blocked') {
    const why = impl ? impl.notes : 'implement agent died'
    await markBlocked(t.id, why)
    return { stopped: 'blocked', id: t.id, why }
  }
  let v = await agent(verifyPrompt(t.id), { label: `verify:${t.id}`, phase: 'Verify', schema: VERDICT })
  if (!v || !v.pass) {
    const fix = await agent(fixPrompt(t.id, v ? v.reason : 'verifier died'), { label: `fix:${t.id}`, phase: 'Fix', schema: IMPL })
    if (fix && fix.outcome === 'ready_for_verify') v = await agent(verifyPrompt(t.id), { label: `reverify:${t.id}`, phase: 'Verify', schema: VERDICT })
  }
  if (!v || !v.pass) {
    const why = v ? v.reason : 'verification failed twice'
    await markBlocked(t.id, why)
    return { stopped: 'blocked', id: t.id, why }
  }
  log(`${t.id} verified (${v.commit})`)
}
return { done: true }
