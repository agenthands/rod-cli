package types

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// ownerFile names the file in a session profile dir that holds the PID of the
// rod-cli process that owns it.
const ownerFile = "owner.pid"

// readCmdline returns a process's command line. A var so tests can fake it.
var readCmdline = func(pid int) (string, error) {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	return strings.ReplaceAll(string(b), "\x00", " "), err
}

// DefaultProfileRoot is where per-session Chrome profiles live unless the
// config names a browserTempDir: on disk in the user cache dir
// (~/.cache/rod-cli/profiles on Linux), never under the working directory.
func DefaultProfileRoot() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "rod-cli", "profiles")
}

// SweepStaleProfiles removes every profile dir under root whose owner is gone:
// no readable owner.pid, a dead PID, or a live PID that is not rod-cli (the PID
// was reused). It covers daemons that never ran their cleanup (kill -9, crash,
// reboot).
func SweepStaleProfiles(root string) {
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		if !ownerAlive(dir) {
			_ = os.RemoveAll(dir)
		}
	}
}

func ownerAlive(dir string) bool {
	b, err := os.ReadFile(filepath.Join(dir, ownerFile))
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 0 || !processAlive(pid) {
		return false
	}
	cmdline, err := readCmdline(pid)
	if err != nil {
		// No /proc (macOS, Windows): a live PID is the best evidence there is.
		return true
	}
	return strings.Contains(cmdline, "rod-cli")
}

func processAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		// FindProcess opens the process there, so success means it exists.
		return true
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, os.ErrPermission)
}
