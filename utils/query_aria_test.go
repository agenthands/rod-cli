package utils

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// ariaServerHTML is a minimal fixture page. It defines a stub `snapshotEngine`
// whose queryAll() mirrors the production contract used by QueryEleByAria's
// injected JS (types/js.QueryEleByAria) by returning a single DOM node for the
// given CSS selector. This lets us exercise QueryEleByAria end-to-end without
// loading the full (embedded) snapshot engine.
const ariaServerHTML = `<!DOCTYPE html>
<html><head><title>aria</title>
<script>
  window.snapshotEngine = {
    queryAll: function(selector) { return document.querySelector(selector); }
  };
</script>
</head>
<body>
  <button id="go" aria-label="Submit">Submit</button>
</body></html>`

func startAriaServer(t *testing.T) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(ariaServerHTML))
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	t.Cleanup(func() { _ = srv.Close() })
	return "http://" + ln.Addr().String()
}

func newAriaBrowser(t *testing.T) *rod.Browser {
	t.Helper()
	path, _ := launcher.LookPath()
	u := launcher.New().Bin(path).Headless(true).NoSandbox(true).MustLaunch()
	b := rod.New().ControlURL(u).MustConnect()
	t.Cleanup(func() { _ = b.Close() })
	return b
}

func TestQueryEleByAria_Found(t *testing.T) {
	url := startAriaServer(t)
	b := newAriaBrowser(t)

	page, err := b.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		t.Fatalf("create page: %v", err)
	}
	defer page.MustClose()

	if err := page.Navigate(url); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	_ = page.Timeout(5 * time.Second).WaitLoad()
	time.Sleep(150 * time.Millisecond)

	el, err := QueryEleByAria(page, "#go")
	if err != nil {
		t.Fatalf("QueryEleByAria returned error: %v", err)
	}
	if el == nil {
		t.Fatal("expected a matched element")
	}
	text, _ := el.Text()
	if text != "Submit" {
		t.Fatalf("expected element text 'Submit', got %q", text)
	}
}

func TestQueryEleByAria_NoEngineErrors(t *testing.T) {
	// On a page without snapshotEngine defined, the injected JS throws a
	// ReferenceError, so ElementByJS surfaces an error. This exercises the
	// error return path of QueryEleByAria.
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><body><div id="x"></div></body></html>`))
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Close()
	url := "http://" + ln.Addr().String()

	b := newAriaBrowser(t)
	page, err := b.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		t.Fatalf("create page: %v", err)
	}
	defer page.MustClose()
	if err := page.Navigate(url); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	_ = page.Timeout(5 * time.Second).WaitLoad()
	time.Sleep(150 * time.Millisecond)

	_, err = QueryEleByAria(page, "#x")
	if err == nil {
		t.Fatal("expected error when snapshotEngine is undefined")
	}
}
