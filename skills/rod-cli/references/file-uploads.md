# File Uploads and Drag-and-Drop

`rod-cli` provides robust primitives for dealing with file inputs, which are notoriously tricky in web automation.

## Standard File Uploads
For a standard `<input type="file">`, you can use the `upload` command. You must provide the element reference and the absolute path to the file on the local filesystem.

```bash
# Upload a single file
rod-cli upload s1e22 "/tmp/resume.pdf"

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
If you need to drag one DOM element and drop it onto another DOM element (e.g., reordering a list, or moving a card on a Kanban board), use the `drag` command.

```bash
# Drag element s1e10 and drop it onto element s1e15
rod-cli drag s1e10 s1e15
```
