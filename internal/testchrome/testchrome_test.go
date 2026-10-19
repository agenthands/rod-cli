package testchrome

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
