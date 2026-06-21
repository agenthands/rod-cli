# Browser Session Management

`rod-cli` is fundamentally built around a persistent background daemon architecture. It does NOT boot a new browser for every command, which would be incredibly slow. Instead, it re-uses sessions.

## The Default Session
If you just run `rod-cli open https://example.com`, it automatically spawns the default background daemon if it isn't running, and uses it. Future commands like `rod-cli click e5` implicitly connect to this same daemon.

## Multiplexing Named Sessions

For complex workflows (like testing a chat application with two users at once), you can create isolated browser instances using the `--session` or `-s` flag.

```bash
# Start an admin session
rod-cli -s=admin open https://example.com/login

# Start a guest session
rod-cli -s=guest open https://example.com/guest

# Interact with the specific sessions
rod-cli -s=admin type "#username" "admin"
rod-cli -s=admin click "#submit"
rod-cli -s=guest click "#join"
```

## Listing Sessions
To see all background daemons currently running:
```bash
rod-cli sessions
```

## Terminating Sessions
To free up RAM, you must close your session when done.
```bash
rod-cli close
rod-cli -s=admin close
```


