//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

// lockFile takes an exclusive, blocking lock on path (created if missing) and
// returns its release func, which removes the file. The lock dies with the
// process, so a crashed client never leaves it held.
//
// Release unlinks the file while still holding the lock, so a waiter may wake
// holding a file no longer at path; it then retries on the current one.
func lockFile(path string) (func(), error) {
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return nil, err
		}
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
			f.Close()
			return nil, err
		}
		held, errHeld := f.Stat()
		cur, errCur := os.Stat(path)
		if errHeld == nil && errCur == nil && os.SameFile(held, cur) {
			return func() {
				os.Remove(path)
				f.Close()
			}, nil
		}
		f.Close()
	}
}
