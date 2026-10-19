// Package testchrome launches Chrome for tests. Every launch gets one short temp
// dir that is both its profile and its TMPDIR, and stopping it ends that Chrome
// and removes the dir, so test runs leave nothing in /tmp/rod/user-data or
// TMPDIR.
package testchrome

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/agenthands/godoll/browser"
	"github.com/go-rod/rod/lib/launcher"
)

// ownerFile holds the PID of the test process that launched the Chrome using
// the dir.
const ownerFile = "rod-cli-test.pid"

// Launch starts l on a fresh dir that is its profile and TMPDIR, and returns its
// control URL and a stop func that kills that Chrome, waits for it to exit and
// removes the dir. It first removes dirs left by test processes that died
// before their stop ran.
func Launch(l *launcher.Launcher) (string, func(), error) {
	dir, err := browser.ChromeTempDir(l)
	if err != nil {
		return "", nil, err
	}
	removeDeadTestDirs(filepath.Dir(dir))
	if err := os.WriteFile(filepath.Join(dir, ownerFile), []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		os.RemoveAll(dir)
		return "", nil, err
	}
	u, err := l.UserDataDir(dir).Launch()
	if err != nil {
		os.RemoveAll(dir)
		return "", nil, err
	}
	return u, func() {
		if p, err := os.FindProcess(l.PID()); err == nil {
			_ = p.Kill()
		}
		l.Cleanup()
	}, nil
}

// removeDeadTestDirs removes the test Chrome dirs under root whose test process
// has exited. A test binary killed by a timeout never runs stop.
func removeDeadTestDirs(root string) {
	owners, _ := filepath.Glob(filepath.Join(root, "chrome-*", ownerFile))
	for _, f := range owners {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
		if err == nil && !alive(pid) {
			os.RemoveAll(filepath.Dir(f))
		}
	}
}

func alive(pid int) bool {
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

// MustLaunch is Launch for one test: it fails the test on a launch error and
// stops Chrome when the test ends.
func MustLaunch(t testing.TB, l *launcher.Launcher) string {
	t.Helper()
	u, stop, err := Launch(l)
	if err != nil {
		t.Fatalf("launch chrome: %v", err)
	}
	t.Cleanup(stop)
	return u
}
