//go:build windows

package daemon

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// lockFile takes an exclusive, blocking lock on path (created if missing) and
// returns its release func, which removes the file. The lock dies with the
// process, so a crashed client never leaves it held.
func lockFile(path string) (func(), error) {
	release, _, err := takeLock(path, windows.LOCKFILE_EXCLUSIVE_LOCK)
	return release, err
}

// tryLockFile is lockFile without the wait: ok is false, with a nil error, when
// another process holds the lock.
func tryLockFile(path string) (release func(), ok bool, err error) {
	return takeLock(path, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY)
}

func takeLock(path string, flags uint32) (func(), bool, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, false, err
	}
	ol := new(windows.Overlapped)
	if err := windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, ol); err != nil {
		f.Close()
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return func() {
		f.Close()
		// Fails while another waiter has the file open, which keeps its lock
		// valid; the last one out removes it.
		os.Remove(path)
	}, true, nil
}
