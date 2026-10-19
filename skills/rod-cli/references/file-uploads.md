# File Uploads and Drag-and-Drop

`rod-cli` provides robust primitives for dealing with file inputs, which are notoriously tricky in web automation.

## Standard File Uploads
For a standard `<input type="file">`, use the `upload` command. `upload`, `drag` and `drop` each take a CSS selector or a snapshot ref, plus (for `upload`/`drop`) the absolute path to the file on the local filesystem.

```bash
# Upload a single file (by snapshot ref)
rod-cli upload s1e22 "/tmp/resume.pdf"

# The same with a CSS selector
rod-cli upload "input[type=file]" "/tmp/resume.pdf"

# Upload multiple files (input must have `multiple`): pass each path as its own argument
rod-cli upload s1e22 "/tmp/image1.png" "/tmp/image2.png"
```

## Drag and Drop Interfaces
Modern web applications often use "Dropzones" instead of standard file inputs. To simulate a user physically dragging a file from their desktop onto a DOM element, use the `drop` command.

```bash
# Drop a file directly onto a Dropzone element
rod-cli drop --path="/tmp/avatar.jpg" s1e45
```

## Element-to-Element Dragging
To drag one DOM element and drop it onto another DOM element (e.g., reordering a list, or moving a card on a Kanban board), use the `drag` command.

```bash
# Drag element s1e10 and drop it onto element s1e15
rod-cli drag s1e10 s1e15
```
