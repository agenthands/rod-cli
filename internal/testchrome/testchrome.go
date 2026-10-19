// Package testchrome launches Chrome for tests. Every launch gets its own temp
// profile dir, and stopping it ends that Chrome and removes the profile, so test
// runs leave nothing in /tmp/rod/user-data or TMPDIR.
package testchrome

import (
	"os"
	"testing"

	"github.com/go-rod/rod/lib/launcher"
)

// Launch starts l on a fresh temp profile and returns its control URL and a
// stop func that kills that Chrome, waits for it to exit and removes the
// profile. Chrome's TMPDIR is the profile, so its lock dir and temp files go
// with it.
func Launch(l *launcher.Launcher) (string, func(), error) {
	dir, err := os.MkdirTemp("", "rod-cli-test-profile-")
	if err != nil {
		return "", nil, err
	}
	u, err := l.UserDataDir(dir).Env(append(os.Environ(), "TMPDIR="+dir)...).Launch()
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
