package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLockFileLeavesNoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.port.lock")
	unlock, err := lockFile(path)
	if err != nil {
		t.Fatalf("lockFile: %v", err)
	}
	unlock()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("lock file %s still exists after release (stat err %v)", path, err)
	}
}

// Removing the file on release must not let two holders in at once: a waiter
// that locked the removed file has to retry on the new one.
func TestLockFileStaysExclusiveAcrossRemoval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.port.lock")
	unlockA, err := lockFile(path)
	if err != nil {
		t.Fatalf("A: %v", err)
	}
	gotB := make(chan func())
	go func() {
		u, err := lockFile(path)
		if err != nil {
			t.Errorf("B: %v", err)
		}
		gotB <- u
	}()
	time.Sleep(200 * time.Millisecond) // B is now waiting on A's file
	unlockA()
	unlockB := <-gotB

	gotC := make(chan func())
	go func() {
		u, err := lockFile(path)
		if err != nil {
			t.Errorf("C: %v", err)
		}
		gotC <- u
	}()
	select {
	case u := <-gotC:
		u()
		unlockB()
		t.Fatal("C got the lock while B held it")
	case <-time.After(300 * time.Millisecond):
	}
	unlockB()
	(<-gotC)()
}

func TestTryLockFileSkipsAHeldLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.port.lock")
	unlock, err := lockFile(path)
	if err != nil {
		t.Fatalf("lockFile: %v", err)
	}
	if release, ok, err := tryLockFile(path); err != nil || ok {
		if ok {
			release()
		}
		t.Fatalf("tryLockFile on a held lock: ok=%v err=%v, want ok=false err=nil", ok, err)
	}
	unlock()
	release, ok, err := tryLockFile(path)
	if err != nil || !ok {
		t.Fatalf("tryLockFile on a free lock: ok=%v err=%v", ok, err)
	}
	release()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("lock file %s still exists after release (stat err %v)", path, err)
	}
}
