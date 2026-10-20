package daemon

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/agenthands/rod-cli/types"
)

// startBareServer runs StartServer with an empty Config (no browser) and
// returns the session and the port its port file names. The caller sets a
// private TMPDIR first. Never send it `close`: that would exit the test process.
func startBareServer(t *testing.T) (session, port string) {
	t.Helper()
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
	privateTmp(t)
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

func TestClientRefusesNonNumericPortFile(t *testing.T) {
	privateTmp(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		io.WriteString(w, `{"result":"pong"}`)
	}))
	defer srv.Close()
	local := strings.TrimPrefix(srv.URL, "http://")

	for _, content := range []string{"1@evil.example:80", "1@" + local, local, "0", "70000", " 1"} {
		session := uniqueSession("evilport")
		if err := os.WriteFile(portFilePath(session), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := ClientExecute(session, Request{Command: "ping"})
		if err == nil || !strings.Contains(err.Error(), "invalid port file") {
			t.Errorf("port file %q: err = %v, want invalid port file", content, err)
		}
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("client dialled a port file's foreign address %d times", n)
	}
}

// deadPort returns a port nothing listens on.
func deadPort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	l.Close()
	return strings.TrimPrefix(l.Addr().String(), "127.0.0.1:")
}

func writePortFile(t *testing.T, session, port string) string {
	t.Helper()
	pf := portFilePath(session)
	if err := os.WriteFile(pf, []byte(port), 0600); err != nil {
		t.Fatal(err)
	}
	return pf
}

func TestStartServerSweepsOnlyDeadPortFiles(t *testing.T) {
	privateTmp(t)
	live, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer live.Close()

	dead := writePortFile(t, "dead", deadPort(t))
	alive := writePortFile(t, "alive", strings.TrimPrefix(live.Addr().String(), "127.0.0.1:"))
	spawning := writePortFile(t, "spawning", deadPort(t))
	release, err := lockFile(spawning + ".lock")
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	session, _ := startBareServer(t)

	if _, err := os.Stat(dead); !os.IsNotExist(err) {
		t.Errorf("dead-port file kept: %v", err)
	}
	for _, pf := range []string{alive, spawning} {
		if _, err := os.Stat(pf); err != nil {
			t.Errorf("%s removed: %v", pf, err)
		}
	}
	fi, err := os.Stat(portFilePath(session))
	if err != nil {
		t.Fatalf("server's own port file: %v", err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Fatalf("server's port file mode = %v, want 0600", fi.Mode().Perm())
	}
}

func TestListSessionsSkipsHungListener(t *testing.T) {
	privateTmp(t)
	hung, err := net.Listen("tcp", "127.0.0.1:0") // never accepts
	if err != nil {
		t.Fatal(err)
	}
	defer hung.Close()
	writePortFile(t, "hung", strings.TrimPrefix(hung.Addr().String(), "127.0.0.1:"))

	done := make(chan []string, 1)
	go func() { s, _ := ListSessions(); done <- s }()
	select {
	case s := <-done:
		for _, name := range s {
			if name == "hung" {
				t.Fatalf("hung session listed: %v", s)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ListSessions still blocked after 5s on a listener that never accepts")
	}
}

func TestPingTimeoutSparesRealCommands(t *testing.T) {
	privateTmp(t)
	old := pingTimeout
	pingTimeout = 100 * time.Millisecond
	t.Cleanup(func() { pingTimeout = old })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		io.WriteString(w, `{"result":"ok"}`)
	}))
	defer srv.Close()
	writePortFile(t, "slow", strings.TrimPrefix(srv.URL, "http://127.0.0.1:"))

	if _, err := ClientExecute("slow", Request{Command: "ping"}); err == nil {
		t.Fatal("slow ping succeeded, want timeout")
	}
	if res, err := ClientExecute("slow", Request{Command: "goto"}); err != nil || res != "ok" {
		t.Fatalf("slow goto: %q, %v; want ok (ping timeout must not apply)", res, err)
	}
}

func TestListSessionsKeepsLockedPortFile(t *testing.T) {
	privateTmp(t)
	spawning := writePortFile(t, "spawning", deadPort(t))
	release, err := lockFile(spawning + ".lock")
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	if s, _ := ListSessions(); len(s) != 0 {
		t.Fatalf("listed %v, want none", s)
	}
	if _, err := os.Stat(spawning); err != nil {
		t.Fatalf("ListSessions removed a port file whose .lock is held: %v", err)
	}
}
