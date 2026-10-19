package testchrome

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/agenthands/godoll/browser"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
)

func TestStopRemovesProfileAndTempFiles(t *testing.T) {
	l := launcher.New().Headless(true)
	_, stop, err := Launch(l)
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	dir := l.Get(flags.UserDataDir)
	lock, err := os.Readlink(filepath.Join(dir, "SingletonSocket"))
	if err != nil {
		stop()
		t.Fatalf("read chrome lock link in %s: %v", dir, err)
	}
	stop()
	if !strings.HasPrefix(lock, dir+string(filepath.Separator)) {
		os.RemoveAll(filepath.Dir(lock))
		t.Fatalf("chrome lock %s is outside the profile %s", lock, dir)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		os.RemoveAll(dir)
		t.Fatalf("profile %s still exists after stop (stat err %v)", dir, err)
	}
}

// A test binary killed by a timeout never runs stop; the next Launch removes
// what it left, and keeps the dirs of test processes still running.
func TestLaunchRemovesDeadTestProfiles(t *testing.T) {
	exited := exec.Command("true")
	if err := exited.Run(); err != nil {
		t.Fatal(err)
	}
	mk := func(pid int) string {
		d, err := browser.ChromeTempDir(launcher.New())
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, ownerFile), []byte(strconv.Itoa(pid)), 0o600); err != nil {
			t.Fatal(err)
		}
		return d
	}
	dead, live := mk(exited.Process.Pid), mk(os.Getpid())
	defer os.RemoveAll(live)

	MustLaunch(t, launcher.New().Headless(true))

	if _, err := os.Stat(dead); !os.IsNotExist(err) {
		os.RemoveAll(dead)
		t.Errorf("dead test's profile %s still exists (stat err %v)", dead, err)
	}
	if _, err := os.Stat(live); err != nil {
		t.Errorf("live test's profile %s was removed: %v", live, err)
	}
}
