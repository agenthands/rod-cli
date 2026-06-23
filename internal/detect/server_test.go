package detect

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestDetectServer proves the offline server + embed wiring without a browser:
// the root serves the embedded HTML and /detect.js serves the probe script.
func TestDetectServer(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	s.Start()
	defer s.Close()

	url := s.URL()
	if !strings.HasPrefix(url, "http://127.0.0.1:") {
		t.Fatalf("URL() = %q, want loopback http://127.0.0.1:<port>", url)
	}

	// Root serves the embedded HTML shell.
	body := getBody(t, url+"/")
	if !strings.Contains(body, "<title>rod-cli detection harness</title>") {
		t.Errorf("root page missing expected <title>; got:\n%s", body)
	}
	if !strings.Contains(body, `src="/detect.js"`) {
		t.Errorf("root page does not load /detect.js; got:\n%s", body)
	}

	// /detect.js serves the probe script that populates window.__detect.
	js := getBody(t, url+"/detect.js")
	if !strings.Contains(js, "window.__detect") {
		t.Errorf("/detect.js does not reference window.__detect; got:\n%s", js)
	}
}

func getBody(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200", url, resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body of %s failed: %v", url, err)
	}
	return string(b)
}
