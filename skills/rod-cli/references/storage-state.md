# Storage State (Cookies & Storage)

`rod-cli` provides direct access to the browser's storage mechanisms, allowing you to extract session tokens or inject authentication state directly into the daemon.

## Cookies

Read cookies from the active session:
```bash
rod-cli cookie-get
rod-cli cookie-get session_id
```

Clear all cookies:
```bash
rod-cli cookie-clear
```

## LocalStorage

Interact with `window.localStorage`:
```bash
# Get a specific key
rod-cli localstorage-get theme

# Set a key
rod-cli localstorage-set theme dark

# Delete a key
rod-cli localstorage-delete theme

# Clear all LocalStorage
rod-cli localstorage-clear
```

## SessionStorage

Interact with `window.sessionStorage`:
```bash
# Get a specific key
rod-cli sessionstorage-get step

# Set a key
rod-cli sessionstorage-set step 3

# Delete a key
rod-cli sessionstorage-delete step

# Clear all SessionStorage
rod-cli sessionstorage-clear
```
