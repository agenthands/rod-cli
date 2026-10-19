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
