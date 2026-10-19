package types

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

func TestDefaultProfileRootIsUserCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/tmp/rod-cli-xdg-test")
	got := DefaultProfileRoot()
	want := filepath.Join("/tmp/rod-cli-xdg-test", "rod-cli", "profiles")
	if !filepath.IsAbs(got) || got != want {
		t.Fatalf("DefaultProfileRoot() = %q, want %q", got, want)
	}
}

func TestSweepRemovesDeadOwnerProfiles(t *testing.T) {
	root := t.TempDir()
	mk := func(name string, pid int) string {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if pid > 0 {
			if err := os.WriteFile(filepath.Join(dir, "owner.pid"), []byte(strconv.Itoa(pid)), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return dir
	}

	exited := exec.Command("true")
	if err := exited.Run(); err != nil {
		t.Fatal(err)
	}
	sleeper := exec.Command("sleep", "30")
	if err := sleeper.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sleeper.Process.Kill(); _ = sleeper.Wait() })

	orig := readCmdline
	readCmdline = func(pid int) (string, error) {
		if pid == os.Getpid() {
			return "/usr/local/bin/rod-cli --session s daemon", nil
		}
		return orig(pid)
	}
	defer func() { readCmdline = orig }()

	noOwner := mk("no-owner", 0)
	dead := mk("dead", exited.Process.Pid)
	notRod := mk("not-rod", sleeper.Process.Pid)
	mine := mk("mine", os.Getpid())

	SweepStaleProfiles(root)

	for _, gone := range []string{noOwner, dead, notRod} {
		if _, err := os.Stat(gone); !os.IsNotExist(err) {
			t.Errorf("%s still exists (stat err %v), want removed", filepath.Base(gone), err)
		}
	}
	if _, err := os.Stat(mine); err != nil {
		t.Errorf("live rod-cli profile removed: %v", err)
	}
}

func TestSweepRemovesStaleProfileTempDir(t *testing.T) {
	root := t.TempDir()
	tmp := filepath.Join(t.TempDir(), "chrome-123")
	notChrome := filepath.Join(t.TempDir(), "keep")
	stale, odd := filepath.Join(root, "stale"), filepath.Join(root, "odd")
	for _, d := range []string{tmp, notChrome, stale, odd} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for dir, target := range map[string]string{stale: tmp, odd: notChrome} {
		if err := os.WriteFile(filepath.Join(dir, chromeTmpFile), []byte(target), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	SweepStaleProfiles(root)

	for _, p := range []string{stale, tmp, odd} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s still exists after sweep (stat err %v)", p, err)
		}
	}
	if _, err := os.Stat(notChrome); err != nil {
		t.Errorf("sweep removed %s, which is not a Chrome temp dir: %v", notChrome, err)
	}
}
