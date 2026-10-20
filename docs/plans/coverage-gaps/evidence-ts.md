# Evidence: TS track

## E01

tdd: buildScreenshotArgs keeps a flag-like selector inside its own --selector= token — red: AssertionError: expected [ '-s', 'admin', 'screenshot', …(4) ] to deeply equal [ '-s', 'admin', 'screenshot', …(3) ] — green: (cd extensions/core && npx vitest run src/__tests__/argv.test.ts) passed
tdd: timeoutFor ignores table keys that appear as user values or session names — red: AssertionError: expected 5000 to be 30000 // Object.is equality — green: (cd extensions/core && npx vitest run src/__tests__/timeouts.test.ts) passed
tdd: argv builders (property) every token before the first -- is a builder literal or a single --<own-flag>=<v> — red (fix reverted): seed -1485628428, path "21:1:0:0": {"name":"screenshot","model":{"selector":"-s=victim"},"argv":["screenshot","--selector","-s=victim","--out","/tmp/rod-cli-out.png"]} stray token "--selector" — green: (cd extensions/core && npx vitest run src/__tests__/argv.test.ts) passed
tdd: withSession (property) timeoutFor(withSession(s, a)) equals timeoutFor(a) — red (fix reverted): seed 1138814629, path "2:0:0:0": ["close",["select","--"," "]] AssertionError: expected 5000 to be 30000 — green: (cd extensions/core && npx vitest run src/__tests__/timeouts.test.ts) passed
tdd: withSession (property) validateInput throws for withSession(s, a) exactly when it throws for a — red not observed (no bug in validateInput; property passed with and without the timeoutFor fix) — green: (cd extensions/core && npx vitest run src/__tests__/timeouts.test.ts) passed
verify: E01 — pass — TS checks green (core 147, pi 40, omp 32); reverting either fix in a scratch copy fails its example + property tests; lockfile integrity matches registry.npmjs.org; TMPDIR left only leakless-*/daemon log

## E02

tdd: index.ts — session_start, shutdown and tools all exec findRodCli()'s path — red: AssertionError: expected last "spy" call to have been called with [ '/opt/fake/rod-cli', …(2) ] (received "rod-cli") — green: (cd extensions/pi && npx vitest run src/__tests__/adversarial.test.ts) passed
tdd: sleep leaves no abort listener on the signal once it resolves — red: AssertionError: expected [ [Function] ] to have a length of +0 but got 1 — green: (cd extensions/core && npx vitest run src/__tests__/wait.test.ts) passed
tdd: waitForSelector polls every 500ms until eval prints true (3 calls, exact eval argv) — red not observed (characterizes existing behaviour) — green: (cd extensions/core && npx vitest run src/__tests__/wait.test.ts) passed
tdd: waitForSelector rejects with the exact timeout message when the selector never appears — red not observed (characterizes existing behaviour) — green: (cd extensions/core && npx vitest run src/__tests__/wait.test.ts) passed
tdd: waitForSelector with timeoutMs 0 / -1 checks exactly once and does not poll — red: Error: Timed out waiting 0ms for selector: #x (0 exec calls; same for -1ms) — green: (cd extensions/core && npx vitest run src/__tests__/wait.test.ts) passed
tdd: waitForSelector propagates an exec rejection without retrying — red not observed (characterizes existing behaviour) — green: (cd extensions/core && npx vitest run src/__tests__/wait.test.ts) passed
tdd: waitForSelector rejects with signal.reason on abort and stops polling — red not observed (characterizes existing behaviour) — green: (cd extensions/core && npx vitest run src/__tests__/wait.test.ts) passed
tdd: waitForSelector puts -s <session> after the --raw/--max-output prefix — red not observed (characterizes existing behaviour) — green: (cd extensions/core && npx vitest run src/__tests__/wait.test.ts) passed
