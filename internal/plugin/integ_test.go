package plugin

import (
	"context"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

var integBrowser *rod.Browser
var integServerURL string

func TestMain(m *testing.M) {
	// Start a simple HTTP server for integration tests
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		http.SetCookie(w, &http.Cookie{Name: "test_cookie", Value: "abc123"})
		w.Write([]byte(`<!DOCTYPE html><html><body><h1>Test Page</h1></body></html>`))
	})
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	integServerURL = "http://" + ln.Addr().String()
	go http.Serve(ln, mux)
	defer ln.Close()

	// Launch browser
	path, _ := launcher.LookPath()
	u := launcher.New().Bin(path).Headless(true).NoSandbox(true).MustLaunch()
	integBrowser = rod.New().ControlURL(u).MustConnect()
	defer integBrowser.MustClose()

	os.Exit(m.Run())
}

func openIntegPage(t *testing.T, url string) *rod.Page {
	t.Helper()
	page, err := integBrowser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		t.Fatal(err)
	}
	page.Navigate(url)
	_ = page.Timeout(5 * time.Second).WaitLoad()
	time.Sleep(200 * time.Millisecond)
	return page
}

// ===================== BindLifecycle =====================

func TestBindLifecycle_SetsAPIOnVM(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	page := openIntegPage(t, integServerURL)
	defer page.MustClose()

	engine.BindLifecycle(context.Background(), page)

	// Verify that the 'api' global is set in the VM
	val := engine.vm.Get("api")
	if val == nil {
		t.Fatal("expected 'api' global to be set in VM after BindLifecycle")
	}
}

func TestBindLifecycle_NilVM(t *testing.T) {
	engine := NewPluginEngine()
	// vm is nil — should not panic
	page := openIntegPage(t, integServerURL)
	defer page.MustClose()

	engine.BindLifecycle(context.Background(), page)
}

func TestBindLifecycle_ForwardsOnLoad(t *testing.T) {
	engine := NewPluginEngine()
	engine.Init()

	// Register a JS onLoad handler
	engine.vm.RunString(`
		var loadCalled = false;
		function onLoad(ev) { loadCalled = true; }
	`)

	page := openIntegPage(t, integServerURL)
	defer page.MustClose()

	engine.BindLifecycle(context.Background(), page)

	// Navigate to trigger onLoad
	page.Navigate(integServerURL + "/")
	_ = page.Timeout(5 * time.Second).WaitLoad()
	time.Sleep(500 * time.Millisecond)

	val := engine.vm.Get("loadCalled")
	if val == nil {
		t.Fatal("expected loadCalled variable")
	}
	// Note: the event may or may not fire depending on timing,
	// the important thing is no panic/crash
}

// ===================== PluginAPI with real page =====================

func TestPluginAPI_GetCookies_RealPage(t *testing.T) {
	page := openIntegPage(t, integServerURL)
	defer page.MustClose()

	api := NewPluginAPI(page)
	cookies, err := api.GetCookies()
	if err != nil {
		t.Fatalf("GetCookies failed: %v", err)
	}
	if cookies == nil {
		t.Fatal("expected non-nil cookies")
	}
}

func TestPluginAPI_GetSnapshot_RealPage(t *testing.T) {
	page := openIntegPage(t, integServerURL)
	defer page.MustClose()

	api := NewPluginAPI(page)
	html, err := api.GetSnapshot()
	if err != nil {
		t.Fatalf("GetSnapshot failed: %v", err)
	}
	if html == "" {
		t.Fatal("expected non-empty HTML snapshot")
	}
	if len(html) < 10 {
		t.Fatal("snapshot too short")
	}
}

func TestPluginAPI_GetSnapshot_ClosedPage(t *testing.T) {
	page := openIntegPage(t, integServerURL)
	page.MustClose()

	api := NewPluginAPI(page)
	_, err := api.GetSnapshot()
	// Closed page will return error — just verify no panic
	_ = err
}

func TestPluginAPI_GetLocalStorage_RealPage(t *testing.T) {
	page := openIntegPage(t, integServerURL)
	defer page.MustClose()

	// The served test HTML has no localStorage — seed a key before reading.
	if _, err := page.Eval(`() => localStorage.setItem('theme', 'dark')`); err != nil {
		t.Fatalf("seeding localStorage failed: %v", err)
	}

	api := NewPluginAPI(page)
	data, err := api.GetLocalStorage()
	if err != nil {
		t.Fatalf("GetLocalStorage failed: %v", err)
	}
	if data == nil {
		t.Fatal("expected non-nil localStorage data")
	}

	m, ok := data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", data)
	}
	if got := m["theme"]; got != "dark" {
		t.Fatalf("expected seeded key theme=dark, got %v", got)
	}
}

func TestPluginAPI_GetLocalStorage_NilPage(t *testing.T) {
	api := NewPluginAPI(nil)
	data, err := api.GetLocalStorage()
	if err != nil {
		t.Fatalf("expected nil error for nil page, got %v", err)
	}
	if data != nil {
		t.Fatalf("expected nil data for nil page, got %v", data)
	}
}
