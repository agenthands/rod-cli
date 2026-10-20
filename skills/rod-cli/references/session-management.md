# Browser Session Management

`rod-cli` runs one background daemon per session, and the daemon keeps one Chrome open across commands instead of launching a browser per command.

## The Default Session
Running `rod-cli open https://example.com` automatically spawns the default background daemon if it isn't running, and uses it. Future commands like `rod-cli click "#submit"` (or `rod-cli click s1e5` using a snapshot ref) implicitly connect to this same daemon.

## Multiplexing Named Sessions

For workflows that need isolated browsers (like testing a chat application with two users at once), use the `--session` or `-s` flag. Each session has its own daemon, Chrome and profile directory. Session names use letters, digits, `.`, `_` or `-` (1 to 64 characters, not `.` or `..`); any other name fails with `invalid session name`.

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
To see all background daemons currently running (it also deletes the port files of daemons that no longer answer):
```bash
rod-cli sessions
```

## Terminating Sessions
Close each session when done to free RAM. `close` shuts down the session's Chrome and daemon and deletes its profile before it returns.
```bash
rod-cli close
rod-cli -s=admin close
```

A session also shuts down by itself after 15 minutes with no command, and on SIGTERM, SIGINT or SIGHUP; each of these deletes the profile too.

## Session files
- Profile: one directory per session under `~/.cache/rod-cli/profiles` on Linux (Go's `os.UserCacheDir()`/`rod-cli/profiles`), unless the config sets `browserTempDir` (the old value `./rod/browser` counts as unset). It holds `owner.pid` (the daemon's PID) and `chrome-tmp` (the path of the session Chrome's temp dir). A session started with `--cdp-endpoint` drives that browser and has no profile.
- Chrome temp dir: `$TMPDIR/chrome-XXXX` (`/tmp/chrome-XXXX` when `TMPDIR` is longer than 40 characters), used as Chrome's `TMPDIR`. It is deleted together with the profile.
- Port file: `$TMPDIR/rod-cli-<session>.port`, plus `<port file>.lock` while a daemon is starting.
- Daemon log: `$TMPDIR/rod-cli-daemon.log`, shared by all sessions and started over once it exceeds 1 MiB. Read it when a daemon fails to start.

A daemon killed with `kill -9` leaves its profile, Chrome temp dir and port file behind. The next daemon start deletes port files of dead daemons. The next session that launches its Chrome under the default profile root deletes every profile there whose `owner.pid` is missing, names a dead PID, or names a PID now used by a process that is not rod-cli; a profile under a custom `browserTempDir` is never swept.
