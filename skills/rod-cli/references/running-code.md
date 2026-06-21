# Running Code in the Browser

You can evaluate custom JavaScript code directly in the browser's context using the `eval` command.

## Basic Evaluation
If you omit a target, the script is executed in the context of the global `window`.

```bash
rod-cli eval "document.title"
rod-cli eval "performance.timing.navigationStart"
```

## Targeting Specific Elements
If you want to evaluate code against a specific element you found in the snapshot, provide the element reference (e.g. `e5`) as the second argument. The script is evaluated with `this` bound to the DOM element.

```bash
# Get text content of an element
rod-cli eval "() => this.textContent" e5

# Modify an element's style
rod-cli eval "() => { this.style.border = '2px solid red' }" e2

# Trigger a custom click event if normal clicking fails
rod-cli eval "() => this.click()" e8
```

## Extracting Data

You can use standard DOM APIs to extract complex data structures and pipe them into your agent via `--raw`.

```bash
rod-cli --raw eval "JSON.stringify([...document.querySelectorAll('a')].map(a => a.href))"
```
