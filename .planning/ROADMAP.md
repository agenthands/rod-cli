# Roadmap: v1.4 Plugin Architecture

**4 phases** | **11 requirements mapped** | All covered ✓

| # | Phase | Goal | Requirements | Success Criteria |
|---|-------|------|--------------|------------------|
| 17 | Engine Sandbox | 1/1 | Complete   | 2026-06-22 |
| 18 | Lifecycle Emitters | 0/1 | Planned    |  |
| 19 | State Context Sharing | Allow scripts to securely read the DOM snapshot and network state. | PLUG-10, PLUG-11 | 2 |
| 20 | Plugin CLI Interface | Expose commands to load, list, and run plugins manually. | PLUG-07, PLUG-08, PLUG-09 | 2 |

### Phase Details

### Phase 17: Engine Sandbox

**Goal**: Implement the plugin engine sandbox and script loader.
**Depends on**: Phase 16
**Requirements**: PLUG-01, PLUG-02
**Success Criteria**:

1. Engine initializes properly inside the daemon process.
2. A generic script can be parsed and executed via file path.

**Plans**: 1/1 plans complete

- [x] 17-01-PLAN.md

### Phase 18: Lifecycle Emitters

**Goal**: Expose godoll browser events to the plugin engine.
**Depends on**: Phase 17
**Requirements**: PLUG-03, PLUG-04, PLUG-05, PLUG-06
**Success Criteria**:

1. Engine can define an `OnRequest` handler that fires before network dispatch.
2. Engine can define `OnLoad` and `OnResponse` handlers that receive state.
3. Engine can intercept dynamic DOM mutations (`OnDOMNodeInserted`).

**Plans**: 0/1 plans executed

- [ ] 18-01-PLAN.md

### Phase 19: State Context Sharing

**Goal**: Allow scripts to securely read the DOM snapshot and network state.
**Depends on**: Phase 18
**Requirements**: PLUG-10, PLUG-11
**Success Criteria**:

1. Plugin scripts can query the token-optimized snapshot tree.
2. Plugin scripts can read active cookies and local storage via standard API mappings.

**Plans**: TBD

### Phase 20: Plugin CLI Interface

**Goal**: Expose commands to load, list, and run plugins manually.
**Depends on**: Phase 19
**Requirements**: PLUG-07, PLUG-08, PLUG-09
**Success Criteria**:

1. `rod-cli plugin load` injects a script into the daemon memory.
2. `rod-cli plugin list` outputs active loaded plugins.

**Plans**: TBD
