---
author: engineer
responsible: engineer
phase: 29-best-effort-live-validation
plan: 02
milestone: v1.6
type: execute
status: complete
requirements: [LIVEWAF-01]
commits: [25a1b90]
files_modified:
  - docs/stealth-validation.md
  - README.md
---

# Phase 29 Plan 02 — SUMMARY (honest-ceiling documentation)

## Objective achieved

Authored the honest-ceiling documentation (LIVEWAF-01 criterion 3): a doc stating
plainly what rod-cli's stealth validation proves (Tier-1 offline, the JS layer)
vs what it cannot (the TLS/IP/CDP layers), with no "undetectable" overclaim, plus
how to run the opt-in live suite. Linked from the README so it is discoverable.

## Must-have truths — status

- **Doc states Tier-1 (offline, blocking) deterministic proof vs Tier-2 (live,
  best-effort) signal** — DONE. Two clearly-labelled sections; Tier-1 enumerates
  the JS-layer signals it proves (UA/Client-Hints/platform/WebGL/canvas/audio/
  WebRTC/webdriver/...), Tier-2 frames the live suite as best-effort
  informational.
- **Explicitly names TLS-fingerprint (JA3/JA4), IP-reputation, and CDP-footprint
  layers a JS-injecting CLI CANNOT control; NO "undetectable" guarantee** — DONE.
  A dedicated "The honest ceiling" section names all three layers with a sentence
  each on why JS injection can't close them; the word "undetectable" appears only
  as an explicit NON-guarantee (TL;DR + closing line). A "no challenge observed"
  result is stated to be not a pass guarantee.
- **Documents how to run: `go test -tags detection_live ./tests/ ...`** — DONE,
  with a fenced block including the mandatory `go build -o rod-cli .` first and a
  note that outcomes are informational/non-blocking.
- **Linked from README** — DONE. Added to the Documentation list with an honest
  one-line description naming the no-"undetectable" ceiling.

## Cross-reference + tone

Cross-references `docs/cdp-footprint.md` twice (inline in the CDP-layer bullet and
in "See also") — that existing doc already establishes the same honesty tone and
names the TLS/IP/CDP ceiling, so the new doc extends rather than contradicts it.
Relative links verified to resolve: `cdp-footprint.md` (sibling), `../ARCHITECTURE.md`
(from docs/), `docs/stealth-validation.md` (from README).

## Why the honesty is the deliverable (not garnish)

CONTEXT is explicit: a doc implying "passes Cloudflare ⇒ undetectable" would FAIL
criterion 3. The doc's TL;DR leads with "rod-cli does NOT claim to be
'undetectable'", and the ceiling section names exactly the layers (TLS/IP/CDP)
that make a clean live run insufficient proof — encoding the project's
validate-live-not-source / no-overclaim ethos.

## Verify checks (plan-specified)

- `test -f docs/stealth-validation.md` → present.
- `grep -iE "tier|undetectable|tls|ja3|cdp|best-effort"` → all present (Tier 1/2,
  TLS×5, JA3, JA4, CDP, best-effort×4, IP reputation, undetectable×3-as-negation).
- `grep -n "stealth-validation" README.md` → linked (line 19).
- `grep -n "cdp-footprint" docs/stealth-validation.md` → cross-ref present.
