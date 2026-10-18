# Evidence

## Baseline

- go version go1.26.4 linux/amd64
- FAIL github.com/agenthands/rod-cli/daemon — TestStartServerWithPpid (daemon_more_test.go:156: server with ppid did not become ready). Flaky under the full parallel run: passed 3/3 in isolation (`go test ./daemon -run 'TestStartServerWithPpid$' -count=3`).
- FAIL github.com/agenthands/rod-cli/tests — TestStorageCommands (cli_test.go:71 "Failed to set localstorage" / "failed to start daemon"). Load flake, not branch-caused: also failed on main (ec2012a) in 1 of 2 full `go test ./... -count=1` runs on 2026-10-18; the 10 s daemon-start poll (daemon/daemon.go:130) races Chrome launches from parallel packages.
- FAIL github.com/agenthands/rod-cli/tests — TestInputsAndForms (interaction_test.go:134 "Inputs were not set correctly: ailce 2"). Pre-existing godoll bug, not branch-caused: humanize/typing.go:126 transpose typo (and :149 missed space) is never corrected; ~1.6% per 5-char `type`. Seen 1 of 2 branch runs; code path unchanged on this branch.
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
tdd: TestSelectByValueFallback — red: actions_full_test.go:380: Select = "", Failed to select option(s): cannot find element; want "Selected \"Banana\" in s1e11" — green: go test ./actions/ -run TestSelectByValueFallback$ -count=1 passed
tdd: TestSelectPartialMatchFails — red: actions_full_test.go:394: Select err = Failed to select option(s): cannot find element; want it to contain "no option matches \"Durian\"" — green: go test ./actions/ -run TestSelectPartialMatchFails$ -count=1 passed
tdd: TestUploadCommaInFilename — red: daemon_more_test.go:649: files = "2:a" (err <nil>), want 1:a,b.txt (run against the pre-T05 comma-split upload dispatch, temporarily restored) — green: go test ./daemon/ -run TestUploadCommaInFilename$ -count=1 passed
verify: T05 — pass — done-when: ok daemon (58.2s), ok actions (44.4s), ok tests (9.4s); 5/5 tdd lines present; go vet clean

## T06
tdd: TestMouseWheelReportsEndOnFakeScroller — red: actions/mousewheel_test.go:16:39: undefined: measureScroll — green: go test ./actions/ -run TestMouseWheelReportsEndOnFakeScroller -count=1 passed
tdd: TestMouseWheelPageDefault — red: mousewheel_test.go:46: MouseWheel = "scrolled dy=0 y=0/0 end", <nil> — green: go test ./actions/ -run TestMouseWheelPageDefault -count=1 passed
tdd: TestMouseWheelCLIArgs — red: cmd_test.go:850: mousewheel 5: err = dx and dy are required — green: go test . -run TestMouseWheelCLIArgs -count=1 passed
verify: T06 — pass — done-check printed only ok for actions, root and tests packages; go vet clean

## T07

verify: T07 — pass — go build+vet clean; go test ./... only failure tests/TestStorageCommands "failed to start daemon" (Baseline-listed flake), all other packages ok; core vitest 139/139, pi 39/39, omp bun 30/30, all tsc clean; no code diff since wip(T07): start

## T08
tdd: TestMouseWheelEndAbsorbsLazyGrowth — red: mousewheel_test.go:80: got "scrolled dy=800 y=800/1000", err <nil> — green: go test ./actions/ -run TestMouseWheelEndAbsorbsLazyGrowth -count=1 passed
tdd: TestMouseWheelEndCap — red not observed (cap landed with cycle 1; mutation check with the " cap" suffix removed failed: mousewheel_test.go:93: got "scrolled dy=1600 y=1600/100000", err <nil>) — green: go test ./actions/ -run TestMouseWheelEndCap -count=1 passed
tdd: TestMouseWheelRefScrollsInnerBox — red: mousewheel_test.go:107: MouseWheel = "", mousewheel by ref is not supported yet — green: go test ./actions/ -run TestMouseWheelRefScrollsInnerBox -count=1 passed
tdd: TestMouseWheelEndLive — red not observed (loop landed with cycle 1; mutation check with the loop forced to one scroll failed: mousewheel_test.go:122: MouseWheel = "scrolled dy=800 y=800/4400", <nil>) — green: go test ./actions/ -run TestMouseWheelEndLive -count=1 passed
tdd: TestAllCommandActionsHappyPath/mousewheel_--end_--max_5_--_0_-300 (CLI seam, new row) — red: cmd_test.go:639: command [mousewheel --end --max 5 -- 0 -300] failed: flag provided but not defined: -end — green: go test . -run TestAllCommandActionsHappyPath/mousewheel -count=1 passed
verify: T08 — pass — done-check printed only ok (actions 24.4s, daemon 58.7s, root); CLI TestAllCommandActionsHappyPath ok; go vet clean; cycles 2 and 4 red shown via mutation check (code landed with cycle 1)

## T09

tdd: TestMouseMoveTracksPosition — red: pointer_test.go:17: GetMousePos = {100 100}, want {120 340} — green: go test ./actions/ -run TestMouseMoveTracksPosition -count=1 passed
tdd: TestHoverDoesNotTeleport — red: pointer_test.go:36: page saw "127,90" (err <nil>), tracked pointer 100,100 — green: go test ./actions/ -run TestHoverDoesNotTeleport -count=1 passed (compares truncated coords: Chrome truncates fractional clientX/Y)
tdd: TestDragHonoursTuning — red: pointer_test.go:77: moveToElement option counts = [], want two moves with 1 (non-zero) — green: go test ./actions/ -run TestDragHonoursTuning -count=1 passed
verify: T09 — pass — done-check: build ok; go test ./actions/ (MouseMove|Hover|Drag|Mouse) ok 75.8s; go test ./tests/ (TestClickAndHover|TestRawInputs|TestHumanize) ok 46.0s; also TestErrPaths ok, go vet ./actions clean

## T10
tdd: TestPruneInteractiveKeepsControlsAndAncestors — red: types/prune_test.go:20:25: undefined: pruneInteractive — green: go test ./types/ -run TestPruneInteractive -count=1 passed
tdd: TestSnapshotInteractiveIsSmaller — red: actions/snapshot_scope_test.go:15:29: too many arguments in call to Snapshot — green: go test ./actions/ -run TestSnapshotInteractiveIsSmaller -count=1 passed (fixture is all controls, so pruning now drops non-interactive children too: link /url and button text)
tdd: TestSnapshotSubtreeCSS — red: snapshot_scope_test.go:40: subtree snapshot wrong: — green: go test ./actions/ -run TestSnapshotSubtreeCSS -count=1 passed
tdd: TestSnapshotSubtreeStaleRef — red (generation check mutated off; ref code landed with cycle 3): snapshot_scope_test.go:56: Snapshot(s1e11) after newer snapshot: err = <nil>, want stale ref — green: go test ./actions/ -run TestSnapshotSubtreeStaleRef -count=1 passed
