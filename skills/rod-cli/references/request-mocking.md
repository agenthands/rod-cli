# Request Mocking

`rod-cli` allows you to intercept and modify network requests made by the browser. This is extremely useful for bypassing captchas, speeding up page loads, or testing edge cases.

## Routing Requests

You can use `rod-cli route` with a glob pattern to match specific URLs. Every route answers matching requests with a mocked `200` response (`Content-Type: text/plain`) containing the `--body` value; there is no flag to set a different status code. The `--body` flag must come **before** the pattern — flags placed after positional arguments are silently ignored.

### Blocking requests (e.g. tracking scripts, heavy images)
Omit `--body` to serve an empty response — the real request never reaches the network:
```bash
rod-cli route "**/*.jpg"
rod-cli route "https://analytics.example.com/**"
```

### Mocking responses
You can return a mocked JSON body instead of the real backend response:
```bash
rod-cli route --body='{"id": 1, "name": "Mock User"}' "https://api.example.com/user"
```

## Listing Routes
To see all active routes in the current session:
```bash
rod-cli route-list
```

## Removing Routes
Routes are removed one at a time, by their exact pattern:
```bash
rod-cli unroute "**/*.jpg"
```

There is no remove-all form — running `rod-cli unroute` without a pattern leaves existing routes active. To restore normal traffic, run `rod-cli route-list` and `unroute` each listed pattern.
