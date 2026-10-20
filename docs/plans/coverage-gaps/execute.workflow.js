export const meta = {
  name: 'rodcli-coverage-gaps-execute',
  description: 'Execute docs/plans/coverage-gaps/PLAN.md: Go and TS tracks in parallel (sf-tdd, property tests), independent verify per task, then final verification; resumes from PLAN.md checkboxes',
  phases: [
    { title: 'Status', detail: 'read task checkboxes from PLAN.md' },
    { title: 'Implement', detail: 'one task per track at a time, test-first, wip commits' },
    { title: 'Verify', detail: 'independent done-check + diff review, tick [x]' },
    { title: 'Fix', detail: 'one repair round on verify failure' },
    { title: 'Final', detail: 'F01 full suite, leak scan, govulncheck, coverage' },
  ],
}

const REPO = '/home/john/go/src/github.com/agenthands/rod-cli'
const DIR = `${REPO}/docs/plans/coverage-gaps`
const PLAN = `${DIR}/PLAN.md`
const BRANCH = 'test/coverage-gaps'
const COAUTH = 'Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>'
const evidenceFor = id => id.startsWith('E') ? `${DIR}/evidence-ts.md` : `${DIR}/evidence-go.md`

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

const section = id => `the section of ${PLAN} from the heading "### ${id} " up to the next "### " heading, plus the sections "## Spec", "## Decisions" and "## Conventions (every task)", plus the rows of ${DIR}/gap-plan.md whose ids the task names`

const implPrompt = (id, resume) => `You are executing task ${id} of the plan ${PLAN} in ${REPO} on branch ${BRANCH}. Another agent may be working on the other track in the same working tree at the same time: touch only files your task needs and stage only your own paths.
1. Read ${section(id)}. That text is your specification.
2. ${resume
  ? `The heading is [~]: a previous run was interrupted. Run: git -C ${REPO} status --short ; git -C ${REPO} log --oneline -15 . Uncommitted changes to your task's files and wip(${id}) commits are your partial work: continue from them and re-run the task's tests to see where you are.`
  : `Change the heading "### ${id} [ ]" to "### ${id} [~]" and commit only the plan: git -C ${REPO} add docs/plans/coverage-gaps/PLAN.md && git -C ${REPO} commit -m "wip(${id}): start" -m "${COAUTH}"`}
3. Call the Skill tool with skill "sf-tdd" and follow it. Write tests only at the seams of the named rows, one red→green cycle at a time; a bug row gets its failing test before the fix; a property row is a rapid (Go) or fast-check (TS) property with the row's relation and generator.
4. After each green cycle: append its "tdd: ..." line under "## ${id}" in ${evidenceFor(id)} (create the heading if missing), then commit by explicit path: git -C ${REPO} add <paths> ${evidenceFor(id)} && git -C ${REPO} commit -m "wip(${id}): <test name> green" -m "${COAUTH}"
5. Run every "Done when" check and compare with its expected result. Commit remaining task changes the same way. Leave the heading at [~]; the verifier ticks it.
6. Return "ready_for_verify" with the done-check output pasted verbatim, or "blocked" with the exact reason only when the task cannot be done as written.`

const verifyPrompt = id => `You are the independent verifier for task ${id} of ${PLAN} in ${REPO}. You did not write this code; treat it as wrong until the checks prove otherwise.
1. Read ${section(id)}.
2. Run git -C ${REPO} status --short : uncommitted changes to files this task touched are a failure. Read this task's commits: git -C ${REPO} log --oneline --grep="(${id})" and their diffs (git show <hash>), excluding docs/plans.
3. Run every "Done when" check yourself with TMPDIR=$(mktemp -d /tmp/rcv.XXXX); afterwards that dir may hold only leakless-* and rod-cli-daemon.log, and no headless chrome or rod-cli daemon you started may be running. Remove the dir.
4. Check against the rows: every named row is implemented at its seam; property tests use rapid/fast-check with the row's relation and a constrained generator (no filtering inside the property) and an oracle independent of the implementation; example assertions compare against literal values; each bug row has a test that fails on the old code (check by reasoning over the diff, or by reverting the fix in a scratch copy — never modify the repo); ${evidenceFor(id)} has a tdd line with a copied red for every cycle; docs updated where behaviour changed.
5. If all checks hold: change "### ${id} [~]" to "### ${id} [x]" in ${PLAN}, append "verify: ${id} — pass — <one-line summary>" under "## ${id}" in ${evidenceFor(id)}, then: git -C ${REPO} add ${PLAN} ${evidenceFor(id)} && git -C ${REPO} commit -m "checkpoint(${id}): verified" -m "${COAUTH}" . Return pass=true with the commit hash.
6. Otherwise change nothing and return pass=false with the failing command, its output, and the gap.`

const fixPrompt = (id, reason) => `Verification of task ${id} of ${PLAN} in ${REPO} failed:
${reason}
Read ${section(id)}. Call the Skill tool with skill "sf-tdd". Close the gap at the task's seams; when the gap is missing behaviour, write the failing test first. Commit by explicit path with subject "wip(${id}): fix <short>" and the line "${COAUTH}". Re-run every "Done when" check and return "ready_for_verify" with the output, or "blocked" with the reason.`

const markBlocked = (id, reason) => agent(`In ${PLAN}, change the heading for task ${id} ("### ${id} [~]" or "### ${id} [ ]") to "### ${id} [!]" and insert this line directly under the heading: "Blocked: ${String(reason).replace(/"/g, "'").slice(0, 600)}". Then: git -C ${REPO} add ${PLAN} && git -C ${REPO} commit -m "blocked(${id})" -m "${COAUTH}"`, { label: `block:${id}`, phase: 'Verify', effort: 'low' })

const runTask = async t => {
  const impl = await agent(implPrompt(t.id, t.status === '~'), { label: `impl:${t.id}`, phase: 'Implement', schema: IMPL })
  if (!impl || impl.outcome === 'blocked') {
    const why = impl ? impl.notes : 'implement agent died'
    await markBlocked(t.id, why)
    return { id: t.id, ok: false, why }
  }
  let v = await agent(verifyPrompt(t.id), { label: `verify:${t.id}`, phase: 'Verify', schema: VERDICT })
  if (!v || !v.pass) {
    const fix = await agent(fixPrompt(t.id, v ? v.reason : 'verifier died'), { label: `fix:${t.id}`, phase: 'Fix', schema: IMPL })
    if (fix && fix.outcome === 'ready_for_verify') v = await agent(verifyPrompt(t.id), { label: `reverify:${t.id}`, phase: 'Verify', schema: VERDICT })
  }
  if (!v || !v.pass) {
    const why = v ? v.reason : 'verification failed twice'
    await markBlocked(t.id, why)
    return { id: t.id, ok: false, why }
  }
  log(`${t.id} verified (${v.commit})`)
  return { id: t.id, ok: true }
}

const runTrack = async tasks => {
  const done = []
  for (const t of tasks) {
    const r = await runTask(t)
    done.push(r)
    if (!r.ok) break
  }
  return done
}

phase('Status')
const st = await agent(`Run exactly: git -C ${REPO} branch --show-current
Then read ${PLAN}. Every task heading has the form "### Xnn [s] title" (X is G, E or F), where s is one of: space, ~, x, !. Return the branch name and, in file order, every task id with its status character.`, { label: 'status', phase: 'Status', schema: STATUS, effort: 'low' })

if (!st || st.branch.trim() !== BRANCH) { log(`Wrong branch "${st && st.branch}"; expected ${BRANCH}.`); return { stopped: 'wrong-branch' } }
const blocked = st.tasks.filter(t => t.status === '!')
if (blocked.length) { log(`${blocked.map(t => t.id).join(' ')} blocked. Resolve in PLAN.md, then re-run.`); return { stopped: 'blocked', ids: blocked.map(t => t.id) } }
const todo = st.tasks.filter(t => t.status !== 'x')
log(`${st.tasks.length - todo.length}/${st.tasks.length} done; remaining: ${todo.map(t => t.id).join(' ') || 'none'}`)

const goTrack = todo.filter(t => t.id.startsWith('G'))
const tsTrack = todo.filter(t => t.id.startsWith('E'))
const tracks = await parallel([() => runTrack(goTrack), () => runTrack(tsTrack)])
const failed = tracks.filter(Boolean).flat().filter(r => !r.ok)
if (failed.length || tracks.some(x => x === null)) return { stopped: 'track-failed', failed }

phase('Final')
const fin = todo.find(t => t.id === 'F01')
if (fin) {
  const r = await runTask(fin)
  if (!r.ok) return { stopped: 'final-failed', why: r.why }
}
return { done: true }
