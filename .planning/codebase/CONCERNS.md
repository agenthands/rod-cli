# Concerns

**Mapped:** 2026-06-18

## Technical Debt & Areas of Concern

1. **Lack of Automated Testing**:
   - There are few or no visible `*_test.go` files in the core logic paths (`tools/`, `types/`). This makes iterating on the snapshotting logic or context management risky, as regressions might break downstream LLM agents.

2. **State Management**:
   - The CLI holds a single shared browser context in `types.Context`. If multiple requests come through the MCP stdio connection concurrently (which some MCP clients might do), there could be race conditions or unintended cross-talk between tool calls. 

3. **JS Build Step Dependency**:
   - The project requires Node.js and `terser` to build the JavaScript assets (`npm run dev`). This slightly conflicts with the "Zero Dependency Hell" goal stated in the README, though it only affects developers, not end-users. A pure Go minifier (like `tdewolff/minify`) could eliminate the Node.js requirement entirely.

4. **Error Handling Verbosity**:
   - Many errors are simply logged and the process might exit abruptly or return generic errors to the MCP client. Robust, user-friendly error propagation back to the LLM over MCP is crucial for autonomous recovery.
