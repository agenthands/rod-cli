//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

// lockFile takes an exclusive, blocking lock on path (created if missing) and
// returns its release func. The lock dies with the process, so a crashed
// client never leaves it held.
func lockFile(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return func() { f.Close() }, nil
}
