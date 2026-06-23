package daemon

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/agenthands/rod-cli/types"
	"github.com/go-rod/rod/lib/launcher"
)

// newPageServer returns an httptest.Server serving a minimal HTML page so
// browser actions have a real, reachable page to operate on.
func newPageServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h1>hi</h1><a href="#">link</a></body></html>`)
	}))
}

// uniqueSession returns a session name unlikely to clash with other tests or
// real daemons, so port files don't collide.
func uniqueSession(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func TestPortFilePath(t *testing.T) {
	p := portFilePath("abc")
	if !strings.Contains(p, "rod-cli-abc.port") {
		t.Fatalf("unexpected port file path: %s", p)
	}
}

func TestClientExecuteNoDaemon(t *testing.T) {
	session := uniqueSession("nodaemon")
	// Ensure no port file exists.
	_ = os.Remove(portFilePath(session))

	_, err := ClientExecute(session, Request{Command: "ping"})
	if err == nil {
		t.Fatal("expected error when no daemon running")
	}
	if err.Error() != "daemon not running" {
		t.Fatalf("expected 'daemon not running', got %v", err)
	}
}

func TestClientExecuteBadPort(t *testing.T) {
	// Write a port file pointing at a port nobody is listening on so the
	// http.Post path returns a connection error (exercises the err branch).
	session := uniqueSession("badport")
	pf := portFilePath(session)
	if err := os.WriteFile(pf, []byte("1"), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(pf)

	_, err := ClientExecute(session, Request{Command: "ping"})
	if err == nil {
		t.Fatal("expected connection error for dead port")
	}
}

func TestClientExecuteDecodeError(t *testing.T) {
	// Server returns a non-JSON body so the decoder errors.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "not json at all")
	}))
	defer srv.Close()

	session := uniqueSession("decodeerr")
	pf := portFilePath(session)
	port := strings.TrimPrefix(srv.URL, "http://127.0.0.1:")
	if err := os.WriteFile(pf, []byte(port), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(pf)

	_, err := ClientExecute(session, Request{Command: "ping"})
	if err == nil {
		t.Fatal("expected decode error")
	}
}

func TestClientExecuteErrorResponse(t *testing.T) {
	// Server returns a Response with a non-empty Error field.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"error":"boom"}`)
	}))
	defer srv.Close()

	session := uniqueSession("errresp")
	pf := portFilePath(session)
	port := strings.TrimPrefix(srv.URL, "http://127.0.0.1:")
	if err := os.WriteFile(pf, []byte(port), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(pf)

	_, err := ClientExecute(session, Request{Command: "ping"})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected error 'boom', got %v", err)
	}
}

func TestClientExecuteSuccessResponse(t *testing.T) {
	// Server returns a Result so the success path is covered too.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"result":"ok"}`)
	}))
	defer srv.Close()

	session := uniqueSession("okresp")
	pf := portFilePath(session)
	port := strings.TrimPrefix(srv.URL, "http://127.0.0.1:")
	if err := os.WriteFile(pf, []byte(port), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(pf)

	res, err := ClientExecute(session, Request{Command: "ping"})
	if err != nil || res != "ok" {
		t.Fatalf("expected result 'ok', got %q err=%v", res, err)
	}
}

func TestStartServerWithPpid(t *testing.T) {
	// Exercise StartServer's TCP listen, port-file write, /execute handler and
	// the ppid>0 watcher goroutine branch, without a browser (ping only) and
	// without triggering any os.Exit path. We use a live ppid so the watcher
	// loop keeps running harmlessly.
	session := uniqueSession("ppid")
	ctx := types.NewContext(context.Background(), types.Config{})
	go func() { _ = StartServer(session, os.Getpid(), ctx) }()

	pf := portFilePath(session)
	ready := false
	for i := 0; i < 200; i++ {
		if _, err := os.Stat(pf); err == nil {
			if r, err := ClientExecute(session, Request{Command: "ping"}); err == nil && r == "pong" {
				ready = true
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !ready {
		os.Remove(pf)
		t.Fatal("server with ppid did not become ready")
	}
	// Let the ppid watcher goroutine run at least one iteration.
	time.Sleep(100 * time.Millisecond)
	os.Remove(pf)
}

func TestListSessionsPrunesDead(t *testing.T) {
	// A stale port file with a dead port should be pruned and not returned.
	session := uniqueSession("stale")
	pf := portFilePath(session)
	if err := os.WriteFile(pf, []byte("1"), 0644); err != nil {
		t.Fatal(err)
	}

	sessions, err := ListSessions()
	if err != nil {
		t.Fatalf("ListSessions error: %v", err)
	}
	for _, s := range sessions {
		if s == session {
			t.Fatalf("dead session %s should have been pruned", session)
		}
	}
	// File should have been removed by ListSessions.
	if _, err := os.Stat(pf); !os.IsNotExist(err) {
		os.Remove(pf)
		t.Fatalf("expected stale port file to be removed")
	}
}

func TestEnsureDaemonFails(t *testing.T) {
	// Point at a non-existent executable so cmd.Start fails immediately.
	session := uniqueSession("ensurefail")
	_ = os.Remove(portFilePath(session))

	err := EnsureDaemon(session, "/nonexistent/rod-cli-binary-xyz", nil)
	if err == nil {
		t.Fatal("expected EnsureDaemon to fail with a bad exe path")
	}
}

func TestEnsureDaemonSpawnAndClose(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser/spawn test in short mode")
	}
	// Build the real rod-cli binary so EnsureDaemon can spawn a working daemon
	// subprocess. EnsureDaemon itself runs in-process, so its spawn+poll-success
	// branch is recorded in this package's coverage profile.
	tmpBin := os.TempDir() + "/rod-cli-daemontest-" + uniqueSession("bin")
	build := exec.Command("go", "build", "-o", tmpBin, "github.com/agenthands/rod-cli")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	defer os.Remove(tmpBin)
	// The spawned daemon (no --config) writes a default config in its cwd.
	defer os.Remove("rod-cli.yaml")

	session := uniqueSession("spawn")
	_ = os.Remove(portFilePath(session))

	// Spawn a real headless daemon and wait for it to become reachable.
	err := EnsureDaemon(session, tmpBin, []string{"--headless"})
	if err != nil {
		t.Fatalf("EnsureDaemon spawn failed: %v", err)
	}
	defer func() {
		// Ask the daemon to close itself (the child calls os.Exit, not us).
		_, _ = ClientExecute(session, Request{Command: "close"})
		time.Sleep(500 * time.Millisecond)
		os.Remove(portFilePath(session))
	}()

	// The spawned daemon should answer ping.
	if r, err := ClientExecute(session, Request{Command: "ping"}); err != nil || r != "pong" {
		t.Fatalf("spawned daemon ping failed: r=%q err=%v", r, err)
	}

	// A second EnsureDaemon should short-circuit (already running).
	if err := EnsureDaemon(session, tmpBin, []string{"--headless"}); err != nil {
		t.Fatalf("second EnsureDaemon should succeed: %v", err)
	}
}

func TestExecuteActionUnknownCommand(t *testing.T) {
	ctx := types.NewContext(context.Background(), types.Config{})
	_, err := executeAction(ctx, Request{Command: "this-does-not-exist"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected 'unknown command' error, got %v", err)
	}
}

// startInProcessDaemon launches StartServer in a goroutine against a real
// headless browser context and waits for the port file to appear. It returns
// the session name and a cleanup func.
func startInProcessDaemon(t *testing.T) (string, *types.Context, func()) {
	t.Helper()

	u := launcher.New().Headless(true).MustLaunch()
	ctx := types.NewContext(context.Background(), types.Config{CDPEndpoint: u})
	if _, err := ctx.EnsurePage(); err != nil {
		t.Fatalf("EnsurePage failed: %v", err)
	}

	session := uniqueSession("inproc")
	go func() {
		// ppid 0 means no parent-watch goroutine; StartServer blocks on Serve.
		_ = StartServer(session, 0, ctx)
	}()

	// Poll for the port file / responsive ping.
	pf := portFilePath(session)
	ready := false
	for i := 0; i < 80; i++ {
		if _, err := os.Stat(pf); err == nil {
			if _, err := ClientExecute(session, Request{Command: "ping"}); err == nil {
				ready = true
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !ready {
		t.Fatal("in-process daemon did not become ready")
	}

	cleanup := func() {
		os.Remove(pf)
		ctx.Close()
	}
	return session, ctx, cleanup
}

func TestInProcessDaemonPingAndActions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}
	srv := newPageServer()
	defer srv.Close()

	session, _, cleanup := startInProcessDaemon(t)
	defer cleanup()

	// ping -> pong
	res, err := ClientExecute(session, Request{Command: "ping"})
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}
	if res != "pong" {
		t.Fatalf("expected pong, got %q", res)
	}

	// EnsureDaemon should be a no-op when the daemon is already running.
	if err := EnsureDaemon(session, "/nonexistent", nil); err != nil {
		t.Fatalf("EnsureDaemon should succeed when daemon already up: %v", err)
	}

	// ListSessions should include the live session.
	sessions, err := ListSessions()
	if err != nil {
		t.Fatalf("ListSessions error: %v", err)
	}
	found := false
	for _, s := range sessions {
		if s == session {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected live session %s in ListSessions: %v", session, sessions)
	}

	// A real action through the /execute handler: navigate to an inline server.
	if _, err := ClientExecute(session, Request{Command: "open", Args: map[string]string{"url": srv.URL}}); err != nil {
		t.Fatalf("open failed: %v", err)
	}
	if _, err := ClientExecute(session, Request{Command: "snapshot"}); err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	// An action that returns an error from executeAction exercises the error
	// branch of the /execute handler (unknown command).
	if _, err := ClientExecute(session, Request{Command: "bogus-cmd"}); err == nil {
		t.Fatal("expected error response from unknown command")
	}

	// Malformed JSON body to /execute exercises the decode-error branch.
	pf := portFilePath(session)
	portData, _ := os.ReadFile(pf)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%s/execute", string(portData)), "application/json", strings.NewReader("{not-json"))
	if err == nil {
		resp.Body.Close()
	}
}

func TestExecuteActionDispatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}
	srv := newPageServer()
	defer srv.Close()

	u := launcher.New().Headless(true).MustLaunch()
	ctx := types.NewContext(context.Background(), types.Config{CDPEndpoint: u})
	if _, err := ctx.EnsurePage(); err != nil {
		t.Fatalf("EnsurePage failed: %v", err)
	}
	defer ctx.Close()

	// Navigate so subsequent actions have a page to operate on.
	if _, err := executeAction(ctx, Request{Command: "open", Args: map[string]string{"url": srv.URL}}); err != nil {
		t.Fatalf("open failed: %v", err)
	}

	// Exercise a broad set of dispatch branches. We don't assert success for
	// each (some may error on about:blank), only that the dispatch path runs.
	cmds := []Request{
		{Command: "goto", Args: map[string]string{"url": srv.URL}},
		{Command: "reload"},
		{Command: "go-back"},
		{Command: "go-forward"},
		{Command: "snapshot"},
		{Command: "eval", Args: map[string]string{"script": "1+1"}},
		{Command: "press", Args: map[string]string{"key": "Escape"}},
		{Command: "mousemove", Args: map[string]string{"x": "1", "y": "2"}},
		{Command: "mousedown", Args: map[string]string{"button": "left"}},
		{Command: "mouseup", Args: map[string]string{"button": "left"}},
		{Command: "mousewheel", Args: map[string]string{"dx": "0", "dy": "10"}},
		{Command: "resize", Args: map[string]string{"width": "800", "height": "600"}},
		{Command: "tab-list"},
		{Command: "tab-new", Args: map[string]string{"url": srv.URL}},
		{Command: "tab-select", Args: map[string]string{"index": "0"}},
		{Command: "tab-close", Args: map[string]string{"index": "1"}},
		{Command: "cookie-get"},
		{Command: "cookie-set", Args: map[string]string{"name": "k", "value": "v"}},
		{Command: "cookie-delete", Args: map[string]string{"name": "k"}},
		{Command: "cookie-clear"},
		{Command: "localstorage-get", Args: map[string]string{"key": "k"}},
		{Command: "localstorage-set", Args: map[string]string{"key": "k", "value": "v"}},
		{Command: "localstorage-delete", Args: map[string]string{"key": "k"}},
		{Command: "localstorage-clear"},
		{Command: "sessionstorage-get", Args: map[string]string{"key": "k"}},
		{Command: "sessionstorage-set", Args: map[string]string{"key": "k", "value": "v"}},
		{Command: "sessionstorage-delete", Args: map[string]string{"key": "k"}},
		{Command: "sessionstorage-clear"},
		{Command: "route", Args: map[string]string{"pattern": "**/x", "body": "b"}},
		{Command: "route-list"},
		{Command: "unroute", Args: map[string]string{"pattern": "**/x"}},
		{Command: "console"},
		{Command: "requests"},
		{Command: "request", Args: map[string]string{"index": "0"}},
		{Command: "highlight-clear"},
		{Command: "plugin-list"},
		{Command: "click", Args: map[string]string{"ref": "nope"}},
		{Command: "dblclick", Args: map[string]string{"ref": "nope"}},
		{Command: "type", Args: map[string]string{"ref": "nope", "text": "x"}},
		{Command: "fill", Args: map[string]string{"ref": "nope", "text": "x", "submit": "true"}},
		{Command: "hover", Args: map[string]string{"ref": "nope"}},
		{Command: "check", Args: map[string]string{"ref": "nope"}},
		{Command: "uncheck", Args: map[string]string{"ref": "nope"}},
		{Command: "upload", Args: map[string]string{"ref": "nope", "files": "a,b"}},
		{Command: "select", Args: map[string]string{"ref": "nope", "values": "v"}},
		{Command: "highlight", Args: map[string]string{"ref": "nope"}},
		{Command: "drag", Args: map[string]string{"start": "a", "end": "b"}},
		{Command: "drop", Args: map[string]string{"ref": "nope", "path": "/tmp/nope"}},
		{Command: "screenshot", Args: map[string]string{"name": "shot", "selector": ""}},
		{Command: "pdf", Args: map[string]string{"name": "doc"}},
		{Command: "state-save", Args: map[string]string{"path": "/tmp/rodclistate-test.json"}},
		{Command: "state-load", Args: map[string]string{"path": "/tmp/rodclistate-test.json"}},
		{Command: "plugin-load", Args: map[string]string{"path": "/tmp/nope-plugin.js"}},
		{Command: "plugin-run", Args: map[string]string{"name": "nope"}},
		{Command: "show", Args: map[string]string{"annotate": "false"}},
	}
	for _, c := range cmds {
		// Ignore results/errors; we only need the dispatch branch to execute.
		_, _ = executeAction(ctx, c)
	}

	// Cover the dialog-accept / dialog-dismiss dispatch branches. These set up
	// a handler goroutine that blocks until a real dialog appears; we trigger
	// one via eval so the goroutine resolves cleanly (otherwise it would panic
	// when the context is later cancelled on Close).
	if _, err := executeAction(ctx, Request{Command: "dialog-accept", Args: map[string]string{"promptText": "x"}}); err == nil {
		_, _ = executeAction(ctx, Request{Command: "eval", Args: map[string]string{"script": "() => alert('hi')"}})
	}
	if _, err := executeAction(ctx, Request{Command: "dialog-dismiss"}); err == nil {
		_, _ = executeAction(ctx, Request{Command: "eval", Args: map[string]string{"script": "() => confirm('q')"}})
	}
	// Give the dialog handler goroutines a moment to resolve before Close.
	time.Sleep(300 * time.Millisecond)
}
