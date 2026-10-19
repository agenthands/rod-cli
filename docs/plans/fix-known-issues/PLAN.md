# PLAN: fix every known open issue in rod-cli and godoll, then retest everything

## Resume after a reboot

Open Claude Code in `/home/john/go/src/github.com/agenthands/rod-cli` and say exactly:

`Run the workflow at /home/john/go/src/github.com/agenthands/rod-cli/docs/plans/fix-known-issues/execute.workflow.js`

The workflow reads the task headings below. It skips `[x]`, resumes `[~]` from the uncommitted and `wip(Tnn)` work, and stops at `[!]`.

Status marks, in the heading `### Tnn [s] title`:

| Mark | Meaning |
|---|---|
| `[ ]` | todo |
| `[~]` | in progress |
| `[x]` | verified and committed |
| `[!]` | blocked; the reason is on the next line |

## Conventions (every task)

- **Repos and branches.** rod-cli: `/home/john/go/src/github.com/agenthands/rod-cli` on branch `fix/known-issues`. godoll: `/home/john/go/src/github.com/agenthands/godoll` on branch `fix/known-issues`. rod-cli builds against the godoll working tree (today through `replace ../godoll`; after T10 through a local `go.work`). Plan, evidence and `wip(Tnn): start`/`checkpoint(Tnn)` commits always go in rod-cli; code commits go in the repo the code lives in.
- **TDD.** Every code change, including test-only code, is written with sf-tdd: call the Skill tool with skill `sf-tdd` before the first test. Only tasks whose Seams line says "none" change no code at all; if such a task turns out to need a code change, return `blocked` instead of coding without sf-tdd. Write tests only at the task's **Seams**; the user has confirmed them. Work one red→green cycle at a time. Append one line per cycle to `docs/plans/fix-known-issues/evidence.md` (in rod-cli) under `## Tnn`:
  `tdd: <test name> — red: <copied failure line> — green: <single-file command> passed`
- **Commits.** Stage by explicit path, never `git add -A` or `git add .`. End every message with the attribution line, using exactly this form:
  `git commit -m "<subject>" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"`
  Subjects:
  - `wip(Tnn): <test name> green` after each green cycle (in the repo the code lives in; commit the evidence line in rod-cli with the same subject).
  - `wip(Tnn): start` when marking `[~]`.
  - `checkpoint(Tnn): verified` (verifier only).
- **No new shortcuts.** This plan exists to remove known ceilings. Do not add `ponytail:` comments. Delete each `ponytail:` comment a task resolves. A real browser limit (for example, a cross-origin iframe cannot be read from the parent) gets a plain comment that says so.
- **Live tests.** Run against headless Chrome. Before any `./tests/` run: `go build -o rod-cli .`
- **Seam mocking.** Mock the existing way: save the package-level var, assign a stub, restore with `defer` (see `actions/actions_errpaths_test.go:153-160`).
- **Extension tests use the fresh build.** Run extension vitest/bun with `ROD_CLI_PATH=/home/john/go/src/github.com/agenthands/rod-cli/rod-cli` (after `go build -o rod-cli .`).
- **Leftover daemons.** After any live run, `pgrep -af 'rod-cli.*daemon' | grep -v pgrep` must print nothing; kill only rod-cli daemon processes your runs started.
- **Full suite** (checkpoint tasks run it):

```bash
cd /home/john/go/src/github.com/agenthands/rod-cli && go build -o rod-cli . && go vet ./... && go test ./... -count=1 && export ROD_CLI_PATH="$PWD/rod-cli" && ./rod-cli close; (cd extensions/core && npx vitest run && npx tsc --noEmit) && (cd extensions/pi && npx vitest run && npx tsc --noEmit) && (cd extensions/omp && /home/john/bin/bun test && npx tsc --noEmit)
```

- **Expected test failures.** None. This plan removes the old Baseline flakes; any failure belongs to the task that caused it.

---

### T00 [x] Baseline: record the suite state before any change
**Seams:** none (no code).
**Change:** Run the Full suite command once, and `cd /home/john/go/src/github.com/agenthands/godoll && go vet ./... && go test ./... -count=1 -timeout 15m`. Under `## Baseline` in `docs/plans/fix-known-issues/evidence.md`, write the Go version and every failing test by name with its package and failure line (or `baseline: all green`).
**Done when:** `grep -c '^## Baseline' docs/plans/fix-known-issues/evidence.md` prints `1`.

### T01 [x] cookie-get takes an optional cookie name
**Why:** the skill shows `rod-cli cookie-get session_id`, but the CLI ignores the argument and dumps every cookie, which wastes tokens.
**Seams:** `actions.GetCookies` (live, `newLiveCtx` + fixture server); CLI `cookie-get` (`./tests/`, `runCli`).
**Change:**
- `actions/actions.go:1104` becomes `GetCookies(ctx *types.Context, name string) (string, error)`. Empty `name` keeps today's output exactly. A non-empty `name` returns only cookies with that exact name, in today's line format; no match returns the error `no cookie named %q`.
- `cmd.go:823` `cookie-get [name]`: send `Args["name"]` from `c.Args().First()`. `daemon/daemon.go:347` passes `req.Args["name"]`.
- Update every `GetCookies(` caller and test.
**TDD cycles:**
1. `TestGetCookiesByName` (actions live): set cookies `a=1` and `b=2` on the fixture origin; `GetCookies(ctx,"b")` contains `b` and `2` and does not contain `a=`; `GetCookies(ctx,"zz")` errors with `no cookie named "zz"`.
2. `TestCookieGetCLIByName` (`./tests/`): `cookie-set k1 v1`, `cookie-set k2 v2`, then `--raw cookie-get k2` contains `v2` and not `v1`.
**Done when:** `go build -o rod-cli . && go test ./actions/ -run 'Cookie' -count=1 && go test ./tests/ -run 'TestCookieGetCLIByName|TestStorageCommands' -count=1` prints only `ok` lines.

### T02 [x] Skill docs: fix all 25 open review findings
**Why:** the skill-improver ledger (`.code-improver/rod-cli/ledger.json`, every finding with `"status": "open"`) lists 11 minor and 14 info findings.
**Seams:** none (docs only).
**Change:** in `skills/rod-cli/` only:
- **SKILL.md**
  - Frontmatter description in third person, keeping every trigger phrase: start it with `This skill should be used when the user needs to browse the web, scrape content, ...`.
  - Body in imperative voice (line 13 "The first command you execute", 14-15, 122).
  - Replace "via godoll/network interceptor" with "via the browser's own network interception".
  - Global Options: add `--headless`, `--proxy`, `--profile`, `--user-agent`, `--locale`, `--timezone`, `--cdp-endpoint`, `--console-capture`, `--request-capture`, each with the exact usage text from `./rod-cli --help`. Say that they take effect when a session's daemon spawns (`close` first to change them).
  - Flag-placement rule (line 34): list every command-local flag the skill documents, including screenshot `--selector`/`--out`/`--full-page`, pdf `--out`, console/requests `--clear`, mousewheel `--end`/`--max`, snapshot `-i`, dialog-accept `--prompt`, fill `--submit`, and `--path`, `--body`, `--name`, `--annotate`.
  - Document `stealth-check [url]` and `cdp-traffic` under a Commands subsection, with the usage text from `./rod-cli stealth-check --help` and `./rod-cli cdp-traffic --help`.
  - Document `show` without flags, dialog-accept `--prompt <text>`, screenshot `--selector`, pdf `--out`, and `sessionstorage-get` "(or omit key for all)" (verify against the CLI first).
  - `cookie-get [name]` from T01.
  - Line 61: "Use a selector or ref first; use pixel input (mousemove/mousedown/mouseup) only when ...".
  - Usage example lines 134-135: drop the `click "#submit-button"` that follows `fill --submit`.
  - Replace the extensions comparison section (lines 144-160) with two lines: the skill folder must be installed whole (SKILL.md + references/), and Pi/OMP users can use the typed extensions instead.
  - Add `version:` to the frontmatter with the value printed by `git describe --tags --abbrev=0`.
- **references/**
  - `storage-state.md`: `cookie-get [name]`; add `state-save`/`state-load` round trip and `cookie-set`/`cookie-delete`, with exact syntax from `--help`; sessionstorage-get "omit key for all".
  - `request-mocking.md` line 3: drop "bypassing captchas".
  - `running-code.md` lines 20-21: replace the style-mutation example with a pointer to `rod-cli highlight <ref>`.
  - `element-attributes.md` lines 28-31: delete the duplicated "Extracting just the value" section.
  - `file-uploads.md` line 6: upload, drag and drop take a CSS selector or a ref; show one CSS example.
  - `plugins.md`: replace `../../../docs/plugins/` links with one line naming the docs folder in the rod-cli repo (`docs/plugins/` at https://github.com/agenthands/rod-cli); line 30 uses `/path/to/plugin.js`.
  - Imperative voice in session-management.md, running-code.md, file-uploads.md, request-mocking.md, storage-state.md, plugins.md.
  - SKILL.md lines 156/158 relative extension README links become absolute GitHub URLs.
**Done when:** all hold:
- `grep -c 'bypassing captchas' skills/rod-cli/references/request-mocking.md` prints `0`;
- `grep -rc '\.\./\.\./' skills/rod-cli | grep -v ':0$'` prints nothing;
- `grep -c 'stealth-check \[url\]' skills/rod-cli/SKILL.md` ≥ 1, `grep -c 'state-save' skills/rod-cli/references/storage-state.md` ≥ 1, `grep -c 'this.style' skills/rod-cli/references/running-code.md` prints `0`;
- `head -5 skills/rod-cli/SKILL.md | grep -c '^version:'` prints `1`;
- every flag named in SKILL.md Global Options appears in `go build -o rod-cli . && ./rod-cli --help` (check each).

### T03 [ ] Daemon start: no double spawns, no orphans, no load-dependent timeout
**Why:** under the parallel full suite, `TestStorageCommands` and `TestStartServerWithPpid` fail intermittently and stray `--session default --headless daemon` processes survive. `EnsureDaemon` (`daemon/daemon.go:106-138`) polls 40×250 ms, then gives up without killing the process it spawned, and nothing stops two clients spawning two daemons for one session.
**Seams:** `daemon.EnsureDaemon` and `daemon.StartServer` (in-process, `daemon/daemon_more_test.go` style); CLI `./tests/` `runCli`.
**Change:**
1. **Find the root cause first.** Measure where daemon start time goes under load: add timestamps around everything `StartServer` and the `daemon` command do before `net.Listen` (`daemon.go:141`), run the full Go suite, and read the log. Work that is not needed to answer `ping` moves after `Listen`. Record the measured before/after numbers in evidence.md under `## T03`. Read `TestStartServerWithPpid` (`daemon_more_test.go:141-160`) and fix the cause of its 10 s readiness miss the same way.
2. **One spawn per session.** `EnsureDaemon` takes an exclusive lock on `<port file>.lock` (`syscall.Flock`) around ping → spawn → wait, and pings again after acquiring it. If the repo builds for Windows (`grep -n windows .goreleaser.y*ml`), put the lock in a `_unix.go` file with a Windows counterpart.
3. **Wait on the process, not a fixed count.** Wait while the spawned process is alive, up to the package var `daemonStartTimeout = 30 * time.Second`; return at once if it exits (error includes its exit status and the daemon log path).
4. **No orphans.** On timeout, kill the spawned process before returning the error.
5. `TestStorageCommands` (`tests/cli_test.go`): each `t.Fatalf` includes the command output.
**TDD cycles:**
1. `TestEnsureDaemonConcurrentSpawnsOne`: build the binary as `TestEnsureDaemonSpawnAndClose` does; 5 goroutines call `EnsureDaemon` on one fresh session; afterwards exactly one process matches `pgrep -f -- "--session <session> .*daemon"`. Close it.
2. `TestEnsureDaemonFailsFastWhenProcessExits`: `exePath` is a temp script `#!/bin/sh\nexit 3`; error within 3 s containing `exit status 3`.
3. `TestEnsureDaemonKillsOnTimeout`: temp script `#!/bin/sh\nexec sleep 60`, `daemonStartTimeout = 1 * time.Second`; error returned, and its PID no longer exists.
**Done when:** `go test ./daemon/ -count=1` prints `ok`, and the Go suite run 3 times in a row, `for i in 1 2 3; do go build -o rod-cli . && go test ./... -count=1 2>&1 | grep -E '^(--- FAIL|FAIL|ok )' | grep -v '^ok'; done`, prints nothing, and afterwards `pgrep -af 'rod-cli.*daemon' | grep -v pgrep` prints nothing.

### T04 [ ] Popups get stealth before their first script runs
**Why:** `types/context.go:416` ponytail: tabs a page opens itself (`window.open`, `target=_blank`) are prepared only when first selected, so their first document runs without evasions.
**Seams:** `actions.Click/TabList/TabSelect/Evaluate` (live, `newLiveCtx`); `types/cdp_footprint_test.go` (unchanged rules).
**Change:**
- At browser setup, call `proto.TargetSetAutoAttach{AutoAttach: true, WaitForDebuggerOnStart: true, Flatten: true}` on the browser. Handle `proto.TargetAttachedToTarget` for `page` targets: build the `*rod.Page` for the session, run `preparePage` (under the state lock) unless already prepared, then `proto.RuntimeRunIfWaitingForDebugger{}` on that session. This must also cover targets rod-cli creates itself (initial page, `tab-new`) so they never stay paused.
- No `Runtime.enable` and no `Network` by default: the CDP footprint rules in `types/cdp_footprint_test.go` must keep passing unchanged.
- `TabSelect` no longer reports `adopted` for popups opened by a prepared page. Keep `adopted` for tabs that existed before rod-cli attached (`--cdp-endpoint`).
- Delete the ponytail comment.
- Fixture: `/popup` with `<button onclick="window.open('/probe')">Open</button>`; `/probe` whose first inline script stores an evasion that `preparePage` installs (pick one from `preparePage` that differs from raw headless Chrome, for example `navigator.webdriver`) in `window.__probe`.
**TDD cycles:**
1. `TestPopupPreparedBeforeFirstScript`: navigate to `/popup`, `Click` the Open button by ref, wait for 2 tabs in `TabList`, `TabSelect` the new tab; `Evaluate(ctx,"String(window.__probe)","")` equals the stealth value (a literal, for example `"false"`), and the select output does not contain `adopted`.
2. `TestTabNewNotLeftPaused`: `TabNew(ctx, srv.URL+"/page2")` then `Evaluate(ctx,"document.title","")` returns the page2 title within 5 s.
**Done when:** `go build -o rod-cli . && go test ./types/ ./actions/ -count=1 && go test ./tests/ -run 'TestTabsAndViewCommands|TestSmokeAntigravity' -count=1` prints only `ok` lines, and `grep -c ponytail types/context.go` drops by one.

### T05 [ ] mousewheel scrolls a same-origin iframe under the pointer
**Why:** `actions/actions.go:733` ponytail: an iframe under the pointer is measured from the parent document, so scrolling inside it reports the wrong box.
**Seams:** `actions.MouseWheel` (live, new fixture routes).
**Change:**
- In the anchor/measure path (`actions.go:681` `measureScrollBoxes`, `:735` anchor): when the element under the point (or the ref element) is an `IFRAME`/`FRAME` whose `contentDocument` is readable, descend: anchor = `contentDocument.elementFromPoint(x - rect.left, y - rect.top) || contentDocument.scrollingElement`, recursively, and measure scroll boxes inside that document first, then the frame's ancestors in the parent.
- A cross-origin frame keeps today's behavior, with a plain comment that the parent cannot read it.
- Delete the ponytail comment.
- Fixture `/scroll-iframe`: `<body style="margin:0"><iframe id="f" style="position:fixed;inset:0;width:100%;height:100%;border:0" srcdoc="<body style='margin:0'><div style='height:3000px'>in</div></body>"></iframe></body>`.
**TDD cycles:**
1. `TestMouseWheelScrollsIframeUnderPointer`: navigate to `/scroll-iframe`; `MouseWheel(ctx,0,300,"",0)` matches `^scrolled dy=([1-9]\d*) y=\d+/\d+$`; afterwards `Evaluate(ctx,"document.getElementById('f').contentWindow.scrollY > 0 && window.scrollY === 0","")` == `"true"`.
2. `TestMouseWheelRefIframe`: `MouseWheel(ctx,0,300,"#f",0)` moves the frame's document the same way.
**Done when:** `go build -o rod-cli . && go test ./actions/ -run 'MouseWheel' -count=1` prints `ok`.

### T06 [ ] `snapshot -i` flattens unnamed wrapper ancestors
**Why:** `types/snapshot.go:288` ponytail: `pruneInteractive` keeps unnamed wrappers (list, listitem, generic, group …), so the interactive view still carries structure the agent cannot act on.
**Seams:** `types.pruneInteractive` (pure, literal YAML); `actions.Snapshot` (live).
**Change:**
- After pruning, a kept ancestor is replaced by its kept children when its key has no quoted name, no `[ref=`, and its role is not a landmark or heading. Landmarks kept even when unnamed: `main navigation form dialog alertdialog banner contentinfo region complementary search`.
- Flatten recursively. Delete the ponytail comment.
- Update `TestPruneInteractiveKeepsControlsAndAncestors` to the new expectation (red first).
**TDD cycles:**
1. `TestPruneInteractiveFlattensUnnamedWrappers`: input `- main:\n  - paragraph: hello\n  - list:\n    - listitem:\n      - link "Docs" [ref=s1e4]\n- contentinfo: footer text`; output `- main:\n  - link "Docs" [ref=s1e4]\n`.
2. `TestPruneInteractiveKeepsNamedGroup`: input `- group "Payment":\n  - generic:\n    - radio "Card" [ref=s1e2]\n`; output `- group "Payment":\n  - radio "Card" [ref=s1e2]\n`.
**Done when:** `go test ./types/ -run Prune -count=1 && go test ./actions/ -run Snapshot -count=1` prints `ok` for both.

### T07 [ ] console and requests say when old entries were dropped
**Why:** the 2000-entry cap (`types/context.go:829`) drops the oldest entries silently, so an agent cannot tell its log is incomplete. The cap stays (it bounds daemon memory); the silence goes.
**Seams:** `actions.ConsoleLogs/NetworkRequests` (live, capture on).
**Change:**
- `appendCapped` also returns how many entries it dropped. `Context` keeps a dropped count for each buffer; `ClearConsoleLogs`/`ClearRequests` reset it; getters return it.
- When the count is above 0, `ConsoleLogs`/`NetworkRequests` print `[dropped <N> older entries]` as their first line.
- Replace the ponytail comment with a plain comment that the cap bounds memory.
**TDD cycles:**
1. `TestConsoleReportsDropped`: 2100 `console.log` calls; output's first line == `[dropped 100 older entries]`, second line == `[log] 100`.
2. `TestConsoleClearResetsDropped`: after `ConsoleLogs(ctx,true)`, one more `console.log("x")`; output == `[log] x`.
**Done when:** `go test ./types/ ./actions/ -run 'Console|Request|Network' -count=1` prints `ok` for both.

### T08 [ ] godoll: typing returns errors from typo keystrokes
**Why:** in `humanize/typing.go` `typeWithLogic`, wrong-key and backspace calls are `_ = typeFn(...)` / `_ = backspaceFn()`. If one fails, typing continues and can leave wrong text with a nil error.
**Seams:** godoll `typeWithLogic` (internal, the buffer-simulating `typeFn`/`backspaceFn` pattern in `humanize/internal_coverage_test.go`).
**Change:** every `typeFn` and `backspaceFn` call in `typeWithLogic` returns its error. Commit in godoll.
**TDD cycles:**
1. `TestTypeWithLogic_ReturnsTypoKeyErrors`: `sleep` stubbed to a no-op; TypoRate 1.0; for k in 1..15 and 50 iterations each, `typeFn`/`backspaceFn` share a call counter and fail with a sentinel error on call k. If the counter reached k, the returned error `errors.Is` the sentinel; if it never reached k, the error is nil and the buffer equals the input.
**Done when:** `cd /home/john/go/src/github.com/agenthands/godoll && go vet ./humanize/ && go test ./humanize/ -count=1` prints `ok`.

### T09 [ ] godoll: live typing examples never hang
**Why:** `examples/typing/live_typing_test.go:27` `page.MustElement("#searchbox_input")` retries forever because DuckDuckGo changed its page; the package hits the 10 m test timeout and `TestLiveTyping_Google` never runs.
**Seams:** godoll `examples/typing` package tests (live).
**Change:**
- Use stable selectors (DuckDuckGo `input[name="q"]`; check the Google test's selector the same way) with `page.Timeout(20*time.Second)` around every element wait.
- Call `internaltesting.SkipIfURLUnreachable` before navigating. If the element does not appear (bot wall, consent page, changed site), `t.Skipf` naming the site and selector.
- Typing assertions stay hard failures.
**TDD cycles:**
1. `live typing never hangs`: red is `cd /home/john/go/src/github.com/agenthands/godoll && go test ./examples/typing/ -count=1 -timeout 120s` today (it fails with `panic: test timed out`). Green: the same command prints `ok` (tests pass or skip with a reason).
**Done when:** `cd /home/john/go/src/github.com/agenthands/godoll && go test ./examples/typing/ -count=1 -timeout 120s -v 2>&1 | grep -E '^(--- (PASS|SKIP|FAIL)|ok|FAIL)'` shows no FAIL line and ends with `ok`.

### T10 [ ] Build from GitHub without the godoll sibling
**Why:** `go.mod:49` `replace github.com/agenthands/godoll => ../godoll` makes a clean clone of rod-cli fail to build unless godoll sits next to it.
**Seams:** none (module wiring and docs; no Go code changes).
**Change:**
- Delete the `replace` line from `go.mod`. Run `GOWORK=off go get github.com/agenthands/godoll@939b13e && GOWORK=off go mod tidy` (939b13e is godoll `origin/main`).
- Create a local, git-ignored `go.work` for development: `go work init . ../godoll` (`go.work` is already in `.gitignore`; confirm with `git check-ignore go.work`).
- In the README's development/build section, or `CONTRIBUTING.md` if present, add: `To build against a local godoll checkout, run: go work init . ../godoll`.
**Done when:** all hold:
- `grep -c '^replace github.com/agenthands/godoll' go.mod` prints `0`;
- `git check-ignore go.work` prints `go.work`;
- `go build -o rod-cli . && GOWORK=off go build ./...` both succeed;
- a clean clone builds: `rm -rf /tmp/claude-1000/-home-john-go-src-github-com-agenthands-rod-cli/f5adcbb0-fa09-40c9-8840-6df439605546/scratchpad/clone && git clone -q --branch fix/known-issues /home/john/go/src/github.com/agenthands/rod-cli /tmp/claude-1000/-home-john-go-src-github-com-agenthands-rod-cli/f5adcbb0-fa09-40c9-8840-6df439605546/scratchpad/clone && cd /tmp/claude-1000/-home-john-go-src-github-com-agenthands-rod-cli/f5adcbb0-fa09-40c9-8840-6df439605546/scratchpad/clone && GOWORK=off go build -buildvcs=false ./... && echo CLONE_BUILD_OK` prints `CLONE_BUILD_OK`.

### T11 [ ] Final: retest everything three times, zero failures
**Seams:** none (no code).
**Change:** run, in order:
1. The Full suite command three times in a row. Each run must exit 0 with no failing test.
2. `cd /home/john/go/src/github.com/agenthands/godoll && go vet ./... && go test ./... -count=1 -timeout 15m`. It must print no FAIL. Live tests that skip with a reason are fine.
3. `cd /home/john/go/src/github.com/agenthands/rod-cli && go run golang.org/x/vuln/cmd/govulncheck@latest ./...` prints `No vulnerabilities found.`, and the same in godoll.
4. `go test ./tests/ -run 'TestSmokeAntigravity|TestInputsAndForms|TestStorageCommands' -count=5` prints `ok`.
5. `pgrep -af 'rod-cli.*daemon' | grep -v pgrep` prints nothing.
6. Append a `## Summary` to evidence.md: each task, the three full-suite results, and the godoll result.

If anything fails, return `blocked` with the failing test, its output, and the task whose files it touches.
**Done when:** every step above holds.
