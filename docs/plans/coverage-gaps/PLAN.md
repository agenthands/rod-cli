# PLAN: close rod-cli test-coverage gaps and the bugs they expose

## Resume after a reboot

Open Claude Code in `/home/john/go/src/github.com/agenthands/rod-cli` and say exactly:

`Run the workflow at /home/john/go/src/github.com/agenthands/rod-cli/docs/plans/coverage-gaps/execute.workflow.js`

The workflow reads the task headings below:
- It skips `[x]`.
- It resumes `[~]` from the uncommitted and `wip(Tnn)` work.
- It stops a track at `[!]`.

Status marks in a heading `### Tnn [s] title`: `[ ]` todo, `[~]` in progress, `[x]` verified and committed, `[!]` blocked (the reason is on the next line).

## Spec

`docs/plans/coverage-gaps/gap-plan.md` is the spec. Every task names row ids from it, such as `daemon:D3` or `root:root-2`. Read those rows first:
- the seam;
- the promise, which for a property gives the exact relation and generator;
- the bug it catches.

The seams in gap-plan.md were confirmed by the user on 2026-10-20 ("All of A–D"). Write tests only at those seams.

## Decisions (user-confirmed or defaulted; apply them as written)

- **Session names** must match `^[A-Za-z0-9._-]{1,64}$` and must not be `.` or `..`. Reject any other name in Go (cmd.go, before any file op) and in TS (`extensions/core`, before exec), with the error `invalid session name %q: use letters, digits, '.', '_' or '-' (max 64)`.
- **Empty values:** an empty cookie or storage value is allowed.
- **`waitForSelector`:** an exec rejection is propagated (no retry). With `timeoutMs <= 0` it checks once and does not poll.
- **Partial config:** a partial `rod-cli.yaml` is decoded onto a deep copy of the defaults, so keys the file does not set keep their default values.
- **Mock routes:** when patterns overlap, the most recently added route wins (Playwright semantics). Rules are kept in insertion order, never map order.
- **`pruneInteractive`:** its doc comment is the spec. Make the code match it.
- **`eval --json` output shape and `--max-output` skipping JSON:** keep today's behaviour. root-7 pins it.
- **Port files** are written 0600, parsed with `strconv.Atoi`, and dialled only at `127.0.0.1:<port>`.
- **The daemon `/execute`** rejects any request that carries an `Origin` header, has a non-`application/json` Content-Type, or has a Host other than `127.0.0.1:<port>`. The rod-cli client must still work, so set the headers it needs.

## Conventions (every task)

- **Branch.** Work in `/home/john/go/src/github.com/agenthands/rod-cli` on branch `test/coverage-gaps`. Two tracks run at the same time in this one working tree:
  - Go track: tasks `G..`; evidence goes to `docs/plans/coverage-gaps/evidence-go.md`.
  - TS track: tasks `E..`; evidence goes to `docs/plans/coverage-gaps/evidence-ts.md`.

  Touch only your own track's files. Before committing, `git diff --cached --stat` must list only your paths.
- **TDD.** Call the Skill tool with skill `sf-tdd` before the first test. Work one red→green cycle at a time.
  - **Go properties:** use `pgregory.net/rapid`.
  - **TS properties:** use `fast-check`.
  - **Bug fixes (group D):** a test that fails on today's code comes first, then the fix.
  - **Evidence:** append one line per cycle under `## Tnn` in your track's evidence file:
    `tdd: <test name> — red: <copied failure line, or seed <n>: <shrunk counterexample>> — green: <single-file command> passed`
- **Commits.**
  - Stage by explicit path, never `git add -A` or `git add .`.
  - Every message ends with exactly: `git commit -m "<subject>" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"`
  - Subjects:
    - `wip(Tnn): <test name> green` after each green cycle.
    - `wip(Tnn): start` when marking `[~]`.
    - `checkpoint(Tnn): verified` (verifier only).
- **Leaks.** `/tmp` is RAM, so tests must leave nothing behind.
  - Every test that touches port files, locks, profiles or the daemon sets `t.Setenv("TMPDIR", t.TempDir())`, or a short `os.MkdirTemp("/tmp", "rc-")` removed in `t.Cleanup` when Chrome needs a TMPDIR of 40 characters or less.
  - Launch Chrome only through `internal/testchrome` or `newLiveCtx`.
  - Close every rod-cli session you start.
  - Run tests with `TMPDIR=$(mktemp -d /tmp/rc.XXXX)` and check that the dir holds only `leakless-*` and `rod-cli-daemon.log` afterwards.
- **Live tests.** Before any `./tests/` run: `go build -o rod-cli .`; delete `./rod-cli` when done.
- **Seam mocking.** Save the package-level var, assign a stub, and restore it in `t.Cleanup`.
- **Shortcuts.** Mark deliberate shortcuts with a `// ponytail:` comment naming the ceiling and the upgrade path.
- **Docs.** When a fix changes behaviour a user sees, update the docs that describe it in the same commit: `skills/rod-cli/SKILL.md`, `skills/rod-cli/references/*.md`, `USAGE.md`, `README.md`, `docs/onboarding/*.md`, `extensions/*/README.md`.
- **Go checks** for a task: `go vet ./... && go test <touched packages> -count=1`.
- **TS checks** for a task: `(cd extensions/core && npx vitest run && npx tsc --noEmit) && (cd extensions/pi && npx vitest run && npx tsc --noEmit) && (cd extensions/omp && /home/john/bin/bun test && npx tsc --noEmit)`.

---

## Go track

### G01 [x] Daemon test hygiene and session-name validation
**Rows:** Prerequisite "Daemon test hygiene (S8)", daemon:S6.
**Change:**
- Every daemon test that pings, writes or deletes port files sets a private TMPDIR. This covers TestListSessionsPrunesDead, TestInProcessDaemonPingAndActions, TestStartServerAnswersPing and the EnsureDaemon tests.
- Session names are validated in Go per the Decisions section, before any file op.
- Docs: add the session-name rule where `--session` is documented.

**Done when:** `go vet ./... && go test ./daemon/ ./ -count=1` pass. A session named `x/../../../etc/foo` fails with `invalid session name`.

### G02 [x] Daemon /execute and port-file hardening
**Rows:** daemon:D3/S1, daemon:S7, daemon:D2, daemon:D4/S2, daemon:S4.
**Change:** as the rows and the Decisions section say. The ping-only timeout must not shorten real commands.
**Done when:** `go test ./daemon/ ./ -count=1` passes with the new red→green tests, and `go build -o rod-cli . && ./rod-cli --session g02 --headless sessions && ./rod-cli --session g02 close` works.

### G03 [x] Daemon robustness
**Rows:** daemon:execSem (panic must not wedge the session), daemon:D6/S3 (numeric args), daemon:D5 (close cleanup ordering), daemon:S5 (ping does not reset the idle timer; add an idle-timer seam).
**Done when:** `go test ./daemon/ -count=1` passes.

### G04 [x] Config and stealth resolution
**Rows:**
- types:T4 / types:LoadConfig, including the partial-config merge decision;
- types:T1 / types:platform (property);
- types:T2;
- types:T3 (property);
- types:T5;
- types:T10.

**Done when:** `go test ./types/ -count=1` passes.

### G05 [x] Snapshot and routes
**Rows:** types:T6 (property), types:T7, types:T8 and types:route-order (insertion order; the newest route wins), plus the pruneInteractive decision.
**Done when:** `go test ./types/ ./actions/ -run 'Snapshot|Prune|Route' -count=1` passes.

### G06 [x] Actions with seams (no Chrome)
**Rows:**
- actions:A2 / actions:parseKey: map the named keys; an unknown key gives `unknown key %q`.
- actions:A5 / actions:wrap-nil.
- actions:A7.
- actions:A10.
- actions:A11.
- actions:A13.

**Done when:** `go test ./actions/ -run 'Press|MouseWheel|Drag|StealthCheck' -count=1` passes. The docs for `press` list the supported key names.

### G07 [ ] Actions, live
**Rows:**
- actions:A3 / actions:selectOne;
- actions:selectByValue (CSS.escape);
- actions:A6 / actions:evaluate;
- actions:A1 / actions:state-0644;
- actions:tabnew-orphan;
- actions:A4 (Fill property, `-rapid.checks=30`, one ctx).

**Done when:** `go test ./actions/ -run 'Select|Evaluate|State|TabNew|Fill' -count=1` passes.

### G08 [ ] CLI parsing and output
**Rows:** root:root-2 (property), root:root-3 / root:trailing-flag, root:root-5 / root:json-detect, root:root-4, root:root-6, root:root-7 / root:eval-json-shape (pin today's shape).
**Done when:** `go test ./ -count=1` passes.

### G09 [ ] Flag forwarding properties
**Rows:** root:root-1 (with daemon:D1 merged), root:root-9 (extract `stealthFlagsFrom(c)` from main.go first, as a pure refactor with no behaviour change).
**Done when:** `go test ./ -run 'Forward|StealthFlags' -count=1` passes, and `go test ./ -count=1` passes.

### G10 [ ] Internal packages and cosmetic fixes
**Rows:**
- internal:cdpproxy-read-normalize-property and internal:cdpproxy-drop;
- internal:cdpproxy-traffic-ringbuffer-model;
- internal:plugin-engine-concurrent-hooks-race and internal:goja-race (add `go test -race ./internal/plugin/...` to the Makefile test target);
- internal:plugin-bind-twice-single-dispatch and internal:bind-twice;
- internal:utils-ishttp-scheme-case and internal:ishttp.

**Cosmetic, no test:**
- The cdp_traffic.go and scanner preview cut stays on a UTF-8 rune boundary.
- `GetChinaZoneTime` handles its parse error.
- snapshotRootExpr never emits an octal literal.

**Done when:** `go test ./internal/... ./utils/ -count=1 && go test -race ./internal/plugin/... -count=1` pass.

### G11 [ ] Go checkpoint: full Go suite
**Change:** Run `go build -o rod-cli . && go vet ./... && TMPDIR=$(mktemp -d /tmp/rc.XXXX) go test ./... -count=1`, then `rm -f rod-cli`. Fix nothing here: if a test fails, return blocked and name the task whose files it touches.
**Done when:** the command exits 0, and the TMPDIR holds only `leakless-*` and `rod-cli-daemon.log`.

---

## TS track

### E01 [!] fast-check, argv builder and timeout properties
Blocked: E01 is blocked because fast-check could not be installed. The two bug fixes are done and committed. The two property tests are not written.

Why: running `cd extensions/core && npm install -D fast-check` fails with `npm error errno DEPTH_ZERO_SELF_SIGNED_CERT ... request to https://localhost:8443/npm/fast-check failed, reason: self-signed certificate`. The ~/.npmrc sets `registry=https://localhost:8443/npm/`, and that server's cert is self-signed (CN=localhost, created Oct 19). I tried to trust that cert for this one install with NODE_EXTRA_CA_CERTS, and the auto-mode classifier denied it as [
**Rows:** Prerequisite "TS: add fast-check", ext:EXT-1 / ext:screenshot-selector (emit `--selector=<sel>` as one token), ext:EXT-5 / ext:timeoutFor.
**Change:** `cd extensions/core && npm install -D fast-check`. Commit package.json and the lockfile.
**Done when:** the TS checks pass.

### E02 [ ] Extension lifecycle, waits and validation
**Rows:**
- ext:EXT-2 / ext:pi-lifecycle;
- ext:EXT-3 / ext:sleep-leak, applying the waitForSelector decision;
- ext:EXT-9;
- ext:EXT-12, the session charset in core, per the Decisions section.

**Cosmetic:** validate.ts now says "characters", not "bytes".
**Done when:** the TS checks pass.

### E03 [ ] Extensions, live against the repo build
**Rows:** ext:EXT-6 (add `<select id="c">` and a scrollable `#pane` to the pi FIXTURE_HTML), ext:EXT-7.
**Change:** These run through the pi integration harness against `<repo>/rod-cli`. Build it with `go build -o rod-cli .` in the repo root, and delete it after. If a Go fix that this needs is not on the branch yet, wait for it: check `git log --oneline` for the Go task's checkpoint, and return blocked only if it is still missing after the Go track finishes.
**Done when:** the TS checks pass, and `extensions/pi` holds no new untracked files.

---

## Final

### F01 [ ] Final verification and ship
**Change:**
1. Full suite with a private TMPDIR and a leak scan:

   ```
   cd /home/john/go/src/github.com/agenthands/rod-cli && T=$(mktemp -d /tmp/rc.XXXX) && go build -o rod-cli . && go vet ./... && TMPDIR=$T go test ./... -count=1 && go test -race ./internal/plugin/... -count=1 && (cd extensions/core && npx vitest run && npx tsc --noEmit) && (cd extensions/pi && npx vitest run && npx tsc --noEmit) && (cd extensions/omp && /home/john/bin/bun test && npx tsc --noEmit); ./rod-cli close; rm -f rod-cli; ls -A $T
   ```

2. Run `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`.
3. Measure coverage again: `go test ./... -count=1 -coverpkg=./... -coverprofile=/tmp/claude-1000/cov2.out`. Record the total next to the starting 82.2% in evidence-go.md under `## Summary`, then remove the profile.

**Done when:**
- the suite exits 0;
- `$T` holds only `leakless-*` and `rod-cli-daemon.log`;
- `git status --short` lists no untracked files outside `docs/plans`;
- govulncheck prints `No vulnerabilities found.`
