package daemon

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/agenthands/rod-cli/types"
)

// startBareServer runs StartServer with an empty Config (no browser) in a
// private TMPDIR and returns the session and the port its port file names.
// Never send it `close`: that would exit the test process.
func startBareServer(t *testing.T) (session, port string) {
	t.Helper()
	privateTmp(t)
	session = uniqueSession("bare")
	go func() { _ = StartServer(session, types.NewContext(context.Background(), types.Config{})) }()
	for i := 0; i < 200; i++ {
		if _, err := ClientExecute(session, Request{Command: "ping"}); err == nil {
			b, _ := os.ReadFile(portFilePath(session))
			return session, strings.TrimSpace(string(b))
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("server did not become ready")
	return
}

func TestExecuteRejectsForeignRequests(t *testing.T) {
	session, port := startBareServer(t)
	url := "http://127.0.0.1:" + port + "/execute"
	body := `{"command":"bogus-cmd"}`

	cases := []struct {
		name  string
		setup func(r *http.Request)
	}{
		{"origin", func(r *http.Request) {
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Origin", "http://evil.example")
		}},
		{"text/plain", func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }},
		{"no content-type", func(r *http.Request) {}},
		{"host", func(r *http.Request) {
			r.Header.Set("Content-Type", "application/json")
			r.Host = "evil.example:" + port
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, _ := http.NewRequest("POST", url, strings.NewReader(body))
			c.setup(r)
			resp, err := http.DefaultClient.Do(r)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			out, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusForbidden || strings.Contains(string(out), "unknown command") {
				t.Fatalf("status %d body %s: want 403 before dispatch", resp.StatusCode, out)
			}
		})
	}

	// The real client still gets through to dispatch.
	if _, err := ClientExecute(session, Request{Command: "bogus-cmd"}); err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("ClientExecute: err = %v, want unknown command", err)
	}
}
