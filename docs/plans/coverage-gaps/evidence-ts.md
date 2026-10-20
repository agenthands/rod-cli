# Evidence: TS track

## E01

tdd: buildScreenshotArgs keeps a flag-like selector inside its own --selector= token — red: AssertionError: expected [ '-s', 'admin', 'screenshot', …(4) ] to deeply equal [ '-s', 'admin', 'screenshot', …(3) ] — green: (cd extensions/core && npx vitest run src/__tests__/argv.test.ts) passed
tdd: timeoutFor ignores table keys that appear as user values or session names — red: AssertionError: expected 5000 to be 30000 // Object.is equality — green: (cd extensions/core && npx vitest run src/__tests__/timeouts.test.ts) passed
