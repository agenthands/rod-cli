# Evidence: TS track

## E01

tdd: buildScreenshotArgs keeps a flag-like selector inside its own --selector= token — red: AssertionError: expected [ '-s', 'admin', 'screenshot', …(4) ] to deeply equal [ '-s', 'admin', 'screenshot', …(3) ] — green: (cd extensions/core && npx vitest run src/__tests__/argv.test.ts) passed
