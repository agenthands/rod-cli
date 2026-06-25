---
author: engineer
responsible: engineer
phase: 28-human-behavior-tuning
plan: 02
milestone: v1.6
type: execute
status: complete
requirements: [HUMANIZE-01]
commits: [715b2c4, 869653d]
files_modified:
  - tests/humanize_tuning_test.go
---

# Phase 28 Plan 02 — SUMMARY (verification test)

## Objective achieved

Proved HUMANIZE-01's observable-behavior criteria against the REAL prebuilt
binary: (2) a configured tuning value measurably changes interaction behavior,
and (3) the tuning surface does not regress the harness wall-clock on the default
path. `tests/humanize_tuning_test.go` is the falsifiable evidence that the Plan-01
wiring is live (not silently ignored) AND safe (default path unchanged).

## Must-have truths — status

- **Slow --typing-speed measurably longer than default (criterion 2)** — DONE +
  PASS. The test types the same text "Hello World" in two freshly-spawned
  sessions: one default, one with `--typing-speed-min 150 --typing-speed-max 200`.
  It asserts `slow > default` AND `slow >= default + 500ms` (a clear margin so
  default-path timing noise cannot false-pass). A silently-ignored knob would
  clock the same as default and fail.
- **Default-path timing within the existing bound (criterion 3)** — DONE + PASS.
  Asserts the no-flag path is `>= 300ms` (the existing `tests/humanize_test.go`
  bound), demonstrating the tuning surface adds no default-path regression.
- **Tests exercise freshly-compiled threading, not a stale binary** — DONE. The
  verify command runs `go build -o rod-cli .` before `go test ./tests/`; the
  harness `runCli` execs the prebuilt `../rod-cli`.

## Design of the test (why it is sound)

- **Per-path fresh session.** Stealth/humanize flags are resolved ONCE at session
  spawn (the established model), so each path does `close` then a `goto` that
  carries (or omits) the tuning flags — the flags are frozen onto that session's
  daemon config. Passing them on a later command would NOT apply.
- **False-pass guard.** The `>= default + 500ms` margin means timing jitter on the
  default path cannot accidentally satisfy "slow > default". With default godoll
  speed 30-120ms/key and the slow pin 150-200ms/key over 11 keys, the real gap is
  ~1s+, comfortably inside the margin.
- **False-fail guard.** The slow value is bounded (150-200ms) so the whole test
  runs ~12s (observed: 11.96s) — fast enough to avoid harness timeouts.

## Operational traps respected

- `go build -o rod-cli .` before any `runCli` (stale-binary trap).
- No `pkill -f` anywhere; the suite uses `close` for daemon teardown.
- No interaction with the user's Chromium; only the test session's daemon.
- Targeted `-run TestHumanizeTuning`, never full `go test ./...`.

## Independent review fix (commit 869653d)

The anvil-code-reviewer flagged a [MAJOR test-soundness] risk: the original test
compared two unseeded timing runs with godoll's default 0.02 typo rate active
(typo-correction sleeps are hardcoded +400ms constants independent of the speed
config, firing on both paths). A default run injecting a couple of typos while
the slow run injected none could erode the slow-vs-default gap toward the margin
— an intermittent false-fail. **Resolved:** both comparison paths now pin
`--typo-rate 0`, removing the high-variance term so the base-delay difference is
the only signal; the slow pin was widened to 300-400ms and the required margin to
`>= 1s` (the real gap is ~3s). The `>= 300ms` zero-regression floor remains
deterministic (11 keys × 30ms = 330ms with typos off).

## Verification run at execute time

- `go test ./tests/ -run TestHumanizeTuning -count=1 -v` → PASS (initial 11.96s;
  post-CR strengthened design 13.87s).
- Re-ran both browser tests together post-CR: `TestHumanizedInteractions` PASS
  (10.97s, unchanged), `TestHumanizeTuning` PASS (13.87s).
