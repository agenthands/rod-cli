# Evidence: TS track

## E01

tdd: buildScreenshotArgs keeps a flag-like selector inside its own --selector= token — red: AssertionError: expected [ '-s', 'admin', 'screenshot', …(4) ] to deeply equal [ '-s', 'admin', 'screenshot', …(3) ] — green: (cd extensions/core && npx vitest run src/__tests__/argv.test.ts) passed
tdd: timeoutFor ignores table keys that appear as user values or session names — red: AssertionError: expected 5000 to be 30000 // Object.is equality — green: (cd extensions/core && npx vitest run src/__tests__/timeouts.test.ts) passed
tdd: argv builders (property) every token before the first -- is a builder literal or a single --<own-flag>=<v> — red (fix reverted): seed -1485628428, path "21:1:0:0": {"name":"screenshot","model":{"selector":"-s=victim"},"argv":["screenshot","--selector","-s=victim","--out","/tmp/rod-cli-out.png"]} stray token "--selector" — green: (cd extensions/core && npx vitest run src/__tests__/argv.test.ts) passed
tdd: withSession (property) timeoutFor(withSession(s, a)) equals timeoutFor(a) — red (fix reverted): seed 1138814629, path "2:0:0:0": ["close",["select","--"," "]] AssertionError: expected 5000 to be 30000 — green: (cd extensions/core && npx vitest run src/__tests__/timeouts.test.ts) passed
tdd: withSession (property) validateInput throws for withSession(s, a) exactly when it throws for a — red not observed (no bug in validateInput; property passed with and without the timeoutFor fix) — green: (cd extensions/core && npx vitest run src/__tests__/timeouts.test.ts) passed
verify: E01 — pass — TS checks green (core 147, pi 40, omp 32); reverting either fix in a scratch copy fails its example + property tests; lockfile integrity matches registry.npmjs.org; TMPDIR left only leakless-*/daemon log
