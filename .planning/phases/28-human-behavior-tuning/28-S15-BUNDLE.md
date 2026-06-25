---
author: substrate
responsible: architect
emitted_at: 2026-06-25
phase: 28
artifact_kind: analysis
status: draft
---

# §15.8 replication bundle (B2 debt_result + B3 run_cost — phase 28)

## Records

```yaml
debt_result:
  raw_debt: 0
  normalized_debt: 0
  authorized_debt: 0
  unauthorized_debt: 0
  denominator:
    lockable_records_count: 0
    active_obligations_count: 0
  by_component:
    "1":
      total: 0
      contributing_records: []
    "2":
      total: 0
      contributing_records: []
    "3":
      total: 0
      contributing_records: []
    "4":
      total: 0
      contributing_records: []
    "5":
      total: 0
      contributing_records: []
    "6":
      total: 0
      contributing_records: []
      reason: no decision carries decision_lock_scope.current_locked_layer ∈ {design,
        implementation_bound} (§12.2 component-6)
    "7":
      total: 0
      contributing_records: []
      reason: no enforcement_debt declarations supplied (§11.6 deferral is prose-only,
        not machine-readable)
  by_severity:
    critical: 0
    high: 0
    medium: 0
    low: 0
  by_accountable_owner:
    architect: 0
    engineer: 0
    qa: 0
    security-engineer: 0
    codebase-archaeologist: 0
    user: 0
  trend:
    series: []
    raw_slope: 0
    normalized_slope: 0
  v11_signals:
    within_milestone_convergence: true
    convergence_threshold_applied:
      relative_peak_factor: 0.5
      absolute_floor: 2
      effective_threshold: 2
    per_component_convergence: true
    cross_milestone_trend: warn
  as_of: null
  as_of_phase: 28
run_cost:
  wall_clock_seconds: null
  total_tokens_consumed: null
  total_worker_invocations: null
  budget_remaining: null
```
