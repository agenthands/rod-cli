# PLAN

## Resume after a reboot

Open Claude Code in `/home/john/go/src/github.com/agenthands/rod-cli` and say exactly:

`Run the workflow at /home/john/go/src/github.com/agenthands/rod-cli/docs/plans/antigravity-borrow/execute.workflow.js`

The workflow reads the task headings below. It skips `[x]`, resumes `[~]` from the uncommitted and `wip(Tnn)` work, and stops at `[!]`.

Status marks, in the heading `### Tnn [s] title`:

| Mark | Meaning |
|---|---|
| `[ ]` | todo |
| `[~]` | in progress |
| `[x]` | verified and committed |
| `[!]` | blocked; the reason is on the next line |

## Conventions (every task)

- **Repo and branch.** Work in `/home/john/go/src/github.com/agenthands/rod-cli` on branch `feat/antigravity-borrow`.
- **TDD.** Every code change, including test-only code, is written with sf-tdd: call the Skill tool with skill `sf-tdd` before the first test. Only tasks whose Seams line says "none" change no code at all; if such a task turns out to need a code change, return `blocked` instead of coding without sf-tdd. Write tests only at the task's **Seams**; the user has confirmed them. Work one red→green cycle at a time. Append one line per cycle to `docs/plans/antigravity-borrow/evidence.md` under `## Tnn`:
  `tdd: <test name> — red: <copied failure line> — green: <single-file command> passed`
- **Commits.** Stage by explicit path, never `git add -A` or `git add .`. End every message with the attribution line, using exactly this form:
  `git commit -m "<subject>" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"`
  Subjects:
  - `wip(Tnn): <test name> green` after each green cycle.
  - `wip(Tnn): start` when marking `[~]`.
  - `checkpoint(Tnn): verified` (verifier only).
- **Live tests.** Run against headless Chrome. Before any `./tests/` run: `go build -o rod-cli .`
- **Seam mocking.** Mock the existing way: save the package-level var, assign a stub, restore with `defer` (see `actions/actions_errpaths_test.go:153-160`).
- **Mark deliberate shortcuts** with a `// ponytail:` comment that names the ceiling and the upgrade path.
- **Full suite** (checkpoint tasks run it):

```bash
cd /home/john/go/src/github.com/agenthands/rod-cli && go build -o rod-cli . && go vet ./... && go test ./... -count=1 && (cd extensions/core && npx vitest run && npx tsc --noEmit) && (cd extensions/pi && npx vitest run && npx tsc --noEmit) && (cd extensions/omp && /home/john/bin/bun test && npx tsc --noEmit)
```

- **Expected test failures.** A failure listed under `## Baseline` in evidence.md existed before this plan. Any other failure belongs to the task that caused it.

---

### T00 [x] Baseline: record the suite state before any change
**Seams:** none (no code).
**Change:** Run the Full suite command once. Under `## Baseline` in evidence.md, write:
- the Go version (`go version`);
- each failing test or package by name, or the line `baseline: all green`.

**Done when:** `grep -c '^## Baseline' docs/plans/antigravity-borrow/evidence.md` prints `1`.

### T01 [~] eval prints the bare value
**Why:** It saves tokens on every call and fixes `browse_wait`, which compares stdout to `"true"` (`extensions/core/src/wait.ts:50`).
**Seams:** `actions.Evaluate` (live, `newLiveCtx`); CLI `rod-cli eval` (`./tests/`).
**Change:**
- `actions/actions.go:279` and `:328`: `return valStr, nil`.
- Update the prefix consumers:
  - `tests/detection_test.go:217-218` and `tests/proxy_integration_test.go:117-122`: drop `evalResultPrefix` and the stripping.
  - `actions/actions_full_test.go` `TestEvaluate` (:432).
  - `extensions/pi/src/__tests__/integration.test.ts:88`.
- Docs:
  - `skills/rod-cli/references/running-code.md:28-32` and `references/element-attributes.md:29-31`: replace the prefix and `sed` text with: `eval prints the bare value. With --json, a JSON object/array value prints as-is; any other value prints as {"result": "<value>"}.`
  - Remove the `sed -n 's/^Evaluate…//p'` pipes.

**TDD cycles:**
1. `TestEvaluate_BareValue` (actions_full_test.go), after `navFixture`:
   - `Evaluate(ctx,"1+1","")` == `"2"`
   - `Evaluate(ctx,"() => document.title","")` == `"Fixture"`
   - `Evaluate(ctx,"() => this.tagName", refFor(t, ctx, "Click Me"))` == `"BUTTON"`
2. `TestEvalCLIBareValue` (new `tests/eval_test.go`): `runCli("--raw","goto",ts.URL+"/page1")`, then `runCli("--raw","eval","1+1")`. Trimmed output == `"2"`.

**Done when:** `go build -o rod-cli . && go test ./actions/ -run 'TestEvaluate' -count=1 && go test ./tests/ -run 'TestEvalCLIBareValue|TestDetection|TestProxy' -count=1` ends with `ok` lines only, and `grep -rn "Evaluate code successfully" --include=*.go --include=*.ts --include=*.md . | grep -v node_modules | grep -v docs/plans` prints nothing.

### T02 [ ] Snapshot: frame-index off-by-one, no page-console logging, portable build script
**Seams:** `types.Snapshot.LocatorInFrame` (unit); `actions.Snapshot` + `actions.ConsoleLogs` (live, `newLiveCtxCfg` with console capture on).
**Change:**
- `types/snapshot.go:229`: `frameIndex > len(s.frames)` → `frameIndex >= len(s.frames)`.
- `types/js/snapshotter_raw.js:1503-1507`: delete both `console.log` calls. The non-element branch becomes `return '';`.
- `types/js/snapshotter.js`: hand-edit, replacing exactly
  `ariaSnapshot(e,t){if(e.nodeType!==Node.ELEMENT_NODE)return void console.log("ariaSnapshot only works on elements.");console.log(e,t);`
  with
  `ariaSnapshot(e,t){if(e.nodeType!==Node.ELEMENT_NODE)return"";`
- `package.json` `"dev"` script, set to exactly:
  `"dev": "npx terser types/js/snapshotter_raw.js -o types/js/snapshotter.js -c dead_code=true,arguments=true,arrows=true -m reserved=[snapshotEngine],toplevel"`
  Do not run it. The minified file is hand-edited so the diff stays small.

**TDD cycles:**
1. `TestLocatorInFrame_IndexEqualsLen` (types/snapshot_test.go): a Snapshot with 1 frame; `LocatorInFrame("f1s1e1")` returns a non-nil error and does not panic.
2. `TestSnapshotDoesNotLogToConsole` (actions_full_test.go):
   - Setup: `newLiveCtxCfg` with `Stealth.ConsoleCapture` true, then `navFixture`, then `Snapshot(ctx)`.
   - Assert: `ConsoleLogs(ctx)` == `"No console logs"`.

**Done when:** `go test ./types/ -run TestLocatorInFrame -count=1 && go test ./actions/ -run 'TestSnapshot' -count=1` prints `ok` for both, and `grep -c 'console.log' types/js/snapshotter.js` prints `0`.

### T03 [ ] Snapshot: Go-owned generation, engine re-inject, options struct
**Why:**
- The JS generation counter restarts at 1 on every document (`snapshotter_raw.js:1509`), so an old `s1e5` can silently match a different element after navigation.
- Tabs and frames that were never prepared have no `snapshotEngine`.

**Seams:** `types.Context.BuildSnapshot` (live); `actions.Snapshot` (live).
**Change:**
- **types/snapshot.go**
  - Add `type SnapshotOpts struct { Gen int; Root string; Interactive bool }`. `Root` is a JS expression; empty means `document.body`. `Interactive` is used in T10.
  - `BuildSnapshot(p *rod.Page, o SnapshotOpts)`. Thread `o` through `captureSnapshotWithFrames` (:95), including the iframe recursion (:194). Child frames always use `document.body`.
  - At :102, opts becomes `fmt.Sprintf("({ref: true, generation: %d})", o.Gen)`.
- **types/context.go**
  - Add field `snapGen int`.
  - `(*Context).BuildSnapshot(o SnapshotOpts)` does `ctx.snapGen++` and sets `o.Gen = ctx.snapGen` before calling.
- **types/js/snapshotter_raw.js:1509**
  - Change it to `const generation = options?.generation ?? ((this._lastAriaSnapshot?.generation || 0) + 1);`
  - In `snapshotter.js`, replace exactly `const i=(this._lastAriaSnapshot?.generation||0)+1;` with `const i=t?.generation??((this._lastAriaSnapshot?.generation||0)+1);`
- **Engine re-inject**
  - `types/js/js.go`: add `var InjectEngine = "() => {" + InjectedSnapShot + "\n;window.snapshotEngine = snapshotEngine}"`.
  - In the retry closure (snapshot.go:100-104), when the eval error contains `snapshotEngine is not defined`, run `pageEval(p, js.InjectEngine)` once and evaluate again.
- **Test call sites:** update `BuildSnapshot(page)` → `BuildSnapshot(page, types.SnapshotOpts{Gen: 1})` in `types/snapshot_test.go` and `types/errpaths_test.go`.

**TDD cycles:**
1. `TestSnapshotGenerationIsMonotonicAcrossNavigations` (actions_full_test.go):
   - Steps: `navFixture`, `refFor(t,ctx,"Click Me")` (prefix `s1e`), then `Navigate` to `/page2`, then `Navigate` back to `/`, then `refFor(t,ctx,"Click Me")`.
   - Assert: the second ref has prefix `s2e`.
   - Assert: `Click(ctx, firstRef)` returns an error.
2. `TestSnapshotInjectsMissingEngine` (actions_full_test.go):
   - Steps: create a raw page with `ctx.GetBrowser().Page(proto.TargetCreateTarget{URL: srv.URL+"/page2"})` (no prep), make it current with today's `ctx.SetPage` (T04 renames this), then `Snapshot(ctx)`.
   - Assert: no error, and the output contains `Page Two`.

**Done when:** `go test ./types/ ./actions/ -run 'Snapshot|LocatorInFrame' -count=1` prints `ok` for both packages.

### T04 [ ] Page lifecycle: one switch point, prepared tabs, tab IDs
**Seams:** `actions.TabNew/TabSelect/TabClose/TabList` + `actions.Snapshot/Click` (live, `newLiveCtx`); CLI `tab-*` (`tests/tabs_view_test.go`).
**Interface:** rename `(*Context).SetPage` to:

```go
// SwitchTo makes p the current tab — the only way to change it. Prepares p
// (stealth, emulation, snapshot engine, capture) the first time it is seen,
// clears the snapshot, moves mock routes to p. Selecting the current tab is a
// no-op. p == nil means no current tab (next EnsurePage opens one).
// adopted = p was opened outside rod-cli; its loaded document has no evasions
// until the next navigation.
func (ctx *Context) SwitchTo(p *rod.Page) (adopted bool)
```

**Change (types/context.go):**
1. Move `createPage` lines 677-782 into `preparePage(page *rod.Page)`. It runs with the lock held, logs errors and continues. Its first line is `ctx.prepared[page.TargetID] = true`, with the map created lazily like `cdpDomains`.
2. Delete `createPage`. The interceptor block (:793-795) moves into `switchTo`.
3. Add `switchTo` (lock held):
   - If `p` is the current TargetID, return `false`.
   - If `p` is unprepared: `adopted = true`, run `preparePage(p)`, then `p.Eval(js.InjectEngine)`.
   - Disable and nil the interceptor.
   - Set `ctx.snapshot, ctx.page = nil, p`.
   - If `p != nil && len(ctx.routes) > 0`, run `ensureInterceptorEnabled(p)`.

   Exported `SwitchTo` is the lock plus `switchTo`.
4. `initial()` (:356, :363): `page, err := ctx.browser.Page(proto.TargetCreateTarget{})` (no URL), then `ctx.switchTo(page)`.
5. `closePage()` (:462-483): `p := ctx.page; ctx.switchTo(nil); return p.Close()` (keep the existing error wrapping).
6. Add a `// ponytail:` comment: popups are prepared when first selected. The upgrade path is `Target.setAutoAttach{waitForDebuggerOnStart:true}`.

**Change (actions/actions.go:635-707):**
- **Helpers:**
  - `shortID(p)` returns the first 8 characters of the TargetID.
  - `tabDesc(p)` returns `"<ID> <Title> (<url>)"`. If `Info()` fails, title and url are empty. This fixes the nil deref at :646-647.
  - `resolveTab(pages, sel)`:
    - All digits and fewer than 8 characters → index, error `Tab index out of range: %d`.
    - Otherwise → case-insensitive TargetID prefix that must be unique, errors `No tab matches %q` / `Tab id %q is ambiguous`.
- **TabList:** one line per tab, `* [i] <desc>` for the current tab and `  [i] <desc>` for the others.
- **TabNew:**
  - Run `browserPage(browser, proto.TargetCreateTarget{})`, then `ctx.SwitchTo(page)`.
  - If `url` is not empty and not `about:blank`, call the existing `Navigate(ctx, url)`.
  - Output: `New tab created: <desc>`.
- **TabSelect(ctx, sel string):**
  - Run resolve → `pageActivate` → `SwitchTo`.
  - Output: `Selected tab <i> <desc>`. When the tab was adopted, append `; adopted, stealth applies from next navigation (reload to apply)`.
- **TabClose(ctx, sel string):**
  - When the target is the current tab, `SwitchTo(rest[max(i-1,0)])` or `SwitchTo(nil)` *before* `pageClose`.
  - Output, one of:
    - `Closed tab <i> <ID>; current: [j] <desc>`
    - `Closed tab <i> <ID>; no tabs left (goto opens one)`
    - `Closed tab <i> <ID>`
- **Wiring:**
  - `daemon/daemon.go:339-346` passes `req.Args["index"]` as a string; drop the `Sscanf`.
  - `cmd.go:741,752`: usage becomes `Close a tab by index or 8-char id` / `Select a tab by index or 8-char id`. Keep the error text `tab index is required`.
- **Call sites:**
  - Update `SetPage` call sites (`types/context_more_test.go:158,205`, `types/errpaths_test.go:370,386`, and the T03 test).
  - Change int tab indices to strings in `actions_full_test.go:713-737` and `actions_errpaths_test.go:474-493`.
  - `tests/tabs_view_test.go:38-83` must still pass. Its `Selected tab`/`Closed tab` substrings are unchanged.
  - Update `skills/rod-cli/references/tabs-and-windows.md` for the IDs and the `*` marker.

**TDD cycles:**
1. `TestTabNewIsPrepared`: `TabNew(ctx, srv.URL+"/page2")`, then `Evaluate(ctx,"typeof snapshotEngine","")` == `"object"`.
2. `TestTabSelectInvalidatesOldRefs`: take a ref on tab A, run `TabNew`, then `Click(ctx, oldRef)` returns an error. A new `refFor` on the new tab works.
3. `TestTabListMarksCurrentWithID`: `TabList` matches `(?m)^\* \[\d+\] [0-9A-F]{8} Page 2`.
4. `TestTabCloseCurrentSwitches`: `TabClose(ctx, <current short ID>)` contains `; current: [`, and then `Snapshot(ctx)` succeeds.
5. `TestTabSelectAdoptsForeignTab`:
   - Setup: a raw `GetBrowser().Page(...URL...)`.
   - `TabSelect(ctx, shortID)` contains `adopted`, and `Snapshot(ctx)` succeeds.
6. `TestRoutesFollowTabSwitch`: add a route, run `TabNew` to a fixture URL, and assert that a fetch on the new tab gets the mocked body. Reuse the `TestRoutes` (:865) pattern.

**Done when:** `go build -o rod-cli . && go test ./types/ ./actions/ ./daemon/ -count=1 && go test ./tests/ -run TestTabsAndViewCommands -count=1` prints only `ok` lines.

### T05 [ ] select and upload: real lists, text→value→substring matching
**Seams:** `daemon.executeAction` (in-process, the `TestExecuteActionDispatch` style at daemon_more_test.go:354); `actions.Select` (live).
**Change:**
- **cmd.go** `upload` (:544) and `select` (:556): send `json.Marshal(c.Args().Slice()[1:])` in `Args["files"]` / `Args["values"]`.
- **daemon.go:294-301**
  - `json.Unmarshal` into `[]string`. If parsing fails, use the raw string as one value; this keeps an older client working.
  - Delete the confused TODO comment.
- **actions.Select(ctx, ref, values)**, for each value in order:
  1. Exact option text, using `elementSelect([v], true, rod.SelectorTypeText)`.
  2. If there is no match: value, using `SelectorTypeCSSSector` with `[value=<strconv.Quote(v)>]`.
  3. If there is still no match: case-insensitive substring of the option text. Find the option value with one `elementEval`, then use the CSS path.

  Every value must match. Otherwise return the error `no option matches %q`.
  Read back the selected labels. Output: `Selected "Apple", "Banana" in <ref>`.
- **Fixture:** add `<select id="multi" multiple aria-label="Multi Box"><option value="a">Apple</option><option value="b">Banana</option><option value="c">Cherry</option></select>` to `fixtureHTML`.

**TDD cycles:**
1. `TestExecuteActionSelectPassesList` (daemon): run select with `values` = `["Apple","Banana"]`. A stub `actions` seam (or the live fixture) receives exactly 2 values.
2. `TestSelectMultiple` (live): `Select(ctx, ref("Multi Box"), []string{"Apple","Banana"})` == `Selected "Apple", "Banana" in <ref>`.
3. `TestSelectByValueFallback`: selecting `"b"` on `Select Box` → `Selected "Banana" in <ref>`.
4. `TestSelectPartialMatchFails`: `[]string{"Apple","Durian"}` → error containing `no option matches "Durian"`.
5. `TestUploadCommaInFilename`: upload `a,b.txt` (a temp file) through `executeAction`. The element gets 1 file named `a,b.txt`.

**Done when:** `go build -o rod-cli . && go test ./daemon/ ./actions/ -run 'Select|Upload|ExecuteAction' -count=1 && go test ./tests/ -run TestInputsAndForms -count=1` prints only `ok`.

### T06 [ ] mousewheel: real wheel input, scroll feedback, optional dx/dy
**Seams:** `actions.MouseWheel` (live on a `/scroll` fixture; unit with a fake scroller via the seams below); CLI `mousewheel` (`tests/tabs_view_test.go`).
**Interface:**

```go
// MouseWheel scrolls with real wheel input and reports what actually moved.
// dx=dy=0 → one viewport down. ref=="" → page; ref → the ref's nearest
// scrollable ancestor (T08). maxSteps 0 = one scroll; n>0 = until end (T08).
func MouseWheel(ctx *types.Context, dx, dy float64, ref string, maxSteps int) (string, error)
```

Output is one line: `scrolled dy=<moved> y=<pos>/<max>`.
- Append ` end` when the requested direction moved 0 px or finished at its limit.
- Prepend `dx=<moved> x=<pos>/<max> ` only when dx was requested or the horizontal position changed.

**Change (actions/actions.go:591-622):**
- **New seams** in the :52 block:
  - `mouseScroll = func(p *rod.Page, dx, dy float64) error { return p.Mouse.Scroll(dx, dy, 1) }`
  - `moveToPoint = humanize.MoveToPoint`
  - `measureScroll = func(el *rod.Element) ([]scrollBox, error)`
  - `wheelSettle = 400 * time.Millisecond`
  - `wheelEndBudget = 10 * time.Second`
- **`type scrollBox struct{ X, Y, MaxX, MaxY, VW, VH float64 }`.** `measureScroll` runs one `elementEval` of a `function(){…}`. It walks `this` up through `parentElement || getRootNode().host`. It collects ancestors whose `overflowX+overflowY` matches `/auto|scroll|overlay/` and whose scroll size is larger than their client size. It always appends `document.scrollingElement`.
- **Page path:**
  1. `proto.PageGetLayoutMetrics{}` → pick the viewport centre ±10% → `moveToPoint(page, pt, mouseOpts(ctx.HumanizeTuning())...)`.
  2. Get the anchor with `page.ElementByJS(rod.Eval("(x,y)=>document.elementFromPoint(x,y)||document.scrollingElement", x, y))`. The `||` fallback is required, because `ElementByJS` polls forever on null.
- **Scroll and measure:**
  1. Measure.
  2. Scroll in chunks: `n = ceil(max(|dx|,|dy|)/100)` calls of `mouseScroll(dx/n, dy/n)`. Each gap is 20–60 ms, or `ScrollDuration/n` when that is set.
  3. Re-measure every 50 ms until two reads are equal, for at most 1 s.
  4. Report the innermost box that moved. If nothing moved, report the innermost box.
  5. The default step is that box's `VH`.
- **Remove** the `humanizeScrollBy` seam (:77) and `scrollOpts` (:148-157). In `cmd.go:362`, `--scroll-physics` usage becomes `ignored: wheel scrolling uses real wheel notches`. Add the ponytail comment.
- **CLI (cmd.go:669-680):** `mousewheel [dx dy] [ref]` with no flags yet. Positional count:
  - 0 → default.
  - 1 → ref. If that one argument is numeric, error `dx and dy are required together`.
  - 2 → dx dy.
  - 3 → dx dy ref.

  Keep the `cmd_test.go:246` missing-arg test passing. `daemon.go:325-329` passes `dx, dy, ref` and `maxSteps=0`.
- **Fixture route** `/scroll` in `newFixtureServer`:
  `<body style="margin:0"><div id="box" style="height:200px;overflow:auto"><div style="height:2000px">in</div></div><div style="height:5000px">tall</div></body>`
- **Callers:** update `actions_errpaths_test.go:404-420` to swap `mouseScroll`, update `actions_full_test.go:650-674` to the new signature, and change `tests/tabs_view_test.go:26` to expect `scrolled dy=`.

**TDD cycles:**
1. `TestMouseWheelReportsEndOnFakeScroller` (unit):
   - Fake `measureScroll`/`mouseScroll`: a clamped `pos` with `max=1000` and `VH=800`.
   - Call 1: `MouseWheel(ctx,0,0,"",0)` == `scrolled dy=800 y=800/1000`.
   - Call 2: == `scrolled dy=200 y=1000/1000 end`.
2. `TestMouseWheelPageDefault` (live, `/scroll`): output matches `^scrolled dy=([1-9]\d*) y=\d+/\d+$`.
3. `TestMouseWheelCLIArgs` (cmd_test.go): `mousewheel 5` errors with `dx and dy are required together`.

**Done when:** `go build -o rod-cli . && go test ./actions/ ./ -run 'MouseWheel|Mouse|MissingArg' -count=1 && go test ./tests/ -run 'TestTabsAndViewCommands|TestRawInputs' -count=1` prints only `ok`.

### T07 [ ] Checkpoint 1: P0 complete, full suite green
**Seams:** none (no code).
**Change:** Run the Full suite command. Fix nothing here. If a test outside the Baseline fails, return `blocked` and name the task whose files it touches.
**Done when:** the Full suite command exits 0, apart from Baseline-listed failures.

### T08 [ ] mousewheel: scroll an inner pane by ref, `--end` with a cap
**Seams:** `actions.MouseWheel` (live `/scroll` + fake scroller); CLI `mousewheel --end`.
**Change:**
- **ref path:** `getElementByRef(ctx, ref)` → `humanize.MoveTo(page, el, mouseOpts(...)...)`, which scrolls it into view and lands at a random point inside it. The anchor is that element. Scroll and measure as in T06.
- **`maxSteps > 0`:**
  - Repeat until the requested direction reports `end`, or until `maxSteps` iterations or `wheelEndBudget` pass.
  - At the limit, wait `wheelSettle` and re-measure once (for lazy loads). Only then print ` end`.
  - When the cap is hit, print ` cap`.
  - `dy` is the total moved.
- **CLI:** `&cli.BoolFlag{Name:"end"}` and `&cli.IntFlag{Name:"max", Value:10}`. Send `end` and `max`. `daemon.go`: `maxSteps = 0` unless `end=="true"`, then `Sscanf(max)`. Negative values need `--`, for example `mousewheel --end -- 0 -300`.
- **Ponytail comments:**
  - Iframe-under-pointer: the measure walks the parent document. The upgrade is to recurse into a same-origin `contentDocument`.
  - Hover side effects from moving the pointer.

**TDD cycles:**
1. `TestMouseWheelEndAbsorbsLazyGrowth` (unit, fake scroller):
   - `max` grows from 1000 to 1500 on the first end. `wheelSettle=0`.
   - `MouseWheel(ctx,0,0,"",10)` == `scrolled dy=1500 y=1500/1500 end`.
2. `TestMouseWheelEndCap` (unit): `max=100000`, `maxSteps=2` → output ends with ` cap`.
3. `TestMouseWheelRefScrollsInnerBox` (live `/scroll`): `MouseWheel(ctx,0,300,"#box",0)` matches `^scrolled dy=([1-9]\d*) y=\d+/1800$`.
4. `TestMouseWheelEndLive` (live): `MouseWheel(ctx,0,0,"",50)` ends with ` end`, and `y` equals `max`.

**Done when:** `go build -o rod-cli . && go test ./actions/ ./daemon/ ./ -run 'MouseWheel|ExecuteAction' -count=1` prints only `ok`.

### T09 [ ] Humanized pointer: mousemove, hover and drag use tuned curves, no settle on raw steps
**Why:**
- `MouseMove` uses the raw `Mouse.MoveTo`. It ignores `--mouse-*` and never updates godoll's tracked position, so the next humanized click curves from a stale point.
- `Hover` teleports to the centre after its curve, because godoll `Hover` calls `el.Hover()`.
- `Drag` ignores the tuning.
- `WaitDOMStable` on mousemove and mousedown costs ≥1 s per step.

**Seams:** `actions.MouseMove/Hover/Drag` (live, plus `humanize.GetMousePos`, godoll's public API).
**Change:**
- `MouseMove` (:749-759): `moveToPoint(page, humanize.Point{X:x,Y:y}, mouseOpts(ctx.HumanizeTuning())...)`. Delete the `WaitDOMStable` here and in `MouseDown` (:776). Keep it in `MouseUp`.
- **Hover:**
  - (:519-533) replace `humanizeHover(element)` with a new seam `moveToElement = humanize.MoveTo`, called with `moveToElement(page, element, mouseOpts(...)...)`.
  - Delete the `humanizeHover` seam (:76).
- **Drag:**
  - (:1002-1020) replace `humanizeDragAndDrop` with: `moveToElement(el1, opts)` → `pageMouseDown(left)` → `moveToElement(el2, opts)` → `pageMouseUp(left)`, then `WaitDOMStable`.
  - Release the button with `defer` on error.
  - Delete the `humanizeDragAndDrop` seam (:78) and update `actions_errpaths_test.go:386`.
- **Fixture:** add `<script>addEventListener('mousemove',e=>{window.__mx=e.clientX;window.__my=e.clientY})</script>` to `fixtureHTML`.

**TDD cycles:**
1. `TestMouseMoveTracksPosition`: `MouseMove(ctx,120,340)`, then `humanize.GetMousePos(page)` == `{120,340}`, and `Evaluate(ctx,"window.__mx+','+window.__my","")` == `"120,340"`.
2. `TestHoverDoesNotTeleport`: `Hover(ctx, ref("Click Me"))`. The page's last `__mx,__my` equals `GetMousePos(page)`, rounded.
3. `TestDragHonoursTuning`: with a stub `moveToElement` recording `len(opts)`, under `newLiveCtxCfg` setting `MouseSteps=7`, both moves receive the same non-zero option count that `mouseOpts` returns for that config.

**Done when:** `go build -o rod-cli . && go test ./actions/ -run 'MouseMove|Hover|Drag|Mouse' -count=1 && go test ./tests/ -run 'TestClickAndHover|TestRawInputs|TestHumanize' -count=1` prints only `ok`.

### T10 [ ] Cheaper reads: `snapshot -i` (interactive only) and `snapshot <ref|css>` (subtree)
**Seams:** `types.pruneInteractive` (pure, unit, on `*yaml.Node`); `actions.Snapshot` (live).
**Change:**
- **types/snapshot.go:** `pruneInteractive(n *yaml.Node) *yaml.Node`. It runs after `walk`, before marshal, when `o.Interactive` is set. It keeps:
  - a scalar or mapping key whose text starts with a role in `button link textbox searchbox checkbox radio combobox listbox option menuitem tab switch slider spinbutton heading`;
  - every ancestor of a kept node.

  It drops everything else. Add `// ponytail: keeps unnamed wrapper ancestors; flatten them if output is still too large`.
- **Root expression** (built in actions from the positional arg):
  - **CSS:** `document.querySelector(<json-quoted css>)||(()=>{throw new Error("no element matches <css>")})()`
  - **Ref `s<G>e<N>`:**
    `(()=>{const s=snapshotEngine._lastAriaSnapshot;if(!s||s.generation!==<G>)throw new Error("stale ref s<G>e<N>: run snapshot");const e=s.elements.get(<N>);if(!e)throw new Error("no element for ref s<G>e<N>");return e})()`
  - **Frame refs** (`f…`): error `snapshot <ref> does not support iframe refs yet`.
- The header gains the line `- Scope: <arg>` when a root is given.
- **CLI (cmd.go:575-581):** `snapshot [-i|--interactive] [ref|css]`, passed as `Args["interactive"]` and `Args["root"]`, then through daemon.go to `actions.Snapshot(ctx, root string, interactive bool)`.
- **Extensions:** stop ignoring `selector` in `buildSnapshotArgs` (`extensions/core/src/argv.ts:16-20`). Put it after `--`, and add `interactive` → `-i` before `--`. Update the Pi/OMP `snapshot.ts` schemas (`interactive?: boolean`) and delete `element-scoped snapshots are not yet supported`.

**TDD cycles:**
1. `TestPruneInteractiveKeepsControlsAndAncestors` (unit, literal YAML):
   - Input: `- main:\n  - paragraph: hello\n  - list:\n    - listitem:\n      - link "Docs" [ref=s1e4]\n- contentinfo: footer text`
   - Output: `- main:\n  - list:\n    - listitem:\n      - link "Docs" [ref=s1e4]\n`
2. `TestSnapshotInteractiveIsSmaller` (live): `len(Snapshot(ctx,"",true)) < len(Snapshot(ctx,"",false))`, and it still contains `Click Me`.
3. `TestSnapshotSubtreeCSS` (live): `Snapshot(ctx,"#sel",false)` contains `Apple` and `- Scope: #sel`, and does not contain `Click Me`.
4. `TestSnapshotSubtreeStaleRef`: take a ref, take a new snapshot, then `Snapshot(ctx, oldRef, false)` errors with `stale ref`.
5. `buildSnapshotArgs` vitest (core `argv.test.ts`): `{selector:"s1e3", interactive:true}` → `["snapshot","-i","--","s1e3"]`.

**Done when:** `go test ./types/ ./actions/ -run 'Prune|Snapshot' -count=1 && (cd extensions/core && npx vitest run && npx tsc --noEmit)` all pass.

### T11 [ ] `--max-output N`: a hard cap on every command's output, used by the extensions
**Why:** Pi requires tools to truncate (`node_modules/@earendil-works/pi-coding-agent/docs/extensions.md:1993`). Today the snapshot description claims 50KB/2000 lines, but nothing truncates.
**Seams:** `capOutput` (pure, cmd.go; property test); `runClientCommand` output (`cmd_test.go` `TestRunClientCommandOutputBranches` style); core `execRodCli` argv.
**Change:**
- **cmd.go global flag:** `&cli.IntFlag{Name:"max-output", Usage:"cap output at N bytes (0 = unlimited)", Value:0}`.
- **`capOutput(msg string, max int) string`:**
  - It returns `msg` when `max<=0`, `len(msg)<=max`, or `isJSONValue(msg)`.
  - Otherwise it cuts at the last `\n` at or before `max` bytes. If there is none, it cuts at `max` and backs up to a UTF-8 rune boundary.
  - Then it appends `\n[truncated: <kept> of <total> bytes; narrow with snapshot -i / snapshot <ref>, or --max-output 0]`.
  - Apply it in `runClientCommand` (:212-227) before printing.
- **Extensions:**
  - `extensions/core/src/exec.ts:39` prepends `["--raw","--max-output","51200", ...args]`. Update `core/__tests__/exec.test.ts:14` and the exact-argv tests that include `--raw` (`pi/src/__tests__/adversarial.test.ts:280-286`).
  - Pi and OMP `snapshot.ts`: replace `Output is truncated to 50KB/2000 lines.` with `Output is capped at 50KB; an oversized result ends with a [truncated: …] line saying how to narrow it.`
- **Dependency:** add `pgregory.net/rapid` with `go get pgregory.net/rapid@latest`.

**TDD cycles:**
1. `TestCapOutputProperties` (rapid, cmd_test.go). For any string `s` and `max` in [1,4096], `out := capOutput(s,max)`:
   - if `len(s)<=max`, then `out==s`;
   - otherwise `out` has the `[truncated:` suffix, the part before it is a prefix of `s` of at most `max` bytes, and that part is valid UTF-8.
2. `TestCapOutputSkipsJSON`: `capOutput("{\"a\":1}", 3)` == `{"a":1}`.
3. `TestExecPrependsMaxOutput` (core vitest): the argv starts with `["--raw","--max-output","51200"]`.

**Done when:** `go test ./ -run 'CapOutput|RunClientCommand' -count=1 && (cd extensions/core && npx vitest run) && (cd extensions/pi && npx vitest run) && (cd extensions/omp && /home/john/bin/bun test)` all pass, and `grep -rn "50KB/2000" extensions/*/src` prints nothing.

### T12 [ ] Screenshots and PDFs: honour --selector, safe --name, absolute --out, --full-page
**Seams:** `actions.Screenshot/Pdf` (live); CLI `screenshot` (`./tests/`).
**Change:**
- **CLI** `screenshot [--name n] [--selector ref|css] [--out path] [--full-page]` and `pdf [--name n] [--out path]`.
  - The client resolves `--out` with `filepath.Abs` (the daemon's cwd differs) and sends `Args["out"]`.
- **actions**
  - `saveArtifact(out, dir, name, ext string, data []byte) (absPath string, err error)`.
    - `out` set → write there, after creating the parent directory.
    - Otherwise → `dir/filepath.Base(name)+ext`. Reject `.`, `..` and an empty base with `invalid name %q`.
    - Use it in both Screenshot and Pdf.
  - `Screenshot(ctx, name, selector, out string, fullPage bool)`. This drops the unused width/height and updates the `daemon.go:311` call.
    - A selector resolves through `getElementByRef` and is captured with `el.Screenshot(proto.PageCaptureScreenshotFormatPng, 0)`.
    - Otherwise use `pageScreenshot(page, fullPage, req)`.
  - Output: `saved <abs path> <W>x<H>`, read from the PNG header (`image.DecodeConfig`). Pdf output: `saved <abs path>`.
- **Tests that change:** update the `Save to` assertions in `actions_full_test.go:546,565`, `tests/misc_test.go:133` and `tests/navigation_test.go:237` to `saved `.

**TDD cycles:**
1. `TestScreenshotSelectorIsCropped`: `Screenshot(ctx,"x","#sel","",false)` → W < viewport width.
2. `TestScreenshotNameCannotEscape`: name `../evil` → the file lands at `tmp/screenshots/evil.png`, and nothing appears outside `tmp/screenshots/`.
3. `TestScreenshotFullPage` (fixture `/scroll`): `fullPage=true` gives H > viewport height.
4. `TestScreenshotOutAbsolute` (`./tests/`): `--out <t.TempDir()>/a.png` → the output starts with `saved /` and the file exists.

**Done when:** `go build -o rod-cli . && go test ./actions/ -run 'Screenshot|Pdf' -count=1 && go test ./tests/ -run 'TestScreenshot|TestMisc|TestNavigation' -count=1` prints only `ok`.

### T13 [ ] Console and requests: record uncaught exceptions, bounded buffers, `--clear`
**Seams:** `actions.ConsoleLogs/NetworkRequests` (live with capture on); `types.Context` getters.
**Change:**
- **types/context.go** console-capture branch (:757-770):
  - Also subscribe to `proto.RuntimeExceptionThrown` and append `[exception] <ExceptionDetails.Exception.Description or Text>`.
  - Cap both buffers at 2000 entries, dropping the oldest, with a ponytail comment.
  - `GetConsoleLogs`/`GetRequests` (:833-843) return copies.
  - Add `ClearConsoleLogs()` and `ClearRequests()`.
- **CLI:** `console [--clear]` and `requests [--clear]`. `--clear` prints the entries, then clears them.
- **Docs:** add to the skill (T16) that filtering is `rod-cli console | grep '^\[error\]' | tail -20`.

**TDD cycles:**
1. `TestConsoleRecordsUncaughtException`: navigate with capture on, run `Evaluate(ctx,"setTimeout(()=>{throw new Error('boom')},0)","")`, wait 300 ms, then `ConsoleLogs` contains `[exception]` and `boom`.
2. `TestConsoleClear`: `ConsoleLogs(ctx, true)` returns the entries; the next call == `No console logs`.
3. `TestConsoleBufferCapped`: 2100 `console.log` calls → exactly 2000 entries, and the first one is `[log] 100`.

**Done when:** `go test ./types/ ./actions/ -run 'Console|Request|Network' -count=1` prints `ok` for both.

### T14 [ ] Checkpoint 2: Go work complete, full suite green
**Seams:** none (no code).
**Change:** Run the Full suite command. If a test outside the Baseline fails, return `blocked` and name the task whose files it touches.
**Done when:** the Full suite command exits 0, apart from Baseline-listed failures.

### T15 [ ] Extensions: scroll feedback, press/select/hover tools, screenshot returns the image
**Seams:**
- core `build*Args` (vitest, `extensions/core/src/__tests__/argv.test.ts`);
- Pi registration and argv (`pi/src/__tests__/adversarial.test.ts`);
- OMP registration and argv (`omp/src/__tests__/registration.test.ts`, `argv.test.ts`);
- the Pi/OMP screenshot tool `execute` with a fake `execFn` and a temp PNG.

**Change:**
- **`buildScrollArgs`** (`argv.ts:59-75`):
  - `direction` is one of `up|down|left|right`.
  - With no `distance`, the result is `["mousewheel"]` plus the selector if given.
  - With a distance, map it to dx/dy (down `0 d`, up `0 -d`, right `d 0`, left `-d 0`) after `--`.
  - A `selector` is passed as the trailing ref.
  - Delete the throw.
- **Pi/OMP `scroll.ts`:**
  - Use the 4-value enum and drop the `.default(300)`.
  - Description: `Scroll with real wheel input. Returns what moved: "scrolled dy=800 y=1600/4600"; " end" means nothing more in that direction. Pass selector (ref or CSS) to scroll an inner pane.`
  - Delete the browse_eval scrolling guidance.
- **New builders:**
  - `buildPressArgs({key})` → `["press","--",key]`
  - `buildSelectArgs({selector, values})` → `["select","--",selector,...values]`
  - `buildHoverArgs({selector})` → `["hover","--",selector]`

  Add new tools `browse_press`, `browse_select` and `browse_hover` in `pi/src/tools` (TypeBox) and `omp/src/tools` (zod), copying `click.ts`, and register them. The tool count goes 13 → 16: update `pi/adversarial.test.ts:182-202,351-357` and `omp/registration.test.ts`.
- **browse_screenshot:**
  - Pass `--out <os.tmpdir()>/rod-cli-<crypto.randomUUID()>.png` and an optional `fullPage` → `--full-page`.
  - Read the file and return `[{type:"text",text:stdout},{type:"image",data:<base64>,mimeType:"image/png"}]`, then delete the file.
  - Confirm the image content shape first with `grep -n '"image"' node_modules/@earendil-works/pi-coding-agent/docs/extensions.md`. For OMP, grep the same word in its host types under `node_modules`.
- **Strict schemas (the useful part of AntiGravity's JSON-schema style):** today no tool schema rejects unknown keys, so a misspelled param such as `selectr` is silently dropped and the tool runs with defaults.
  - Pi: pass `{ additionalProperties: false }` as the second argument of every `Type.Object(...)`.
  - OMP: call `.strict()` on every `z.object(...)`.
  - Every field keeps a `description`, and fields without a default are required.
- **Docs:** update the `extensions/pi/README.md` and `extensions/omp/README.md` tool tables (:37-45) for scroll, snapshot, screenshot and the 3 new tools.

**TDD cycles:**
1. `buildScrollArgs` cases:
   - `{}` → `["mousewheel"]`
   - `{direction:"left",distance:200}` → `["mousewheel","--","-200","0"]`
   - `{selector:"s1e4"}` → `["mousewheel","--","s1e4"]`
2. press/select/hover builders: exact argv, including the session prefix.
3. Pi and OMP: 16 tools registered, with the names `browse_press`, `browse_select` and `browse_hover` present.
4. Screenshot execute: a fake `execFn` writes a 1×1 PNG to the `--out` path. The result has an `image` item with `mimeType` `image/png`, and the temp file is gone.
5. `TestAllToolSchemasStrict`:
   - Pi (`adversarial.test.ts`): for each of the 16 registered tools, `parameters.additionalProperties === false`, and every property has a non-empty `description`.
   - OMP (`registration.test.ts`): for each tool, `parameters.safeParse({...validArgs, bogus: 1}).success === false`.

**Done when:** `(cd extensions/core && npx vitest run && npx tsc --noEmit) && (cd extensions/pi && npx vitest run && npx tsc --noEmit) && (cd extensions/omp && /home/john/bin/bun test && npx tsc --noEmit)` all pass.

### T16 [ ] Skill and references: document what shipped, fix false claims, add read recipes
**Seams:** none (docs). The check is the greps below plus `--help` parity.
**Change (`skills/rod-cli/SKILL.md`, references):**
- **Line 26:**
  - `click`/`dblclick`/`fill`/`type`/`hover`/`drag`/`mousewheel` dispatch real, trusted, humanized input.
  - `select`/`check`/`uncheck` set state through JS and fire synthetic events (`isTrusted=false`), but they are still preferred over hand-written `eval`.
- **One rule line under Interaction:** `Use refs first; use pixel input (mousemove/mousedown/mouseup) only when no ref or selector works (canvas, sliders without ARIA, hover-only targets). Coordinates are viewport CSS px.`
- **Browser Lifecycle & Tabs:**
  - `tab-list` shows `*` and 8-char IDs; `tab-select`/`tab-close` take an index or an ID.
  - Rule: `Drive only tabs you created (IDs printed by tab-new); under --cdp-endpoint never navigate the user's existing tabs.`
- **Update these lines:**
  - `mousewheel [--end [--max n]] [dx dy] [ref]` and its output/` end`/` cap` meaning;
  - `snapshot [-i] [ref|css]`;
  - `--max-output N` under Global Options;
  - `screenshot [--name] [--selector] [--out] [--full-page]` → `saved <path> WxH`;
  - `console [--clear]` (+ `[exception]` entries, grep/tail for filtering);
  - `select` matching order (text → value → substring, all values required);
  - `eval` prints the bare value.
- **`references/running-code.md`:** add a "Read recipes" section with exactly these commands:
  - `rod-cli eval "(document.querySelector('main,article,[role=main]')||document.body).innerText"`
  - `rod-cli eval "() => this.outerHTML" <ref>`
  - `rod-cli eval "() => this.scrollIntoView({block:'center'})" <ref>`
  - `rod-cli eval "fetch('/api/items',{credentials:'include'}).then(r=>r.text())"` (runs on Chrome's own network stack; never fetch from Go)
- `references/tabs-and-windows.md`: IDs, `*`, ownership rule (if T04 did not already).

**Done when all of these hold:**
- `grep -c "isTrusted=false" skills/rod-cli/SKILL.md` prints `1`.
- `grep -c "\-\-max-output" skills/rod-cli/SKILL.md` ≥ 1.
- `grep -c "mousewheel \[--end" skills/rod-cli/SKILL.md` prints `1`.
- `grep -c "snapshot \[-i\]" skills/rod-cli/SKILL.md` prints `1`.
- `grep -rc "Read recipes" skills/rod-cli/references/running-code.md` prints `1`.
- `go build -o rod-cli . && ./rod-cli mousewheel --help && ./rod-cli snapshot --help && ./rod-cli screenshot --help` shows `--end`, `--interactive` and `--full-page`.

### T17 [ ] Hide snapshotEngine from page scripts (isolated world): spike, then implement or record no-go
**Why:** `window.snapshotEngine` is a global that any page script can detect.
**Deliverable:** one of these two counts as done:
- (a) the implementation passes the checks below, or
- (b) evidence.md has a `## T17 no-go` section that names the CDP call that would break the CDP-footprint rule or the test that failed, with the command output. Then the heading is ticked `[x]` with the title suffix `(no-go recorded)`.

**Seams:** `actions.Snapshot`/`Click` by ref (live); a main-world `Evaluate`.
**Approach:**
- Inject with `proto.PageAddScriptToEvaluateOnNewDocument{Source: js.InjectedSnapShot, WorldName: "rodcli"}` in place of `EvalOnNewDocument` in `preparePage`.
- Evaluate snapshot calls in that world, through a context ID from `proto.PageCreateIsolatedWorld{FrameID, WorldName:"rodcli"}`, cached per frame and document.
- The no-go rule applies if this needs `Runtime.enable`, which the CDP-footprint rule in `types/cdp_footprint_test.go` forbids by default.

**TDD cycles:**
1. `TestSnapshotEngineNotVisibleToPage`: after `Snapshot(ctx)`, `Evaluate(ctx,"typeof window.snapshotEngine","")` == `"undefined"`.
2. `TestRefClickStillWorksFromIsolatedWorld`: `Click(ctx, refFor(t,ctx,"Click Me"))`, then the heading text == `clicked`.

**Done when:** either `go test ./types/ ./actions/ -count=1` passes with both tests included, or `grep -c '^## T17 no-go' docs/plans/antigravity-borrow/evidence.md` prints `1` and `go test ./types/ ./actions/ -count=1` passes.

### T18 [ ] Final verification: full suite, vulnerability scan, live smoke
**Seams:** CLI end-to-end (`./tests/`, `runCli` + `SetupTestServer`). Write `TestSmokeAntigravity` with sf-tdd: run it red first (assert one output format from a not-yet-asserted step), then green.
**Change:**
1. Run the Full suite command.
2. Run `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`.
3. Run the live smoke against the test server, in a new `tests/smoke_antigravity_test.go`. It drives one session: `goto /page1`, then `tab-new <ts>/page2`, then `snapshot -i`, then `mousewheel`, then `tab-list`, then `tab-close <current id>`, then `eval 1+1`, then `screenshot --out <tmp>`. It asserts the output formats from T01, T04, T06, T10 and T12.
4. Append a `## Summary` to evidence.md: tasks done, tasks with no-go, commit range (`git log --oneline main..HEAD | wc -l`).

**Done when:**
- the Full suite command exits 0 (Baseline excepted);
- `govulncheck` prints `No vulnerabilities found.`;
- `go test ./tests/ -run TestSmokeAntigravity -count=1` prints `ok`.
