package actions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/agenthands/rod-cli/internal/testchrome"
	"github.com/agenthands/rod-cli/types"
	"github.com/go-rod/rod/lib/launcher"
)

// TestPluginLoadTwiceSingleDispatch: loading a plugin twice on the same page
// must not register a second set of lifecycle listeners, so one navigation
// fires onLoad exactly once.
func TestPluginLoadTwiceSingleDispatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body>ok</body></html>`))
	}))
	defer srv.Close()

	u := testchrome.MustLaunch(t, launcher.New().Headless(true))
	ctx := types.NewContext(context.Background(), types.Config{CDPEndpoint: u})
	page, err := ctx.EnsurePage()
	if err != nil {
		t.Fatalf("EnsurePage: %v", err)
	}
	defer ctx.CloseBrowser()

	pluginPath := filepath.Join(t.TempDir(), "count.js")
	if err := writeFile(pluginPath, `var loads = 0; function onLoad(){ loads++ } function getLoads(){ return String(loads) }`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := PluginLoad(ctx, pluginPath); err != nil {
			t.Fatalf("PluginLoad #%d: %v", i+1, err)
		}
	}

	wait := page.WaitNavigation("load")
	if err := page.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	wait()

	got := ""
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline) && got != "1"; time.Sleep(20 * time.Millisecond) {
		got, _ = PluginRun(ctx, "getLoads")
	}
	time.Sleep(500 * time.Millisecond) // let any duplicate dispatch land
	if got, _ = PluginRun(ctx, "getLoads"); got != "1" {
		t.Fatalf("onLoad count after one navigation = %s, want 1", got)
	}
}
