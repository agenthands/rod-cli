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
