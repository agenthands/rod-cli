---
author: architect
responsible: architect
phase: 44
phase_type: implementation
hard_bar: false
security_relevant: false
design_fork: false
status: locked
parent_artifacts:
  - .planning/REQUIREMENTS.md (PROXY-03, PROXY-04)
---

# Phase 44: Jitter Validation + Sensitivity Warning — CONTEXT/PLAN

## PROXY-03: Jitter soft-warning
In `types/context.go launchBrowser()`, when `jitterMs > 1000`, print to stderr:
`"warning: --cdp-jitter-ms=%d is very high; navigation may be extremely slow\n"`

## PROXY-04: cdp-traffic sensitivity caveat
Update `cmd.go` cdp-traffic command Usage to include:
`"WARNING: output may contain sensitive CDP payload data (URLs, cookies, page content)"`

## Success criteria
1. `go build ./...` passes
2. `--cdp-jitter-ms=5000` produces stderr warning
3. `cdp-traffic --help` shows sensitivity caveat
