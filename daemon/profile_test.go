//go:build !windows

package daemon

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// spawnProfileDaemon builds rod-cli, points XDG_CACHE_HOME at a temp dir,
// starts a headless daemon for a new session and navigates it to a fixture
// page so Chrome (and its profile dir) exists. It returns the session, the
// daemon PID and the profile root.
func spawnProfileDaemon(t *testing.T, bin, prefix string) (string, int, string) {
	t.Helper()
	cache := os.Getenv("XDG_CACHE_HOME")
	session := uniqueSession(prefix)
	if err := EnsureDaemon(session, bin, []string{"--headless"}, nil); err != nil {
		t.Fatalf("EnsureDaemon: %v", err)
	}
	t.Cleanup(func() { _, _ = ClientExecute(session, Request{Command: "close"}) })
	srv := newPageServer()
	t.Cleanup(srv.Close)
	if _, err := ClientExecute(session, Request{Command: "goto", Args: map[string]string{"url": srv.URL}}); err != nil {
		t.Fatalf("goto: %v", err)
	}
	out, _ := exec.Command("pgrep", "-f", "--", "--session "+session+" .*daemon").Output()
	pids := strings.Fields(string(out))
	if len(pids) != 1 {
		t.Fatalf("want 1 daemon for %s, got %v", session, pids)
	}
	pid, _ := strconv.Atoi(pids[0])
	return session, pid, filepath.Join(cache, "rod-cli", "profiles")
}

func buildProfileTestBin(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping browser/spawn test in short mode")
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	bin := filepath.Join(t.TempDir(), "rod-cli")
	if out, err := exec.Command("go", "build", "-o", bin, "github.com/agenthands/rod-cli").CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	// The spawned daemon (no --config) writes a default config in its cwd.
	t.Cleanup(func() { os.Remove("rod-cli.yaml") })
	return bin
}

func profileDirs(t *testing.T, root string) []string {
	t.Helper()
	entries, _ := os.ReadDir(root)
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(root, e.Name()))
		}
	}
	return dirs
}

func waitExit(t *testing.T, pid int) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if syscall.Kill(pid, 0) != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("daemon %d still alive 10s after the signal", pid)
}

// chromeTmp returns the session Chrome's temp dir recorded in profile, after
// checking that Chrome's lock lives in it and not in the shared TMPDIR.
func chromeTmp(t *testing.T, profile string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(profile, "chrome-tmp"))
	if err != nil {
		t.Fatalf("profile %s records no chrome temp dir: %v", profile, err)
	}
	tmp := strings.TrimSpace(string(b))
	lock, err := os.Readlink(filepath.Join(profile, "SingletonSocket"))
	if err != nil {
		t.Fatalf("read chrome lock link in %s: %v", profile, err)
	}
	if !strings.HasPrefix(lock, tmp+string(filepath.Separator)) {
		t.Fatalf("chrome lock %s is outside the session temp dir %s", lock, tmp)
	}
	return tmp
}

func assertGone(t *testing.T, path, when string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		os.RemoveAll(path)
		t.Fatalf("%s survived %s (stat err %v)", path, when, err)
	}
}

func TestDaemonSIGTERMRemovesProfile(t *testing.T) {
	bin := buildProfileTestBin(t)
	_, pid, root := spawnProfileDaemon(t, bin, "sigterm")
	dirs := profileDirs(t, root)
	if len(dirs) != 1 {
		t.Fatalf("want 1 profile dir under %s, got %v", root, dirs)
	}
	tmp := chromeTmp(t, dirs[0])
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	waitExit(t, pid)
	assertGone(t, dirs[0], "SIGTERM")
	assertGone(t, tmp, "SIGTERM")
}

func TestDaemonKillProfileSweptOnNextStart(t *testing.T) {
	bin := buildProfileTestBin(t)
	_, pid, root := spawnProfileDaemon(t, bin, "killed")
	dirs := profileDirs(t, root)
	if len(dirs) != 1 {
		t.Fatalf("want 1 profile dir under %s, got %v", root, dirs)
	}
	tmp := chromeTmp(t, dirs[0])
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	waitExit(t, pid)
	if _, err := os.Stat(dirs[0]); err != nil {
		t.Fatalf("SIGKILL cannot clean up, yet profile %s is gone: %v", dirs[0], err)
	}

	spawnProfileDaemon(t, bin, "next")
	assertGone(t, dirs[0], "the next start's sweep")
	assertGone(t, tmp, "the next start's sweep")
}
