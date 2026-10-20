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
	removeDeadTestDirs(filepath.Join(filepath.Dir(dir), "chrome-*"), nil)
	if err := claim(dir); err != nil {
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

// MkdirTemp creates a dir in os.TempDir() named prefix plus a random suffix,
// owned by this test process, after removing the dirs with that prefix whose
// test process has exited and that inUse (nil: none) does not report busy. The
// caller removes it when done.
func MkdirTemp(prefix string, inUse func(dir string) bool) (string, error) {
	removeDeadTestDirs(filepath.Join(os.TempDir(), prefix+"*"), inUse)
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return "", err
	}
	return dir, claim(dir)
}

// claim records this test process as dir's owner; on error dir is removed.
func claim(dir string) error {
	err := os.WriteFile(filepath.Join(dir, ownerFile), []byte(strconv.Itoa(os.Getpid())), 0o600)
	if err != nil {
		os.RemoveAll(dir)
	}
	return err
}

// removeDeadTestDirs removes the test dirs matching pattern whose test process
// has exited, except those inUse reports busy. A test binary killed by a
// timeout never runs its cleanup.
func removeDeadTestDirs(pattern string, inUse func(dir string) bool) {
	owners, _ := filepath.Glob(filepath.Join(pattern, ownerFile))
	for _, f := range owners {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
		if dir := filepath.Dir(f); err == nil && !alive(pid) && (inUse == nil || !inUse(dir)) {
			os.RemoveAll(dir)
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
