# Inspecting Element Attributes

To read an element attribute, run `rod-cli eval` with the element's ref from the snapshot. This is a **read**, not an interaction: `eval` is reserved for extracting data such as an attribute value; to interact with an element, use `click`/`fill`/`type`/etc.

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
