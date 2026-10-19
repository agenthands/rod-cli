//go:build !windows

package daemon

import (
	"errors"
	"os"
	"syscall"
)

// lockFile takes an exclusive, blocking lock on path (created if missing) and
// returns its release func, which removes the file. The lock dies with the
// process, so a crashed client never leaves it held.
func lockFile(path string) (func(), error) {
	release, _, err := takeLock(path, true)
	return release, err
}

// tryLockFile is lockFile without the wait: ok is false, with a nil error, when
// another process holds the lock.
func tryLockFile(path string) (release func(), ok bool, err error) {
	return takeLock(path, false)
}

// takeLock locks path. Release unlinks the file while still holding the lock,
// so a waiter may wake holding a file no longer at path; it then retries on the
// current one.
func takeLock(path string, wait bool) (func(), bool, error) {
	how := syscall.LOCK_EX
	if !wait {
		how |= syscall.LOCK_NB
	}
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return nil, false, err
		}
		if err := syscall.Flock(int(f.Fd()), how); err != nil {
			f.Close()
			if errors.Is(err, syscall.EWOULDBLOCK) {
				return nil, false, nil
			}
			return nil, false, err
		}
		held, errHeld := f.Stat()
		cur, errCur := os.Stat(path)
		if errHeld == nil && errCur == nil && os.SameFile(held, cur) {
			return func() {
				os.Remove(path)
				f.Close()
			}, true, nil
		}
		f.Close()
	}
}
