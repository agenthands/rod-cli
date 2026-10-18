# Evidence

## Baseline

- go version go1.26.4 linux/amd64
- FAIL github.com/agenthands/rod-cli/daemon — TestStartServerWithPpid (daemon_more_test.go:156: server with ppid did not become ready). Flaky under the full parallel run: passed 3/3 in isolation (`go test ./daemon -run 'TestStartServerWithPpid$' -count=3`).
- All other Go packages ok; go vet clean.
- extensions/core: vitest 139/139 passed, tsc clean. extensions/pi: vitest 39/39 passed, tsc clean. extensions/omp: bun 30/30 passed, tsc clean. (Run separately because the suite chain stops at the Go failure.)

## T00

verify: T00 — pass — done-check `grep -c '^## Baseline'` printed 1; Baseline has go1.26.4 + named daemon TestStartServerWithPpid flake (isolated rerun ok); no code changed outside docs/plans
