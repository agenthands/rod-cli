//go:build windows

package daemon

import (
	"os"

	"golang.org/x/sys/windows"
)

// lockFile takes an exclusive, blocking lock on path (created if missing) and
// returns its release func, which removes the file. The lock dies with the
// process, so a crashed client never leaves it held.
func lockFile(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	ol := new(windows.Overlapped)
	if err := windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, ol); err != nil {
		f.Close()
		return nil, err
	}
	return func() {
		f.Close()
		// Fails while another waiter has the file open, which keeps its lock
		// valid; the last one out removes it.
		os.Remove(path)
	}, nil
}
