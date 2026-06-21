# Inspecting Element Attributes

To retrieve element attributes, use the `rod-cli eval` command along with a target element reference from the snapshot.

## Getting standard attributes
```bash
# Get the 'id' attribute of the element with reference 'e5'
rod-cli eval "() => this.id" e5

# Get the 'href' attribute of a link
rod-cli eval "() => this.href" e12

# Get the 'src' of an image
rod-cli eval "() => this.src" e3
```

## Getting custom attributes
For `data-*` attributes or any other custom attribute, use the `getAttribute` function:

```bash
# Get a 'data-testid'
rod-cli eval "() => this.getAttribute('data-testid')" e5

# Get 'aria-expanded'
rod-cli eval "() => this.getAttribute('aria-expanded')" e2
```

## Raw Output
When passing the result to another script, use the `--raw` flag so you only get the value without the surrounding snapshot text:
```bash
rod-cli --raw eval "el => el.href" e12
```
