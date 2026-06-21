# Request Mocking

`rod-cli` allows you to intercept and modify network requests made by the browser. This is extremely useful for bypassing captchas, speeding up page loads, or testing edge cases.

## Routing Requests

You can use `rod-cli route` with a glob pattern to match specific URLs.

### Blocking requests (e.g. tracking scripts, heavy images)
```bash
rod-cli route "**/*.jpg" --status=404
rod-cli route "https://analytics.example.com/**" --status=404
```

### Mocking responses
You can return a mocked JSON body instead of the real backend response:
```bash
rod-cli route "https://api.example.com/user" --body='{"id": 1, "name": "Mock User"}'
```

## Listing Routes
To see all active routes in the current session:
```bash
rod-cli route-list
```

## Removing Routes
To remove a specific route pattern:
```bash
rod-cli unroute "**/*.jpg"
```

To remove all active routes and allow normal traffic again:
```bash
rod-cli unroute
```
