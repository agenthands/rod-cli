package plugin

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/dop251/goja"
)

// vmGet reads a VM global under the engine lock, for tests that inspect the VM
// while lifecycle hooks may be firing.
func vmGet(e *PluginEngine, name string) goja.Value {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.vm.Get(name)
}

// TestConcurrentHooksAndRunFunc: lifecycle hooks fire on rod's event goroutine
// while RunFunc is called from another; the shared goja VM must be serialized
// (run under -race) and every onLoad must be counted.
func TestConcurrentHooksAndRunFunc(t *testing.T) {
	const n = 5
	engine := NewPluginEngine()
	engine.Init()
	if _, err := engine.vm.RunString(`var loads = 0; function onLoad(){ loads++ } function getLoads(){ return String(loads) }`); err != nil {
		t.Fatal(err)
	}
	page := openIntegPage(t, integServerURL)
	defer page.MustClose()
	engine.BindLifecycle(context.Background(), page, nil)

	done := make(chan error, 1)
	go func() {
		for i := 0; i < n; i++ {
			wait := page.WaitNavigation("load")
			if err := page.Navigate(integServerURL + "/?i=" + strconv.Itoa(i)); err != nil {
				done <- err
				return
			}
			wait()
		}
		done <- nil
	}()

	deadline := time.Now().Add(15 * time.Second)
	got := ""
	for time.Now().Before(deadline) {
		var err error
		if got, err = engine.RunFunc("getLoads"); err != nil {
			t.Fatal(err)
		}
		if got == strconv.Itoa(n) {
			break
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got != strconv.Itoa(n) {
		t.Fatalf("onLoad count = %s, want %d", got, n)
	}
}
