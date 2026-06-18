# Phase 5: Annotation & Debugging

**Status:** Planned
**Goal:** Deliver powerful feedback and diagnostic tools for developers and visual reasoning models.
**Requirements:** DBG-01, DBG-02, DBG-03

<domain>
## Context & Scope
The final phase focuses on observability and human-in-the-loop workflows. As agents perform actions autonomously, humans (or secondary Vision-Language Models) often need to verify exactly what happened. This phase implements video recording of the browser viewport, visual element highlighting (to prove the agent selected the correct target), and an interactive `show` command tailored for UI design and visual validation feedback.
</domain>

<design>
## Technical Design
1. **Video Recording (DBG-01)**:
   - Introduce `video-start` and `video-stop` commands.
   - We will utilize `go-rod`'s built-in screencast/video recording capabilities to stream the viewport frames to an `.mp4` or `.webm` file.
2. **Visual Highlighting (DBG-02)**:
   - Introduce a `highlight [selector]` command.
   - Use `Page.Eval` to dynamically inject CSS that draws a highly visible, persistent red border (or overlay) around the targeted element.
   - Provide a `highlight-clear` command to remove existing highlights.
3. **Interactive UI (`show --annotate`) (DBG-03)**:
   - Rather than just returning text, `rod-cli show` will un-hide the browser (temporarily turn off headless mode or leverage a VNC/WebSocket debug UI) to let the user visually inspect the current state.
   - Passing `--annotate` will drop a transparent overlay on the screen allowing users to click and visually mark areas that are failing, which the CLI will translate back into coordinates/selectors for the LLM. *(Note: Given CLI constraints, we may implement this as a server that returns an HTML page with the snapshot and an annotation canvas).*
</design>

<tasks>
## Task Breakdown

### 1. Highlight Commands
- Add `Highlight` and `ClearHighlights` functions to `actions/actions.go`.
- Inject a `<style id="rod-cli-highlight">` block into the DOM for tracking highlighted elements.
- Map `highlight` and `highlight-clear` in the IPC Daemon and `cmd.go`.

### 2. Video Recording Commands
- Add `VideoStart` and `VideoStop` to `actions/actions.go`.
- Use `page.Context(...)` or `rod.Video()` equivalent to begin dumping frames to `tmp/videos/`.
- Ensure the state (video closer/cancel func) is held safely within the daemon's runtime memory.
- Map `video-start` and `video-stop` CLI commands.

### 3. Interactive Show Command
- Implement a `show` command.
- If running headless, `show` could dump an immediate annotated screenshot.
- If `--annotate` is provided, spawn a local lightweight web server returning the current snapshot with a Javascript overlay to capture bounding boxes, blocking until the user hits "Submit Feedback".
</tasks>

<verification>
## Acceptance Criteria
- [ ] Running `rod-cli highlight "#submit"` visually alters the DOM.
- [ ] Running `video-start` -> `click` -> `video-stop` produces a playable video file of the interaction.
- [ ] Running `rod-cli show --annotate` opens a feedback loop that blocks until user annotation is completed and returned to the CLI output.
</verification>

---
*Plan created: 2026-06-18*
