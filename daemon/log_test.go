package daemon

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenDaemonLogStartsOverPastTheCap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rod-cli-daemon.log")
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), daemonLogCap+1), 0600); err != nil {
		t.Fatal(err)
	}
	f, err := openDaemonLog(path)
	if err != nil {
		t.Fatalf("openDaemonLog: %v", err)
	}
	f.WriteString("new\n")
	f.Close()
	if b, _ := os.ReadFile(path); string(b) != "new\n" {
		t.Fatalf("log over the cap was kept: %d bytes", len(b))
	}

	f, err = openDaemonLog(path)
	if err != nil {
		t.Fatalf("openDaemonLog: %v", err)
	}
	f.WriteString("more\n")
	f.Close()
	if b, _ := os.ReadFile(path); string(b) != "new\nmore\n" {
		t.Fatalf("log under the cap was not appended to: %q", b)
	}
}
