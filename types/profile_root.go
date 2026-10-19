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
	"time"

	"github.com/agenthands/rod-cli/utils"
	"github.com/go-rod/rod/lib/launcher"
)

// ownerFile names the file in a session profile dir that holds the PID of the
// rod-cli process that owns it.
const ownerFile = "owner.pid"

// chromeTmpFile names the file in a session profile dir that holds the path of
// the session Chrome's temp dir (see browser.ChromeTempDir), which lives
// outside the profile so its socket path stays short.
const chromeTmpFile = "chrome-tmp"

// legacyBrowserTempDir is the old default profile root. Config files generated
// before the move to DefaultProfileRoot still carry it, so it counts as unset.
const legacyBrowserTempDir = "./rod/browser"

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
			removeProfileDir(dir)
		}
	}
}

func ownerAlive(dir string) bool {
	b, err := os.ReadFile(filepath.Join(dir, ownerFile))
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || !processAlive(pid) {
		return false
	}
	cmdline, err := readCmdline(pid)
	if err != nil {
		// No /proc (macOS, Windows): a live PID is the best evidence there is.
		return true
	}
	return strings.Contains(cmdline, "rod-cli")
}

// newSessionProfile creates this session's profile dir under root (the default
// root when empty) and writes owner.pid. The default root is swept first.
func newSessionProfile(root string) (string, error) {
	if root == "" || root == legacyBrowserTempDir {
		root = DefaultProfileRoot()
		SweepStaleProfiles(root)
	}
	dir := filepath.Join(root, utils.RandomString(10))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, os.WriteFile(filepath.Join(dir, ownerFile), []byte(strconv.Itoa(os.Getpid())), 0o600)
}

// removeSessionProfile deletes dir once l's Chrome has exited, since Chrome
// writes into its profile until then. A Chrome still up after 5s is killed.
func removeSessionProfile(dir string, l *launcher.Launcher) {
	waitGone := func() {
		for i := 0; i < 50 && processAlive(l.PID()); i++ {
			time.Sleep(100 * time.Millisecond)
		}
	}
	waitGone()
	if processAlive(l.PID()) {
		l.Kill()
		waitGone()
	}
	_ = os.RemoveAll(dir)
}

// removeProfileDir removes a profile whose Chrome has exited, and that Chrome's
// temp dir, which holds the lock dir and temp files a killed Chrome leaves.
func removeProfileDir(dir string) {
	if b, err := os.ReadFile(filepath.Join(dir, chromeTmpFile)); err == nil {
		if tmp := strings.TrimSpace(string(b)); strings.HasPrefix(filepath.Base(tmp), "chrome-") {
			_ = os.RemoveAll(tmp)
		}
	}
	_ = os.RemoveAll(dir)
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
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
