# File Uploads and Drag-and-Drop

`rod-cli` sets files on file inputs and drags elements onto other elements.

## Standard File Uploads
For a standard `<input type="file">`, use the `upload` command. `upload`, `drag` and `drop` each take a CSS selector or a snapshot ref, plus (for `upload`/`drop`) the absolute path to the file on the local filesystem; a relative path resolves against the directory of the command that started the session's daemon.

```bash
# Upload a single file (by snapshot ref)
rod-cli upload s1e22 "/tmp/resume.pdf"

# The same with a CSS selector
rod-cli upload "input[type=file]" "/tmp/resume.pdf"

# Upload multiple files (input must have `multiple`): pass each path as its own argument
rod-cli upload s1e22 "/tmp/image1.png" "/tmp/image2.png"
```

## Drag and Drop Interfaces
For a "Dropzone", target the zone's own `<input type="file">` (often hidden) with `drop`. `drop` sets one file on that input, like `upload` with one path; it does not fire drag or drop events, and on any element that is not a file input it fails.

```bash
# Set a file on the Dropzone's file input
rod-cli drop --path="/tmp/avatar.jpg" s1e45
```

## Element-to-Element Dragging
To drag one DOM element and drop it onto another DOM element (e.g., reordering a list, or moving a card on a Kanban board), use the `drag` command.

```bash
# Drag element s1e10 and drop it onto element s1e15
rod-cli drag s1e10 s1e15
```
