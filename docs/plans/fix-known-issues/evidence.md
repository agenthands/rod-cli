# Evidence

## Baseline

Go: go version go1.26.9 linux/amd64 (2026-10-19, branch fix/known-issues in both repos; rod-cli suite and godoll suite ran concurrently).

rod-cli Full suite: `go build` and `go vet ./...` clean. `go test ./... -count=1` FAILED:
- `github.com/agenthands/rod-cli/daemon` TestStartServerWithPpid — `daemon_more_test.go:158: server with ppid did not become ready`
- `github.com/agenthands/rod-cli/tests` TestStorageCommands — `cli_test.go:71: Failed to set localstorage: exit status 1`

All other Go packages ok. The failed `go test` short-circuited `export ROD_CLI_PATH` and `./rod-cli close`, so the extension suites ran against `rod-cli` on PATH (`/home/john/bin/rod-cli`): core vitest 142/142 passed, pi vitest 40/40 passed, omp bun 32/32 passed, all three `tsc --noEmit` clean. Two default-session daemons were left behind (repo build and `/home/john/bin/rod-cli`); `./rod-cli close` removed both.

godoll (`go vet ./...` clean; `go test ./... -count=1 -timeout 15m`) FAILED:
- `github.com/agenthands/godoll/examples/typing` TestLiveTyping_DuckDuckGo — `panic: test timed out after 15m0s` (hung in `page.MustElement` at `examples/typing/live_typing_test.go:27`)

The other 23 godoll packages were ok.

## T00

verify: T00 — pass — `grep -c '^## Baseline'` prints 1; no code changed (diff only docs/plans); recorded failure lines match the source (daemon_more_test.go:158, cli_test.go:71, live_typing_test.go:27)

## T01

tdd: TestGetCookiesByName — red: actions/actions_full_test.go:882:31: too many arguments in call to GetCookies — green: go test ./actions/ -run TestGetCookiesByName -count=1 passed
tdd: TestCookieGetCLIByName — red: cli_test.go:73: cookie-get k2 returned: "Cookies:\n- testcookie: 123 (domain: 127.0.0.1)\n- k1: v1 (domain: 127.0.0.1)\n- k2: v2 (domain: 127.0.0.1)\n\n" — green: go test ./tests/ -run TestCookieGetCLIByName -count=1 passed
verify: T01 — pass — done-check printed only `ok actions` and `ok tests` (TestGetCookiesByName, TestCookies, TestErrPaths_Cookies, TestCookieGetCLIByName, TestStorageCommands); diff limited to actions.go/cmd.go/daemon.go + tests; no leftover daemons

## T02

verify: T02 — pass — captchas grep 0; no ../../ links; stealth-check [url] 1, state-save 2, this.style 0; version: line 1 (v1.7 = git describe); all 13 Global Options flags found in ./rod-cli --help; docs-only diff limited to skills/rod-cli/ (SKILL.md + 7 references), help/usage text matches the CLI

## T03

measure (before, full Go suite, timestamps in the spawned daemon): main-pkg-init→port file written median 1.1 ms, max 1.6 ms over 101 spawns (LoadConfig 0.1 ms, ResolveStealth 0.2 ms, NewContext 0.0 ms, Listen 0.1 ms). Client spawn→ready median 251 ms, which is the first 250 ms poll sleep. One client gave up after 10.01 s even though its daemon (session default) wrote its port file 3 ms after spawn. That daemon was left running as an orphan. In the same run TestStartServerWithPpid failed at 10.03 s with its port file written 0.07 ms after StartServer entered.
root cause: startup is not slow, so no work moves after Listen. cmd_test.go clearPortFiles globbed and deleted every rod-cli-*.port in the shared TMPDIR. The root-package tests run in parallel with ./daemon/ and ./tests/, so it deleted live daemons' port files. Their clients could no longer find them, spawned duplicates, and left the originals orphaned. Fix: clearPortFiles points TMPDIR at t.TempDir() instead.
tdd: TestStartServerWithPpid (regression) — red: --- FAIL: TestStartServerWithPpid (10.03s) (3/3 runs of `go test . ./daemon/ -run 'TestSessions|TestStartServerWithPpid|TestRunApp|Port' -count=1`) — green: same command 3/3 ok
tdd: TestEnsureDaemonConcurrentSpawnsOne — red: daemon_more_test.go:277: want exactly 1 daemon for session concurrent-1792402135692698720, got 4: [3112307 3112308 3112313 3112314] — green: go test ./daemon/ -run TestEnsureDaemonConcurrentSpawnsOne -count=1 passed
tdd: TestEnsureDaemonFailsFastWhenProcessExits — red: daemon_more_test.go:300: took 10.015105414s, want < 3s; daemon_more_test.go:303: error "failed to start daemon" does not contain "exit status 3" — green: go test ./daemon/ -run TestEnsureDaemonFailsFastWhenProcessExits -count=1 passed
tdd: TestEnsureDaemonKillsOnTimeout — red: daemon_more_test.go:327: spawned process 3132569 still alive after the timeout error — green: go test ./daemon/ -run TestEnsureDaemonKillsOnTimeout -count=1 passed
verify: T03 — pass — go test ./daemon/ ok (96.4s); 3x build+go test ./... printed no FAIL lines; no rod-cli daemons left; GOOS=windows build of ./daemon ok

## T02b

tdd: TestDefaultProfileRootIsUserCache — red: types/profile_root_test.go:10:9: undefined: DefaultProfileRoot — green: go test ./types/ -run TestDefaultProfileRootIsUserCache -count=1 passed
tdd: TestSweepRemovesDeadOwnerProfiles — red: types/profile_root_test.go:59:2: undefined: SweepStaleProfiles — green: go test ./types/ -run TestSweepRemovesDeadOwnerProfiles -count=1 passed
tdd: TestDaemonSIGTERMRemovesProfile — red: profile_test.go:83: want 1 profile dir under /tmp/TestDaemonSIGTERMRemovesProfile3246516978/001/rod-cli/profiles, got [] — green: go test ./daemon/ -run TestDaemonSIGTERMRemovesProfile -count=1 passed
tdd: TestDaemonKillProfileSweptOnNextStart — red (sweep call disabled, since the sweep was already wired in cycle 3): profile_test.go:111: dead daemon's profile /tmp/TestDaemonKillProfileSweptOnNextStart1636159535/001/rod-cli/profiles/xIirUPEgOy not swept by the next start (stat err <nil>) — green: go test ./daemon/ -run TestDaemonKillProfileSweptOnNextStart -count=1 passed
tdd: TestBrowserCloseRemovesProfile — red: close_cleanup_test.go:39: profile /tmp/rod/user-data/2bf2e7ab65d1e5c2 still exists after Close (stat err <nil>) — green: cd /home/john/go/src/github.com/agenthands/godoll && go test ./browser/ -run TestBrowserCloseRemovesProfile -count=1 passed
verify: T02b — pass — types/daemon/godoll browser ok; profiles before=0/0 after=0/0; no rod dirs; no leftover daemons

## T04

tdd: TestPopupPreparedBeforeFirstScript — red: actions_full_test.go:1371: TabSelect output = "Selected tab 0 AD206FD5 Probe (http://127.0.0.1:46167/probe); adopted, stealth applies from next navigation (reload to apply)", popup of a prepared page must not be adopted — green: go test ./actions/ -run TestPopupPreparedBeforeFirstScript -count=1 passed
tdd: TestTabNewNotLeftPaused — red (resume call disabled, since the resume was already wired in cycle 1): actions_full_test.go:1421: new tab still paused: TabNew + Evaluate did not return within 5s — green: go test ./actions/ -run TestTabNewNotLeftPaused -count=1 passed
verify: T04 — pass — go build ok; go test ./types/ ./actions/ ok (22s, 330s); go test ./tests/ -run TestTabsAndViewCommands|TestSmokeAntigravity ok; grep -c ponytail types/context.go 2→1; no leftover daemons

## T05
tdd: TestMouseWheelScrollsIframeUnderPointer — red: mousewheel_test.go:136: MouseWheel = "scrolled dy=0 y=0/0 end", <nil> — green: go test ./actions/ -run TestMouseWheelScrollsIframeUnderPointer -count=1 passed
tdd: TestMouseWheelRefIframe — red: mousewheel_test.go:153: MouseWheel = "scrolled dy=0 y=0/0 end", <nil> — green: go test ./actions/ -run TestMouseWheelRefIframe -count=1 passed
verify: T05 — pass — go build + go test ./actions/ -run MouseWheel -count=1: ok github.com/agenthands/rod-cli/actions 36.667s; no leftover daemons, tree clean

## T06
tdd: TestPruneInteractiveFlattensUnnamedWrappers (updated from TestPruneInteractiveKeepsControlsAndAncestors) — red: prune_test.go:29: got: - main: - list: - listitem: - link "Docs" [ref=s1e4] want: - main: - link "Docs" [ref=s1e4] — green: go test ./types/ -run TestPruneInteractiveFlattensUnnamedWrappers -count=1 passed
tdd: TestPruneInteractiveKeepsNamedGroup — red: not observed against cycle-1 code (cycle-1 flattening already covered it); against pre-T06 snapshot.go: prune_test.go:35: got: - group "Payment": - generic: - radio "Card" [ref=s1e2] — green: go test ./types/ -run TestPruneInteractiveKeepsNamedGroup -count=1 passed
tdd: TestPruneInteractiveFlattensMultiChildWrapperKeepsSiblings (verify gap: kept aliased n.Content) — red: prune_test.go:41: got: - link "A" [ref=s1e1] - link "B" [ref=s1e2] - link "B" [ref=s1e2] want: - link "A" [ref=s1e1] - link "B" [ref=s1e2] - button "C" [ref=s1e3] — green: go test ./types/ -run TestPruneInteractiveFlattensMultiChildWrapperKeepsSiblings -count=1 passed
verify: T06 — pass — go test ./types/ -run Prune ok; go test ./actions/ -run Snapshot ok (28s); unnamed wrappers flattened, landmarks/heading/named/ref kept, ponytail comment removed

## T07
tdd: TestConsoleReportsDropped — red: console_test.go:82: first lines = ["[log] 100" "[log] 101"]; want ["[dropped 100 older entries]" "[log] 100"] — green: go test ./actions/ -run TestConsoleReportsDropped -count=1 passed
tdd: TestConsoleClearResetsDropped — red: console_test.go:108: ConsoleLogs after clear = "[dropped 100 older entries]\n[log] x"; want "[log] x" (observed with the reset line removed; the reset itself landed in cycle 1) — green: go test ./actions/ -run TestConsoleClearResetsDropped -count=1 passed
verify: T07 — pass — `go test ./types/ ./actions/ -run 'Console|Request|Network' -count=1`: ok types (no tests to run), ok actions 13.4s; TestConsoleReportsDropped/TestConsoleClearResetsDropped PASS; no leftover daemons

## T08
tdd: TestTypeWithLogic_ReturnsTypoKeyErrors — red: internal_coverage_test.go:195: k=1 iteration 0: call 1 failed but err=<nil> (typed "hello world") — green: cd /home/john/go/src/github.com/agenthands/godoll && go test ./humanize/ -run TestTypeWithLogic_ReturnsTypoKeyErrors -count=1 passed
verify: T08 — pass — `go vet ./humanize/ && go test ./humanize/ -count=1` in godoll: vet clean, ok humanize 73.5s; all 12 typeFn/backspaceFn calls in typeWithLogic now return errors (godoll 3918946)

## T09
tdd: live typing never hangs — red: panic: test timed out after 2m0s — green: cd /home/john/go/src/github.com/agenthands/godoll && go test ./examples/typing/ -count=1 -timeout 120s passed (Google PASS; DuckDuckGo SKIP: site serves "Application error: a client-side exception has occurred", input[name="q"] absent)
verify: T09 — pass — done-check: `--- SKIP: TestLiveTyping_DuckDuckGo (20.62s)` (input[name="q"] absent within 20s), `--- PASS: TestLiveTyping_Google (2.74s)`, `ok github.com/agenthands/godoll/examples/typing 23.365s`; no FAIL

## T10
verify: T10 — pass — replace count `0`; `git check-ignore go.work` → `go.work`; `go build -o rod-cli .` + `GOWORK=off go build ./...` ok; clean clone `GOWORK=off go build -buildvcs=false ./...` → CLONE_BUILD_OK (godoll v0.0.0-20261019070714-939b13e6d02f from GitHub)

note(T11): first T11 run blocked on godoll govulncheck (33 vulns: go1.26.0 stdlib, x/net v0.49.0). Fixed in godoll 1b4884d (toolchain go1.26.9, x/net v0.60.0); govulncheck clean in both repos. Full suite command fixed so go test failures fail the chain.

## Summary
- Tasks T00–T10 verified [x] by independent verifier; T11 run directly (2026-10-19).
- Full suite ×3: run1 exit=0 fails=0; run2 exit=0 fails=0; run3 exit=0 fails=0.
- godoll go vet + go test ./... -timeout 15m: exit=0, 0 FAIL.
- govulncheck: rod-cli "No vulnerabilities found."; godoll "No vulnerabilities found." (after godoll 1b4884d: toolchain go1.26.9, x/net v0.60.0).
- go test ./tests/ -run 'TestSmokeAntigravity|TestInputsAndForms|TestStorageCommands' -count=5: ok.
- Leftover daemons: none. Profiles /tmp/rod/user-data / ~/.cache/rod-cli/profiles: before 0/0, after 0/0. No rod dirs in either repo.
- Commits: rod-cli main..fix/known-issues = 68; godoll main..fix/known-issues = 5.

## Chrome temp-file leak (2026-10-19)
scan (before): per run actions +93, types +26, daemon +6 Chrome lock dirs in /tmp; Google Chrome also leaves /tmp/com.google.Chrome.* lock and url_fetcher temp files. Cause: Chrome writes them to TMPDIR and removes them only on clean exit; tests and kill paths SIGKILL it.
fix: every Chrome we launch gets a TMPDIR we own and remove with its profile.
tdd: godoll TestBrowserTempFilesStayInProfile — red: chrome lock /tmp/.org.chromium.Chromium.HS0mKJ/SingletonSocket is outside the profile /tmp/rod/user-data/368af63807e51f8c — green: go test ./browser/ passed
tdd: TestStopRemovesProfileAndTempFiles — red: chrome lock /tmp/.org.chromium.Chromium.Q53bxY/SingletonSocket is outside the profile /tmp/rod-cli-test-profile-3007012446 — green: go test ./internal/testchrome/ passed
tdd: godoll TestChromeTempDirIsShortEvenWhenTMPDIRIsLong — red: undefined: browser.ChromeTempDir — green: go test ./browser/ passed
note: godoll live TestBrowserTempFilesStayInProfile with long TMPDIR — red not observed (go-rod fixes its default profile prefix at init from /tmp; the crash it guards against is macOS, where that prefix is long). Found by testchrome: Chrome FATAL "Socket path too long" when TMPDIR is deep.
tdd: TestLaunchRemovesDeadTestProfiles — red: undefined: ownerFile — green: go test ./internal/testchrome/ passed
tdd: TestSweepRemovesStaleProfileTempDir — red: undefined: chromeTmpFile — green: go test ./types/ -run TestSweep passed
tdd: TestDaemonSIGTERMRemovesProfile + TestDaemonKillProfileSweptOnNextStart (chrome temp dir) — red: profile .../YRAnJbpgiK records no chrome temp dir — green: go test ./daemon/ -run Daemon(SIGTERM|Kill) passed
tdd: TestLockFileLeavesNoFile — red: lock file .../s.port.lock still exists after release — green: go test ./daemon/ -run TestLockFile -count=3 -race passed (TestLockFileStaysExclusiveAcrossRemoval guards the unlink race; green before and after)
tdd: TestTryLockFileSkipsAHeldLock — red: undefined: tryLockFile — green: go test ./daemon/ -run TestTryLockFile -race passed
tdd: TestDaemonKillProfileSweptOnNextStart (port file) — red: /tmp/rod-cli-killed-1792426004983348912.port survived the next start's sweep — green: go test ./daemon/ -run TestDaemonKill passed
tdd: godoll TestLaunchLeavesNothingInCallerTMPDIR — red: launch left [.org.chromium.Chromium.2qxUdC .org.chromium.Chromium.puyHIO] (go-rod Validate --dump-dom run, found with strace) — green: go test ./browser/ passed; launches 9s -> 0.2s
tdd: TestOpenDaemonLogStartsOverPastTheCap — red: undefined: daemonLogCap — green: go test ./daemon/ -run TestOpenDaemonLog passed (EnsureDaemon also closes its copy of the log fd)
scan (private TMPDIR per package, both repos): only rod-cli-daemon.log (now capped) and the user's own browsers' scoped_dir notification icons in /tmp (TMPDIR unset on pids 5538/749167; 2 per hour since 14:00)
tdd: pi integration.test.ts daemon cleanup — red: after vitest run, daemon 984150 "rod-cli --session default --headless daemon" still running — green: vitest run integration.test.ts 9 passed, no daemons left (own session pi-integration-<pid>, closed in afterAll)
tdd: TestMkdirTempRemovesDeadTestDirs — red: internal/testchrome/testchrome_test.go:90:14: undefined: MkdirTemp — green: go test ./internal/testchrome/ -count=1 passed
tdd: TestMkdirTempRemovesDeadTestDirs (in-use dir kept) — red: internal/testchrome/testchrome_test.go:92:34: too many arguments in call to MkdirTemp — green: go test ./internal/testchrome/ -count=1 passed
live: tests binary killed with kill -9 left rod-cli-tests-cache-3032369357 + a daemon; a rerun kept the dir while the daemon lived; after the daemon's SIGTERM cleanup the next run removed it; no chrome-* left

## Review-found issues (2026-10-20)
change: G3 godoll cmd/godoll build (unused rod import, go mod tidy -> go 1.26.0, test_output.log removed) — cd cmd/godoll && go build ./... && go vet ./... ; make build ; GOBIN=<tmp> make install — all succeeded, binaries print version (godoll b138191)
tdd: TestLoadConfig_EmptyPathNoFileUsesDefaultsAndWritesNothing — red: config_test.go:259: LoadConfig wrote into the working directory: [- rod-cli.yaml] — green: go test ./types/ -run TestLoadConfig_EmptyPath -count=1 passed (InitDefaultConfig and its two tests deleted; defaults deep-copied via a yaml round trip)
tdd: TestLoadConfig_EmptyPathReadsLocalFile — red not observed (reading an existing ./rod-cli.yaml already worked)
change: R2 tracked rod-cli.yaml (== current defaults) and tests/rod-cli.yaml (old defaults; e2e binary runs with cmd.Dir "..", never read it) git rm'd; untracked extensions/pi/rod-cli.yaml was an old generated default (browserTempDir ./rod/browser), deleted — go test ./types/ -count=1; go test . -run TestRunDaemonServerBad -count=1 — ok
tdd: godoll TestFailedCommandClosesBrowser — red: Should be empty, but was [/tmp/gd-118974294/chrome-3234706119] (chrome dirs left in TMPDIR) — green: go test -run 'TestFailedCommandClosesBrowser$' ./cmd/godoll (in cmd/godoll module) passed
tdd: godoll TestREPLSIGTERMClosesBrowser — red: Should be empty, but was [/tmp/gd-2301935717/chrome-3981321617] (chrome dirs left in TMPDIR) — green: go test -run 'TestREPLSIGTERMClosesBrowser$' ./cmd/godoll (in cmd/godoll module) passed
tdd: TestLaunchBrowser_FallsBackToDownloadedChromium — red: errpaths_test.go:320: expected the downloaded Chromium /tmp/TestLaunchBrowser_FallsBackToDownloadedChromium2576278333/001/downloaded-chrome to be launched, got: the machine does not have Chrome installed. Please run 'rod-cli install' to fetch it, or set the browser executable path. — green: go test ./types/ -run TestLaunchBrowser_ -count=1 passed
tdd: TestLaunchBrowser_PrefersSystemChrome — red not observed (guard: system Chrome stays first)
tdd: TestLaunchBrowser_NoChrome (now also stubs the downloaded path) — red not observed (guard: existing error text when neither exists)
tdd: godoll TestSnapshotCommand_ListsPageElements — red: "Page snapshot captured. Available elements:\n" does not contain ": button \"Submit\"" — green: go test -run 'TestSnapshotCommand_ListsPageElements$' ./commands (in cmd/godoll module) passed
change: R6a ppid watcher, StartServer ppid param and hidden --ppid flags removed (nothing passed --ppid; each client exits after its command); TestStartServerWithPpid -> TestStartServerAnswersPing; no user docs claimed it — go build ./... && go vet ./daemon/ . && go test ./daemon/ -count=1 — ok
change: R6b --scroll-duration Usage (and StealthConfig.ScrollDuration comment) now says what wheelScroll does: ms one mousewheel step spreads across its ~100px notches, unset = random 20-60 ms gaps — go build -o rod-cli . && ./rod-cli --help | grep scroll-duration — new text shown
check: go build -o rod-cli . && go vet ./... && TMPDIR=<short private dir> go test ./types/ ./daemon/ ./ -count=1 — ok (types 15.8s, daemon 95.6s, root 1.3s); no rod-cli.yaml written, no daemons left, private TMPDIR removed. Note: with a TMPDIR longer than ~40 chars TestInstallJSON fails: go-rod's Validate crashes Chromium ("Socket path too long"), so install re-downloads and the download log lands on stdout before the JSON.
tdd: TestLoadConfig_DefaultLoggerWritesNothingInWorkingDirectory — red: logger_test.go:61: the default logger wrote into the working directory: [d log/] — green: go test ./types/ -run TestLoadConfig_DefaultLogger -count=1 passed (default loggerFileName now empty: the daemon logs to stderr = the capped TMPDIR rod-cli-daemon.log; stray untracked log/server.log dirs in ., daemon/, docs/, extensions/pi/ deleted)
change: R6a stale PPID claims fixed (correcting the earlier 'no user docs claimed it'): extensions/pi/src/lifecycle.ts comment, .planning/PROJECT.md, .planning/v2.2-SECURITY.md, .planning/research/ARCHITECTURE.md, .planning/research/PITFALLS.md (moot fork/PPID unknown removed); dated phase/plan records left as history — git grep -niE 'ppid|parent process' -- . ':!docs/plans' ':!.planning/phases' — no hits
tdd: godoll TestSessionManagerErrorScenarios (Delete after the browser was closed) — red: Error: An error is expected but got nil. (Get found the session Delete had failed on; before, the test asserted NoError on "failed to close browser: ... use of closed network connection") — green: go test ./session/ -run TestSessionManagerErrorScenarios$ (cmd/godoll module) passed
tdd: godoll TestSessionManagerIsolation (rewritten: localStorage on an httptest origin, read back per session) — red not observed for the rewrite (isolation already held); the old test hung: "panic: test timed out after 4m0s / running tests: TestSessionManagerIsolation" at MustElement("div") on a text/plain data: page — green: go test ./session/ -run TestSessionManagerIsolation$ (cmd/godoll module) passed in 13s, no chrome-* left
tdd: godoll TestSessionManagerRaceDetector (rewritten: 10 goroutines create, GetOrCreate and delete their own sessions; was ~350 Chrome launches and deleted never-created names, checked with errors.Is against a fresh errors.New that never matches) — red: manager_integration_test.go:351 Error: Received unexpected error (old test, 62s) — green: go test -race ./session/ -run TestSessionManagerRaceDetector$ (cmd/godoll module) passed in 23s, no chrome-* left
tdd: godoll TestSessionManagerConcurrentAccess (readers skip sessions deleted between List and Get, close their pages; fewer browsers: 15+15 instead of 50+100) — red: reader 16: failed to get session writer-2-session-0: session 'writer-2-session-0' not found — green: go test -race ./session/ -run TestSessionManagerConcurrentAccess$ (cmd/godoll module) passed 3/3, no chrome-* left
tdd: godoll TestREPLExecuteReturnsLargeMultiLineOutput (3000-line ~100KB command output through repl.Execute) — red: repl_integration_test.go:574: Execute did not return within 10s for a 100KB output — green: go test ./repl -run TestREPLExecuteReturnsLargeMultiLineOutput -count=1 (cmd/godoll module) passed; full ./repl ok; binary check: page open file://big.html (3000 links) + snapshot via piped REPL exit=0 in 3s, 3003 lines, one ref per line; cmd/godoll/README.md "line breaks removed" claim corrected
tdd: godoll session unit tests (TestSessionManagerCreate/Get/List/GetOrCreate/Create_Duplicate never closed their managers) — red: each run alone with a private TMPDIR left 1-2 chrome-* dirs (whole package: 6) — green: TMPDIR=<private> go test ./session/ (cmd/godoll module) passed, 0 chrome-* left
tdd: TestInitLogger_FullConfig cleanup (restores log output) — red: TMPDIR=<private> go test ./types/ left TestInitLogger_FullConfig3499763962 (later logging recreated its lumberjack file after t.TempDir removal) — green: TMPDIR=<private> go test ./types/ -count=1 passed, only the shared leakless binary left
