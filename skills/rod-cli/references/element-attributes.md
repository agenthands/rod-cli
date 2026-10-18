# Inspecting Element Attributes

To retrieve element attributes, use the `rod-cli eval` command along with a target element reference from the snapshot. This is a **read**, not an interaction — `eval` is the right tool here because you're extracting data (an attribute value) rather than acting on the page, which is exactly what `eval` is reserved for; use `click`/`fill`/`type`/etc. instead whenever you need to actually interact with an element.

## Getting standard attributes
```bash
# Get the 'id' attribute of the element with reference 's1e5'
rod-cli eval "() => this.id" s1e5

# Get the 'href' attribute of a link
rod-cli eval "() => this.href" s1e12

# Get the 'src' of an image
rod-cli eval "() => this.src" s1e3
```

## Getting custom attributes
For `data-*` attributes or any other custom attribute, use the `getAttribute` function:

```bash
# Get a 'data-testid'
rod-cli eval "() => this.getAttribute('data-testid')" s1e5

# Get 'aria-expanded'
rod-cli eval "() => this.getAttribute('aria-expanded')" s1e2
```

## Extracting just the value
eval prints the bare value. With --json, a JSON object/array value prints as-is; any other value prints as {"result": "<value>"}.
```bash
rod-cli eval "() => this.href" s1e12
```
