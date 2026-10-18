# Evidence

## Baseline

- go version go1.26.4 linux/amd64
- FAIL github.com/agenthands/rod-cli/daemon — TestStartServerWithPpid (daemon_more_test.go:156: server with ppid did not become ready). Flaky under the full parallel run: passed 3/3 in isolation (`go test ./daemon -run 'TestStartServerWithPpid$' -count=3`).
- All other Go packages ok; go vet clean.
- extensions/core: vitest 139/139 passed, tsc clean. extensions/pi: vitest 39/39 passed, tsc clean. extensions/omp: bun 30/30 passed, tsc clean. (Run separately because the suite chain stops at the Go failure.)

## T00

verify: T00 — pass — done-check `grep -c '^## Baseline'` printed 1; Baseline has go1.26.4 + named daemon TestStartServerWithPpid flake (isolated rerun ok); no code changed outside docs/plans

## T01
tdd: TestEvaluate_BareValue — red: actions_full_test.go:515: Evaluate("1+1") = "Evaluate code successfully with result: 2", want "2" — green: go test ./actions/ -run TestEvaluate_BareValue -count=1 passed
tdd: TestEvalCLIBareValue — red: eval_test.go:21: eval output = "Evaluate code successfully with result: 2", want "2" — green: go build -o rod-cli . && go test ./tests/ -run TestEvalCLIBareValue -count=1 passed
verify: T01 — pass — go build ok; go test ./actions -run TestEvaluate ok; go test ./tests -run 'TestEvalCLIBareValue|TestDetection|TestProxy' ok (158s); prefix grep prints nothing

## T02
tdd: TestLocatorInFrame_IndexEqualsLen — red: snapshot_test.go:135: LocatorInFrame panicked: runtime error: index out of range [1] with length 1 — green: go test ./types/ -run TestLocatorInFrame_IndexEqualsLen -count=1 passed
tdd: TestSnapshotDoesNotLogToConsole — red: actions_full_test.go:624: ConsoleLogs after Snapshot = "[log] ", err=<nil>; want "No console logs" — green: go test ./actions/ -run TestSnapshotDoesNotLogToConsole -count=1 passed
verify: T02 — pass — go test ./types/ -run TestLocatorInFrame ok; go test ./actions/ -run TestSnapshot ok; grep -c console.log types/js/snapshotter.js = 0

## T03
tdd: TestSnapshotGenerationIsMonotonicAcrossNavigations — red: actions_full_test.go:1115: second ref = "s1e5", want prefix s2e — green: go test ./actions/ -run TestSnapshotGenerationIsMonotonicAcrossNavigations -count=1 passed
tdd: TestSnapshotInjectsMissingEngine — red: actions_full_test.go:1139: Snapshot on unprepared page: Failed to capture snapshot: capture snapshot with frames failed, frame index: 0: eval js error: ReferenceError: snapshotEngine is not defined — green: go test ./actions/ -run TestSnapshotInjectsMissingEngine -count=1 passed
verify: T03 — pass — go test ./types/ ./actions/ -run 'Snapshot|LocatorInFrame' -count=1 → ok types, ok actions; go vet clean

## T04
tdd: TestTabNewIsPrepared — red: actions_full_test.go:1160: typeof snapshotEngine = "undefined", want "object" — green: go test ./actions/ -run TestTabNewIsPrepared -count=1 passed
tdd: TestTabSelectInvalidatesOldRefs — red: (switchTo snapshot-clear removed, since cycle 1 shipped it) panic: test timed out after 1m0s — running tests: TestTabSelectInvalidatesOldRefs (1m0s) — green: go test ./actions/ -run TestTabSelectInvalidatesOldRefs -count=1 passed
tdd: TestTabListMarksCurrentWithID — red: actions_full_test.go:1199: TabList does not mark the current tab with its id: [0] Page 2 (http://127.0.0.1:44333/page2) — green: go test ./actions/ -run TestTabListMarksCurrentWithID -count=1 passed
tdd: TestTabCloseCurrentSwitches — red: actions_full_test.go:1217:28: cannot use shortID(cur) (value of type string) as int value in argument to TabClose — green: go test ./actions/ -run TestTabCloseCurrentSwitches -count=1 passed
tdd: TestTabSelectAdoptsForeignTab — red: actions_full_test.go:1242:29: cannot use shortID(raw) (value of type string) as int value in argument to TabSelect — green: go test ./actions/ -run TestTabSelectAdoptsForeignTab -count=1 passed
tdd: TestRoutesFollowTabSwitch — red: (switchTo ensureInterceptorEnabled removed, since cycle 1 shipped it) actions_full_test.go:1272: route did not follow the tab switch; fetch got: <!DOCTYPE html> — green: go test ./actions/ -run TestRoutesFollowTabSwitch -count=1 passed
verify: T04 — pass — done-check printed only ok lines (types 18.7s, actions 257.6s, daemon 70.6s, tests 5.3s); 6 tdd cycles recorded; types/config.go diff is comment-only createPage→preparePage rename

## T05
tdd: TestExecuteActionSelectPassesList — red: daemon_more_test.go:627: select failed: Failed to select option(s): cannot find element — green: go test ./daemon/ -run TestExecuteActionSelectPassesList -count=1 passed
tdd: TestSelectMultiple — red: actions_full_test.go:366: Select = "Select option(s) in element s1e14 successfully", <nil>; want "Selected \"Apple\", \"Banana\" in s1e14" — green: go test ./actions/ -run TestSelectMultiple$ -count=1 passed
