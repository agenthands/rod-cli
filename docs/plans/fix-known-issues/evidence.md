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
