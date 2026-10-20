# Storage State (Cookies & Storage)

`rod-cli` reads and writes the browser's cookies and web storage, to extract session tokens or inject authentication state into a session.

## Cookies

Read cookies from the active session (all, or only the cookie with that exact name; a name with no match is an error):
```bash
rod-cli cookie-get
rod-cli cookie-get session_id
```

Set and delete a cookie (both apply to the current page's URL, so open the site first):
```bash
rod-cli cookie-set session_id abc123
rod-cli cookie-delete session_id
```

Clear all cookies:
```bash
rod-cli cookie-clear
```

## LocalStorage

Interact with `window.localStorage`:
```bash
# Get a specific key (omit the key for all, as JSON)
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
# Get a specific key (omit the key for all, as JSON)
rod-cli sessionstorage-get step

# Set a key
rod-cli sessionstorage-set step 3

# Delete a key
rod-cli sessionstorage-delete step

# Clear all SessionStorage
rod-cli sessionstorage-clear
```

## Saving and restoring state

`state-save` writes the session's cookies to a JSON file; `state-load` reads that file back into a session (both require the path). localStorage and sessionStorage are not included.
```bash
rod-cli state-save /tmp/auth.json
rod-cli close
rod-cli goto https://example.com
rod-cli state-load /tmp/auth.json
rod-cli reload
```
