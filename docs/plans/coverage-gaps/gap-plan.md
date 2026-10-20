# rod-cli test-gap plan (merged)

This plan merges the six area proposals into one. IDs keep their original names behind an area prefix. Within each group, rows are ordered by value, then by risk. **RED** means the test fails on today's code.

## (A) Property tests (Go `rapid`, TS `fast-check`)

| id | seam | promise (relation + generator) | catches | value |
|---|---|---|---|---|
| ext:EXT-1 | core argv builders (mainly `buildScreenshotArgs`) | Gen: each model string field is drawn from `fc.string({minLength:1})` or the set {`-s=victim`,`--session=x`,`--json`,`--cdp=ws://x`,`--out`,`--`}. Rel: drop a leading `['-s',session]`. Every token before the first `--` must be a builder literal or a single `--<own-flag>=<v>`. No model value may stand alone as a token. | A screenshot `--selector <sel>` passed as two tokens lets the selector take over `-s` and swallow `--out`. **RED** | high |
| root:root-1 (+daemon:D1 merged) | `runApp` client → `EnsureDaemon` spawn (`os.Args[0]` is a `/bin/sh` recorder that exits 1) → `getApp()` re-parse with a capture Action | Gen: every flag in `getApp().Flags` except {banner,raw,json,max-output,session,proxy-auth}, each unset or set. Strings: non-empty, no NUL, may start with `-` or contain `=` and spaces; profile≠`list`. Ints: `rapid.Int`. Floats: finite. Bools: either. Rel: re-parsed value == client value. IsSet matches for gated flags and non-empty strings (headless/vision compared by value only). The proxy-auth sentinel is absent from argv and `ROD_CLI_PROXY_AUTH=` appears exactly once in env. The error names `$TMPDIR/rod-cli-daemon.log`. | A flag that is never forwarded, misnamed or read with the wrong type; a dropped `=false`; a credential in argv; a lost extraEnv | high |
| types:T1 | `ResolveStealth(cfg,{Profile,UserAgent})` | Gen: P is a builtin profile or `""`. U comes from a literal Win/Mac/Linux OS-token table plus `Chrome/<90..199>`. Fresh cfg per draw. Rel: err==nil ⇒ Platform==table[U]. P==`""` ⇒ no error and Platform==table[U]. | The profile's platform is kept under a foreign `--user-agent`, which leaves a fingerprint tell. **RED** | high |
| internal:cdpproxy-read-normalize-property | `cdpproxy.New(fakeWS).Read()` | Gen: `Runtime.getProperties` responses with optional sessionId. Descriptors have name, optional value/get/set and extra keys. Optional internalProperties, privateProperties, exceptionDetails. Rel: decoded output DeepEqual model, with `value` removed only where `get` exists. No `get` ⇒ byte-identical. Any frame with a `method` ⇒ byte-identical. Read∘Read is idempotent. | The rebuild drops internalProperties, privateProperties and exceptionDetails. **RED** | high |
| root:root-9 | `stealthFlagsFrom(c)` (extracted from runDaemonServer), driven by root-1's capture Action | Same generator as root-1. Rel: a field is set ⇔ the client set the flag, with an equal value (typo-rate compared via float32). Unset ⇒ nil. ProxyAuth==sentinel. A flag that is in neither the field table nor the client-only set fails the test. | A flag that is forwarded but never captured (the cdp-proxy landmine) | medium |
| ext:EXT-5 | `timeoutFor` / `validateInput` / `withSession` | Gen: argv `a` from any builder without a session. `s` from `fc.string({minLength:1})`, plus `close`, `goto`, `--version`. Rel: `timeoutFor(withSession(s,a))==timeoutFor(a)`, and validateInput throws in the same cases. Examples: `['hover','--','close']` → DEFAULT_TIMEOUT. | The table-key scan matches user values, so a command gets a 5s kill or 60s. **RED** | medium |
| types:T6 | `BuildSnapshot(nil,{Interactive})` with pageEval/pageInfo stubbed | Gen: engine-shaped aria-YAML (scalars and single-key maps) with interactive and non-interactive roles, optional name, unique ref with p≈0.5, optional `/url` children, no iframe. Rel: every interactive ref appears as an exact `[ref=…]` token. Refs with no interactive descendant are absent. The output is a fixed point when fed back in. | Controls lost in prune or flatten | medium |
| actions:A4 (live) | `Fill` with TypoRate 0.5 and typing speed 1–2 | Gen: `rapid.StringOf(L,N,P,S,Zs)` of length 0–20, for both the text and the existing field value. Rel: `el.value==text`. Use `-rapid.checks=30` and one ctx. | Typing knobs, clear-then-type, godoll typo correction | medium |
| types:T3 | `ResolveStealth` unsafe-value gate | Gen: a clean string matching `[ -!#-\[\]-~]{0,40}` with one bad rune from {`"`,`\`,0x00–0x1f,0x7f} inserted, placed in one field (UA/Timezone/Locale/Platform flag, Vendor, AcceptLanguage, Languages[k]). Rel: the error starts with `fingerprint field <label> ` (labels from a literal table). A clean Timezone alone never errors. | A field dropped from the check list; off-by-one at 0x1f/0x7f | medium |
| root:root-2 | `reorderGlobalFlags(getApp(),argv)` | Gen: labelled segments: global bools (name or alias), value flags as `--x=v` or `--x v`, plain tokens that never look like global flags (`-`,`-10`,`---`,`--out`, text), and an optional `--` with any tail. Rel: the output is a permutation with [0] fixed. Plain tokens keep their order after all globals. Each value flag is directly followed by its value. The `--` tail is a byte-identical suffix. Idempotent. | Lost or duplicated tokens, broken value pairing, `--` regressions | medium |
| internal:cdpproxy-traffic-ringbuffer-model | `Proxy.Send/Read/Traffic` over a fake WebSocket | Gen: cap in [1,8]; ops are send (inner may fail), recv ok, recv err. Rel: `Traffic()` == the last min(cap,n) entries of a list model. A send is logged even when the inner Send fails; a recv is logged only on success. Examples: cap≤0 holds 256 entries. Mutating a buffer after the call leaves Traffic() unchanged. | Ring-buffer off-by-one; aliasing of rod's reused buffers | medium |

## (B) Example and unit tests

| id | seam | promise | catches | value |
|---|---|---|---|---|
| daemon:D3 | `StartServer` /execute | `{"command":"bogus-cmd"}` sent with `Origin: http://evil.example` or as `text/plain` is rejected before dispatch. A request shaped like ClientExecute's still gets `unknown command`. Use a temp TMPDIR and `Config{}`, and never send `close`. | Any web page can drive the session. **RED** until S1 is fixed | high |
| types:T4 | `LoadConfig` | A missing path returns (nil, err naming the path). `other.yaml`, `cli.yaml` and `yaml` return (nil, err). Replace config_test.go:304/316. | (nil,nil) leads to a nil-deref panic under `--headless`; the name check is inverted. **RED** | high |
| actions:A2 | `Press` (keyboardPress stub) | Delete, Home, End, PageDown, Space and F5 each send the named key or an `unknown key` error, never the key for their first byte. `é` never sends 0xC3. `a`, `A`, `1` are unchanged. | The wrong key is sent and success is reported. **RED** | high |
| root:root-3 | `runApp` | `sessions --session` and `goto https://x -s` return an error (the text is not pinned). | A trailing global flag with no value swallows the subcommand and exits 0. **RED** | high |
| daemon:D2 | `StartServer` → `removeDeadPortFiles` | With a temp TMPDIR: a dead-port file is removed; a live-listener file is kept; a dead-port file whose `.lock` is held is kept; the server's own port file exists. | The sweep deletes the port file of a live or mid-spawn session, which leads to a duplicate Chrome. | high |
| types:T2 | `ResolveStealth` table | Mac UA + `--platform Win32` → `contradicts UA OS`. UA without an OS token + Solaris → `could not be verified`; with Linux it passes. With a UA set: `fr-FR` vs `[en-US]` is rejected, `en_GB` passes, no locale gives `en-US`. Win and Linux UAs derive their platform. Screen and hardware bounds checked on both return paths. | A reject or derive branch going silent | high |
| ext:EXT-2 | pi `extension(mockPi)` + `registerLifecycle` | With `findRodCli` mocked to F: session_start runs `exec(F,['--version'])`, shutdown runs `exec(F,['close'])`, and the tools also exec F. Fix adversarial.test.ts:106/142. | The bare `rod-cli` fails with ENOENT, `close` never reaches the daemon, and Chrome keeps running. **RED** | high |
| ext:EXT-3 | core `sleep` / `waitForSelector` | (a) A resolved sleep leaves 0 abort listeners. (b) With fake timers and stdout false, false, true: 3 calls and the exact eval argv. (c) Timeout message is exact. (d) Abort rejects with `signal.reason` and stops polling. (e) A session adds `-s` after the prefix. | Abort-listener leak (MaxListenersExceeded). **RED** (a) | high |
| daemon:D4 | `ListSessions` | A port file points at a listener that never accepts. The call returns within 5s, with t.Fatal as the bound, and the session is not listed. | `http.Post` with no timeout hangs `sessions` and every command for that session. **RED** until S2 is fixed | medium |
| daemon:D6 | `executeAction` numeric args | `request x`, `resize abc 600`, `mousemove 1 y` and `mousewheel 0 abc` each error and name the bad value. Uses a bogus `CDPEndpoint`, so no Chrome. | Ignored Sscanf errors turn bad input into a silent 0. **RED** until S3 is fixed | medium |
| root:root-5 | `main()` via runMainSubprocess | `--json=true click` and `-json click` print a JSON `{"error":…}` and exit non-zero. `--json=false` prints plain text. | JSON error mode is detected only for an exact `--json`. **RED** | medium |
| root:root-6 | `runApp` screenshot/pdf `--out` with a recording fake daemon and t.Chdir | The daemon receives `out==Join(tmp,"rel/x.png")`. Without `--out` it receives `""`. | A relative path resolved against the daemon's cwd | medium |
| root:root-7 | `runApp --json` with a fake daemon whose result string is configurable | An object result is printed verbatim. A string result prints as `{"result":…}`. Pin `[1,2]` printing bare as a deliberate decision. | The JSON output shape that agents parse | medium |
| root:root-4 | `runApp --profile=list` (`os.Args[0]` missing) | `--raw` == `BuiltinProfileNames()`. `--json` names match, with non-empty fields and screen `^\d+x\d+$`. Default output has one line per name. With a subcommand the call returns nil, so no spawn. | Profile list launching a daemon; the output formats disagreeing | medium |
| actions:A7 | `Drag` seams | When the 2nd move fails, mouseUp is called exactly once. When mouseDown fails, mouseUp is called 0 times. Both errors carry their prefixes. | A mouse button left stuck after a refactor | medium |
| actions:A10 | `MouseWheel` horizontal, with a fake scroller | Exact strings for `dx=300` and then `--end` (`x=1000/1000 … end`). wheelSettle=0. | X/Y mix-up; a sign error in `--end` | medium |
| types:T5 | profile resolution with HOME=tmp | `custom` resolves to `~/.rod-cli/profiles/custom.json` with UA and Platform copied. `nope` gives an error naming the path. Traversal-shaped names return (nil,false,nil). | The D-02 home fallback; traversal names | medium |
| types:T7 | `BuildSnapshot` Gen:2 with a pageEval stub | After a "not defined" error, the engine is injected once and the retry succeeds. When it always fails, the call errors and the engine is still injected exactly once. | Re-inject on every retry | medium |
| ext:EXT-9 | `captureScreenshot` | The execFn writes the file and then returns code 1: the call rejects and the file is gone. With code 0 it returns the base64 and the file is gone. | Leaked temp files in RAM `/tmp` | medium |
| ext:EXT-12 | `execRodCli` + `withSession` | After the charset decision: session `/../../tmp/evil` is rejected before exec, and `admin` passes. | A model-supplied session escaping TMPDIR (defense in depth for S6) | medium |
| actions:A5 | `MouseWheel`, with measureScroll returning `[]` | Returns an error containing `Failed to measure scroll`. | `errors.Wrap(nil)` returns ("",nil). **RED** | low |
| internal:utils-ishttp-scheme-case | `utils.IsHttp` | `HTTP://` and `Https://` → true. `ftp`, `httpx` → false. | Case-sensitive scheme check. **RED** | low |
| types:T10 | `SweepStaleProfiles` with readCmdline failing | A dir owned by our own pid is kept. A regular file is left alone. A dir owned by a reaped pid is removed. | Live profiles deleted on macOS or Windows | low |
| actions:A13 | `StealthCheck` with runtimeEvaluateCall stubbed | Exact FAIL string for the listed signals. A read error gives `read-error`. All passing gives `PASS`. | Verdict threshold regressions | low |
| actions:A11 | `MouseWheel` ScrollDuration=400 | dy=400: each gap between scroll calls is ≥90ms (lower bound only). | Units mistake, or d vs d/n | low |

## (C) Live tests (Chrome)

| id | seam | promise | catches | value |
|---|---|---|---|---|
| actions:A3 | `Select` | Options Item 10 and Item 1: `"Item 1"` selects value 1. `"1"` never selects Item 10. `ban` selects Banana through the fallback. | The substring text match picks the wrong option. **RED** | high |
| internal:plugin-engine-concurrent-hooks-race | `BindLifecycle` + `RunFunc` under `-race` | Background navigations run while RunFunc is called in a loop. No race is reported, and the count reaches N. | `goja.Runtime` shared across goroutines. **RED** under -race | high |
| ext:EXT-6 | core builders → the repo binary (pi integration harness) | fill, select, scroll, tabs, screenshot, hover and press each work and leave the expected DOM state, and no stray files. | Builder argv drifting from Go parsing | high |
| actions:A6 | `Evaluate` | A declaration plus a call → `5`. An `async function` → `7`. `(a=1) => a+1` → `2`. A null element result → `null`. Never prints `<nil>`. | The isFunc heuristic causes SyntaxErrors and a silent `<nil>`. **RED** | medium |
| actions:A1 | `StateSave`/`StateLoad` | Save, clear, load: a session cookie and a persistent cookie both come back. File `perm&0077==0`. | Cookie jar written 0644. **RED** (mode) | medium |
| internal:plugin-bind-twice-single-dispatch | `PluginLoad` ×2 | One navigation leaves the count at `1`. | Duplicate EachEvent goroutines. **RED** | medium |
| types:T8 | `AddRoute`/`RemoveRoute` | With two mocks, partial removal works, and removing the last route sets the interceptor to nil. After the order fix, overlapping patterns serve the documented winner every time. | Multi-route rebuild; map-order nondeterminism | medium |
| daemon:D5 | `ClientExecute(close)` on a real daemon | When `closing` returns, the port file, profile dir and chromeTmp are already gone. The pid exits within 5s, and a later ping returns `not running`. | Cleanup moved to after the reply | medium |
| ext:EXT-7 | `waitForSelector` with the real execFn | `#msg` resolves. `#nope` gives the exact timeout message. | Eval output format drift | medium |

## (D) Suspected bugs (each gets a red test first)

| id | seam | promise (red test) | catches (bug / fix) | value |
|---|---|---|---|---|
| daemon:S1 | daemon.go:283-289 | daemon:D3 | /execute has no Origin, Host or Content-Type check. Fix: reject any Origin, non-JSON bodies, and Host ≠ `127.0.0.1:<port>`. | high |
| daemon:S7 | daemon.go:65-72, 214 | New: port file content `1@evil.example:80` → the client errors without dialing. | Port files are 0644 and trusted. Another user can hijack the session's args or send them off-host. Fix: `strconv.Atoi`, an owner uid check, or a 0700 per-user dir. | high |
| daemon:S6 (+ext dup) | daemon.go:59-61, cmd.go:122 | New Go test: session `x/../../../etc/foo` is rejected before any file op. ext:EXT-12 covers the TS side. | The `.port` and `.lock` files are created and removed outside TMPDIR. Fix: validate the session in Go. | high |
| daemon:execSem | daemon.go:307-309; actions.go:1180, 1197 | New: in an in-process daemon, an action panics, and the next command must not get `previous command has not completed`. Needs a test-only panicking action. | A single panic wedges the session. Fix: `defer` the release, and check the `page.Info` error. | high |
| types:LoadConfig | config.go:932, 946-948 | types:T4 | Returns (nil,nil) and the name check is inverted. Fix: `errors.Errorf` and `fileName==ConfigName`. | high |
| ext:screenshot-selector | argv.ts:58 | ext:EXT-1 | Fix: emit `--selector=<sel>` as one token. | high |
| types:platform | config.go:416-417, 717-724 | types:T1 | Fix: re-derive the platform from a flag UA, or reject the combination. | high |
| actions:parseKey | actions.go:1036-1058 | actions:A2 | Fix: map named keys, and error on unknown keys. | high |
| actions:selectOne | actions.go:490-491 | actions:A3 | Fix: exact text first, then value. | high |
| root:trailing-flag | cmd.go:286-289 | root:root-3 | Fix: leave the token in `rest`, or return an error. | high |
| internal:goja-race | lifecycle.go:37-47, engine.go:65-89 | internal:plugin-engine-concurrent-hooks-race | Fix: a mutex around every VM use. | high |
| internal:cdpproxy-drop | filters.go:66-67 | internal:cdpproxy-read-normalize-property | Fix: use `map[string]RawMessage` and replace only `result`. | high |
| ext:pi-lifecycle | pi lifecycle.ts:30, 43 | ext:EXT-2 | Fix: use rodCliPath, as omp already does. | high |
| daemon:S2 | daemon.go:72 | daemon:D4 | Fix: a short timeout on ping only (real commands take up to 60s plus about 45s). | medium |
| daemon:S3 | daemon.go:383-453 | daemon:D6 | Fix: parse with strconv and return the error. | medium |
| root:json-detect | main.go:170-178 | root:root-5 | Fix: detect JSON mode from the parsed flag, not os.Args. | medium |
| actions:evaluate | actions.go:275-284 | actions:A6 | Fix the isFunc heuristic. | medium |
| actions:state-0644 | actions.go:1410 | actions:A1 | Fix: write the file 0600. | medium |
| internal:bind-twice | lifecycle.go:20, 37; plugin.go:27 | internal:plugin-bind-twice-single-dispatch | Fix: bind once per page, and honour ctx. | medium |
| ext:timeoutFor | timeouts.ts:28-30 | ext:EXT-5 | Fix: run the key scan only when cmdIndex<0. | medium |
| ext:sleep-leak | wait.ts:12-18 | ext:EXT-3 | Fix: remove the listener when the timer fires. | medium |
| types:route-order | context.go:994 | types:T8 follow-up | Rules come out in map order. Fix: sort or keep insertion order, then pin. | medium |
| daemon:S4 | daemon.go:97-101, 214 | New: ListSessions keeps a dead-port file whose `.lock` is held (same setup as D2). | Deletes without a lock. Fix: `tryLockFile`, and write the port file atomically. | medium |
| actions:tabnew-orphan | actions.go:916-930 | New live test: `TabNew("file:///x")` errors and the tab count and active tab are unchanged. | Leaves an orphan about:blank tab active. Fix: validate the URL before creating the tab. | medium |
| root:eval-json-shape | cmd.go:103-110, 220 | root:root-7 pins today's behaviour | The output shape depends on the value, and eval results bypass `--max-output`. Needs a decision. | medium |
| daemon:S5 | daemon.go:284 | Needs an idle-timer seam. | Ping resets the idle timer, so polling keeps every daemon alive. Fix: skip the reset for ping. | low |
| actions:wrap-nil | actions.go:775-776 | actions:A5 | Fix: `errors.New` when the result is empty. | low |
| internal:ishttp | utils/net.go:5-7 | internal:utils-ishttp-scheme-case | Fix: case-insensitive scheme check. | low |
| actions:selectByValue | actions.go:511 | New: an option value containing `\t` can be selected. | `strconv.Quote` is not a CSS escape. Fix: `CSS.escape`. | low |

Cosmetic items to fix without a test:
- cdp_traffic.go:39-40 cuts the preview at byte 120, which can split a UTF-8 rune (the scanner has the same pattern).
- `GetChinaZoneTime` ignores the parse error.
- validate.ts says "bytes" but counts UTF-16 units.
- snapshotRootExpr can produce an octal literal.

Decisions to make before testing:
- Session charset (S6/EXT-12).
- Whether an empty cookie or storage value is allowed (EXT-14).
- Whether waitForSelector retries after an exec rejection and polls when timeoutMs≤0.
- pruneInteractive:338 contradicts its doc comment.
- Partial-config merge in LoadConfig.

## Prerequisites (listed once)

- **Go:** `rapid` is already in go.mod. No new dependencies.
- **TS:** add `fast-check` as a devDependency to `extensions/core` only (EXT-1, EXT-5). pi and omp need nothing new.
- **Daemon test hygiene (S8):** every daemon test sets `t.Setenv("TMPDIR", t.TempDir())`. This includes fixing TestListSessionsPrunesDead, TestInProcessDaemonPingAndActions, TestStartServerAnswersPing and the EnsureDaemon tests, which today ping and delete the developer's real port files.
- **Race detector:** add `go test -race ./internal/plugin/...` to the Makefile and CI.
- **Test helpers:** a recording fake daemon whose result string is configurable (root-6, root-7). Write it locally in the test, with no production change.
- **Refactor:** extract `stealthFlagsFrom(c)` in main.go (root-9). This changes no behaviour.
- **Fixtures:** add `<select>` options Item 10/Item 1 to the actions fixture (A3). Add `<select id="c">` and a scrollable `#pane` to the pi `FIXTURE_HTML` (EXT-6).
- **Fixes that turn red tests green:** S1 (D3), S2 with a ping-only timeout (D4), S3 (D6).
- **Not parallel-safe:** root-1 and root-4 swap `os.Args[0]`. Restore it in t.Cleanup.
- **Live tests:** close every session and remove profiles on exit, because `/tmp` is tmpfs. Run EXT-1's hostile live cases only after the EXT-1 fix; before it they spawn a stray session that nothing closes.

## Not worth testing

- **Dropped as low value:**
  - root-8: covered by root-2(d) and D6.
  - A8 snapshot errors.
  - A9 Pdf names: `filepath.Base` already rules out traversal.
  - A12 TabSelect prefix.
  - T9 proxy parse: the private helper is already tested.
  - D7 mousewheel end/max: Chrome cost, and actions already pins it.
  - EXT-8: JSON.stringify is correct today.
  - EXT-10: thin pass-throughs.
- **Merged or dropped proposals:**
  - D1 merged into root-1.
  - EXT-11 merged into EXT-2.
  - EXT-13 is covered by EXT-5.
  - EXT-4's premise is false: Navigate already rejects non-http(s) URLs. The orphan-tab bug is listed in D.
  - D8 (legacy argList fallback) and D9 (cdp-traffic without --cdp-proxy) are one-line pass-throughs.
- Scanner BuildTestURL and ExtractEvidence: no production importer. Delete the package instead.
- `detect` handleResult: an unused fixture. Delete it.
- Seam var default bodies, `json.Marshal`/Glob/Abs failures, and wrap-and-forward error paths in actions and daemon: plain propagation.
- `exitDaemon`, the signal goroutine, StartServer listen/write errors and main.go:161-165: these need a subprocess or Chrome, and the outcomes are already covered.
- Seams that would only be needed to provoke rare failures: Navigate's 30s deadline, `launchBrowser` failures, profile_root OS branches, crypto/rand fallback, lock OpenFile errors.
- Fallbacks for corrupt embedded profiles, which TestBuiltinProfilesAreVetted guards.
- capOutput (has property tests), the install command (tested, and it downloads Chromium), plugin.go (already at 100%).
- `profileFromStealth` and identity helpers, the CDP proxy wrapper, and CDPTraffic JSON: already covered end to end in tests/.
- mouseOpts randomness, Show `--annotate` (needs a human), the argList roundtrip (tests encoding/json), and a dispatch-table scan (brittle).
- Description and prompt strings, re-export and index files, Windows branches in binary.ts, NaN scroll args, the omp mock shim, and the exec null-code message.
- `lock_windows.go`: no Windows CI.
- Rejected suspected bugs:
  - The stealthRequested warning scope is deliberate.
  - `ROD_CLI_PROXY_AUTH` left set in the daemon env is hygiene only.
  - GetCDPProxy has no race: execSem serializes access.
  - Using a bare `--profile` name from the cwd is documented behaviour.
  - StateSave session-cookie loss was refuted live.