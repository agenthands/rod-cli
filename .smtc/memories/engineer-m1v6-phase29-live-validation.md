# engineer — Phase 29 (v1.6) Best-effort live validation (LIVEWAF-01)

## What it did (the milestone's last build phase)
Added an opt-in, NON-blocking Tier-2 live detection smoke test +
honest-ceiling docs. Commits 7526360 (test+CI comment) 25a1b90 (doc+README)
a47e949 (CR fix) 17c600a (SUMMARYs).

## The verifiable bar was EXCLUSION + HONESTY, never a green live result
A live green is flaky by nature and explicitly NOT the bar. What you prove:
1. EXCLUSION by construction — `//go:build detection_live` (Go 1.25 form, FIRST
   line, blank line 2). Plain `go build ./...`/`go test ./...` and CI
   (`go test ./... -count=1`, NO -tags) never compile it. PROVE it empirically:
   `go test ./tests/ -list` (no tag) → TestLiveDetection ABSENT; with
   `-tags detection_live` → PRESENT. This is the strongest exclusion proof.
2. NON-BLOCKING in BOTH senses: excluded from CI (tag) AND informational within
   the suite — every verdict t.Logf, unreachable target/no-egress t.Skip, and
   ZERO t.Fatal/t.Errorf on a live detection or network error. grep
   't\.(Fatal|Error|Fail)' the file → should match ONLY the doc-comment.
3. HONESTY doc names the layers a JS-injecting CLI CANNOT control: TLS
   fingerprint (JA3/JA4), IP reputation, CDP transport footprint. NO
   "undetectable" guarantee (use the word ONLY as a negation). A doc implying
   "passes Cloudflare => undetectable" FAILS the criterion.

## First build-tagged test in the repo
`//go:build detection_live` then blank line then `package tests`. Verify with
`go build -tags detection_live ./...` (compiles = valid Go, not dead code) AND
`go vet` both with/without the tag. Symbols from non-tagged same-package files
(runCli in cli_test.go, evalResultPrefix in detection_test.go) resolve fine.

## CI exclusion = comment only, NEVER change the gate
test.yml gate stays `go test ./... -count=1` (no -tags). Add a "DELIBERATE
EXCLUSION / Do NOT add -tags detection_live" comment so a future maintainer
doesn't "helpfully" re-enable it. (Editing a GH Actions workflow trips a security
hook — a comment-only edit has no injection surface; fine.)

## CR fixes (anvil-code-reviewer, verdict SHIP, both MINOR)
- best-effort eval helper returns (val, ok); when you fold a SECONDARY read into a
  heuristic, HONOR ok (`body, bodyOK := ...; if !bodyOK { body="" }`) — else tool
  error text can spuriously flip an informational verdict.
- a skip/error message must not assert a cause you didn't establish: runCli execs
  the PREBUILT ../rod-cli, so a goto error = unreachable OR binary-not-built; name
  both, don't claim "no egress".

## Honesty-doc precedent in the repo
docs/cdp-footprint.md already set the tone + names TLS/IP/CDP ceiling. Cross-ref it
rather than contradict. README Documentation list uses root-relative md links.

## Operational (unchanged)
go build -o rod-cli . before runCli; pkill -9 -x rod-cli only; never the user's
Chromium; never `go test ./...`; targeted -run. The live suite needs egress and is
built to SKIP without it — don't treat a skip as failure. See
[[engineer-m1v6-phase28-humanize-tuning]] and
[[engineer-m1v6-phase27-canvas-webrtc-hardening]].
