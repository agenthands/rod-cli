package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/agenthands/rod-cli/internal/testchrome"
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

// privateTmp points TMPDIR (read by os.TempDir) at a fresh dir so the port
// files, locks and logs a test writes, pings or prunes are its own, never the
// developer's live sessions. The dir is short because a daemon's Chrome needs a
// TMPDIR of 40 characters or less.
func privateTmp(t *testing.T) {
	t.Helper()
	d, err := os.MkdirTemp("/tmp", "rc-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	t.Setenv("TMPDIR", d)
}

func TestPortFilePath(t *testing.T) {
	p := portFilePath("abc")
	if !strings.Contains(p, "rod-cli-abc.port") {
		t.Fatalf("unexpected port file path: %s", p)
	}
}

func TestClientExecuteNoDaemon(t *testing.T) {
	privateTmp(t)
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
	privateTmp(t)
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
	privateTmp(t)
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
	privateTmp(t)
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
	privateTmp(t)
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

func TestStartServerAnswersPing(t *testing.T) {
	privateTmp(t)
	// Exercise StartServer's TCP listen, port-file write and /execute handler
	// without a browser (ping only) and without triggering any os.Exit path.
	session := uniqueSession("ping")
	ctx := types.NewContext(context.Background(), types.Config{})
	go func() { _ = StartServer(session, ctx) }()

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
		t.Fatal("server did not become ready")
	}
	os.Remove(pf)
}

func TestListSessionsPrunesDead(t *testing.T) {
	privateTmp(t)
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
	privateTmp(t)
	// Point at a non-existent executable so cmd.Start fails immediately.
	session := uniqueSession("ensurefail")
	_ = os.Remove(portFilePath(session))

	err := EnsureDaemon(session, "/nonexistent/rod-cli-binary-xyz", nil, nil)
	if err == nil {
		t.Fatal("expected EnsureDaemon to fail with a bad exe path")
	}
}

func TestEnsureDaemonSpawnAndClose(t *testing.T) {
	privateTmp(t)
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
	session := uniqueSession("spawn")
	_ = os.Remove(portFilePath(session))

	// Spawn a real headless daemon and wait for it to become reachable.
	err := EnsureDaemon(session, tmpBin, []string{"--headless"}, nil)
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
	if err := EnsureDaemon(session, tmpBin, []string{"--headless"}, nil); err != nil {
		t.Fatalf("second EnsureDaemon should succeed: %v", err)
	}
}

func TestEnsureDaemonConcurrentSpawnsOne(t *testing.T) {
	privateTmp(t)
	if testing.Short() {
		t.Skip("skipping browser/spawn test in short mode")
	}
	tmpBin := os.TempDir() + "/rod-cli-daemontest-" + uniqueSession("bin")
	build := exec.Command("go", "build", "-o", tmpBin, "github.com/agenthands/rod-cli")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	defer os.Remove(tmpBin)

	session := uniqueSession("concurrent")
	pattern := "--session " + session + " .*daemon"
	defer exec.Command("pkill", "-f", "--", pattern).Run()

	var wg sync.WaitGroup
	errs := make([]error, 5)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = EnsureDaemon(session, tmpBin, []string{"--headless"}, nil)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("EnsureDaemon #%d: %v", i, err)
		}
	}

	out, _ := exec.Command("pgrep", "-f", "--", pattern).Output()
	pids := strings.Fields(string(out))
	if len(pids) != 1 {
		t.Errorf("want exactly 1 daemon for session %s, got %d: %v", session, len(pids), pids)
	}
	_, _ = ClientExecute(session, Request{Command: "close"})
}

// writeScript writes an executable shell script to a temp dir and returns its path.
func writeScript(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fake-rod-cli")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEnsureDaemonFailsFastWhenProcessExits(t *testing.T) {
	privateTmp(t)
	exe := writeScript(t, "exit 3")
	start := time.Now()
	err := EnsureDaemon(uniqueSession("exits"), exe, nil, nil)
	if err == nil {
		t.Fatal("expected an error when the spawned process exits")
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Errorf("took %s, want < 3s", d)
	}
	if !strings.Contains(err.Error(), "exit status 3") {
		t.Errorf("error %q does not contain %q", err, "exit status 3")
	}
}

func TestEnsureDaemonKillsOnTimeout(t *testing.T) {
	privateTmp(t)
	old := daemonStartTimeout
	daemonStartTimeout = 1 * time.Second
	defer func() { daemonStartTimeout = old }()

	pidFile := filepath.Join(t.TempDir(), "pid")
	exe := writeScript(t, "echo $$ > "+pidFile+"\nexec sleep 60")
	if err := EnsureDaemon(uniqueSession("hangs"), exe, nil, nil); err == nil {
		t.Fatal("expected a timeout error")
	}
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("script never wrote its pid: %v", err)
	}
	var pid int
	fmt.Sscan(string(raw), &pid)
	proc, _ := os.FindProcess(pid)
	if err := proc.Signal(syscall.Signal(0)); err == nil {
		proc.Kill()
		t.Errorf("spawned process %d still alive after the timeout error", pid)
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
	privateTmp(t)

	u := testchrome.MustLaunch(t, launcher.New().Headless(true))
	ctx := types.NewContext(context.Background(), types.Config{CDPEndpoint: u})
	if _, err := ctx.EnsurePage(); err != nil {
		t.Fatalf("EnsurePage failed: %v", err)
	}

	session := uniqueSession("inproc")
	go func() {
		// StartServer blocks on Serve.
		_ = StartServer(session, ctx)
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
	if err := EnsureDaemon(session, "/nonexistent", nil, nil); err != nil {
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

	u := testchrome.MustLaunch(t, launcher.New().Headless(true))
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

// TestExecMuEliminatesConcurrentNavigateErrors is the regression case for a
// bug found live: two concurrent rod-cli client processes hitting the SAME
// session raced unsynchronized at the CDP level on the single shared
// *rod.Page (types.Context.stateLock only ever guards the page POINTER
// field, never the operation performed once an action holds it).
//
// Empirically (see git history / PR discussion for the throwaway experiment
// this replaced): WITHOUT execMu serializing dispatch, firing N concurrent
// goto+eval pairs against one session reliably produced net::ERR_ABORTED
// browser-level errors on ~60% of navigations — one goto's in-flight
// Page.navigate/WaitDOMStable gets torn out from under it by another
// concurrent goto on the same tab. WITH execMu, that error class is fully
// eliminated (0 errors across 30 trials in 3 independent runs) because each
// command's full CDP interaction now completes atomically before the next
// one starts.
//
// What execMu does NOT and structurally cannot guarantee: that each caller's
// OWN eval reads back ITS OWN goto's page, when a DIFFERENT concurrent
// caller's goto is interleaved in between on the same shared tab. With 10
// callers alternating between two destinations against one session, roughly
// half will legitimately observe whichever destination's navigation ran
// last among the batch — that is the expected, inherent consequence of
// multiple callers sharing one browser tab, not something serialization at
// the daemon level can paper over. True per-caller isolation requires
// distinct --session names (see the skill's multi-target guidance) — that
// is the caller's responsibility, not the daemon's. This test therefore
// asserts zero CDP/browser errors (the actual execMu contract) and only
// logs the destination-mismatch rate for visibility, without asserting on
// it.
func TestExecMuEliminatesConcurrentNavigateErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}
	srvA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1 id="marker">PAGE_A</h1></body></html>`)
	}))
	defer srvA.Close()
	srvB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1 id="marker">PAGE_B</h1></body></html>`)
	}))
	defer srvB.Close()

	session, _, cleanup := startInProcessDaemon(t)
	defer cleanup()

	const n = 10
	var wg sync.WaitGroup
	type outcome struct {
		gotoErr, evalErr error
		wantMarker       string
		gotMarker        string
	}
	results := make([]outcome, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			url, want := srvA.URL, "PAGE_A"
			if i%2 == 1 {
				url, want = srvB.URL, "PAGE_B"
			}
			var o outcome
			o.wantMarker = want
			_, o.gotoErr = ClientExecute(session, Request{Command: "open", Args: map[string]string{"url": url}})
			r, err := ClientExecute(session, Request{Command: "eval", Args: map[string]string{"script": "document.getElementById('marker') ? document.getElementById('marker').innerText : 'NULL_MARKER'"}})
			o.evalErr = err
			o.gotMarker = r
			results[i] = o
		}(i)
	}
	wg.Wait()

	mismatchCount := 0
	for i, o := range results {
		// This IS the execMu contract: no command may fail outright under
		// concurrent same-session load.
		if o.gotoErr != nil {
			t.Errorf("trial %d: goto errored under concurrent load (execMu should prevent this): %v", i, o.gotoErr)
		}
		if o.evalErr != nil {
			t.Errorf("trial %d: eval errored under concurrent load (execMu should prevent this): %v", i, o.evalErr)
		}
		if o.gotoErr == nil && o.evalErr == nil && !strings.Contains(o.gotMarker, o.wantMarker) {
			mismatchCount++
		}
	}
	t.Logf("informational: %d/%d trials observed a DIFFERENT concurrent caller's destination (expected/inherent for N callers sharing one session — not an execMu regression)", mismatchCount, n)
}

// TestExecAcquireTimeoutBoundsAHungCommand is the regression case for a bug
// found live while verifying the execSem serialization fix:
// actions.Screenshot (and ~25 other seam-wrapped go-rod calls in
// actions.go, unlike Navigate) carry no page.Timeout(), so a CDP call that
// never gets a response from the browser (reproduced live: rod.Page.
// Screenshot's Page.captureScreenshot round-trip hung >20s, confirmed via a
// SIGQUIT goroutine dump blocked inside the CDP client) hangs the whole
// request indefinitely. With a plain sync.Mutex serializing dispatch, that
// single hang would wedge the ENTIRE session PERMANENTLY — every
// subsequent command queuing forever behind a lock that never releases,
// strictly worse than the pre-serialization interleaving risk it replaced.
// execAcquireTimeout bounds the wait instead.
//
// A real browser hang isn't practical to wait out in a test, so this
// simulates one deterministically: a synchronous JS busy-wait longer than a
// shrunk execAcquireTimeout stands in for "a command that never returns
// within the bound" — from the daemon's perspective the two are
// indistinguishable, it only ever observes "has the semaphore been
// released yet".
func TestExecAcquireTimeoutBoundsAHungCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}
	srv := newPageServer()
	defer srv.Close()

	session, _, cleanup := startInProcessDaemon(t)
	defer cleanup()

	if _, err := ClientExecute(session, Request{Command: "open", Args: map[string]string{"url": srv.URL}}); err != nil {
		t.Fatalf("open failed: %v", err)
	}

	orig := execAcquireTimeout
	execAcquireTimeout = 1500 * time.Millisecond
	defer func() { execAcquireTimeout = orig }()

	// Command A: a busy-wait comfortably longer than the shrunk timeout,
	// standing in for a hung CDP call. Fired fire-and-forget in the
	// background — the point under test is only that it does NOT block the
	// daemon from answering B; A's own eventual result is irrelevant (it may
	// even error if test cleanup closes the browser out from under it).
	const busyMs = 4000
	script := fmt.Sprintf("() => { const s = Date.now(); while (Date.now() - s < %d) {} return 'done'; }", busyMs)
	go func() {
		_, _ = ClientExecute(session, Request{Command: "eval", Args: map[string]string{"script": script}})
	}()
	time.Sleep(300 * time.Millisecond) // let A actually acquire execSem first

	start := time.Now()
	_, err := ClientExecute(session, Request{Command: "eval", Args: map[string]string{"script": "1+1"}})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected command B to fail with a session-busy error while A is still running, got success")
	}
	if !strings.Contains(err.Error(), "previous command has not completed") {
		t.Fatalf("expected the bounded-acquire-timeout error, got: %v", err)
	}
	// B must fail close to execAcquireTimeout (not immediately — proving it
	// actually waited — and not anywhere near busyMs, proving it didn't just
	// wait for A).
	if elapsed < execAcquireTimeout || elapsed > execAcquireTimeout+2*time.Second {
		t.Fatalf("expected B to fail around execAcquireTimeout (%v), took %v", execAcquireTimeout, elapsed)
	}
}

// liveCtxWithHTML launches headless Chrome, serves html, and opens it.
func liveCtxWithHTML(t *testing.T, html string) *types.Context {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	}))
	t.Cleanup(srv.Close)
	u := testchrome.MustLaunch(t, launcher.New().Headless(true))
	ctx := types.NewContext(context.Background(), types.Config{CDPEndpoint: u})
	t.Cleanup(func() { _ = ctx.CloseBrowser() })
	if _, err := executeAction(ctx, Request{Command: "open", Args: map[string]string{"url": srv.URL}}); err != nil {
		t.Fatalf("open failed: %v", err)
	}
	return ctx
}

func TestExecuteActionSelectPassesList(t *testing.T) {
	ctx := liveCtxWithHTML(t, `<select id="multi" multiple aria-label="Multi Box"><option value="a">Apple</option><option value="b">Banana</option><option value="c">Cherry</option></select>`)
	if _, err := executeAction(ctx, Request{Command: "select", Args: map[string]string{"ref": "#multi", "values": `["Apple","Banana"]`}}); err != nil {
		t.Fatalf("select failed: %v", err)
	}
	got, err := executeAction(ctx, Request{Command: "eval", Args: map[string]string{"script": "document.getElementById('multi').selectedOptions.length"}})
	if err != nil || got != "2" {
		t.Fatalf("selected count = %q (err %v), want 2", got, err)
	}
}

func TestUploadCommaInFilename(t *testing.T) {
	ctx := liveCtxWithHTML(t, `<input id="f" type="file" multiple aria-label="File Input">`)
	fp := filepath.Join(t.TempDir(), "a,b.txt")
	if err := os.WriteFile(fp, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	files, _ := json.Marshal([]string{fp})
	if _, err := executeAction(ctx, Request{Command: "upload", Args: map[string]string{"ref": "#f", "files": string(files)}}); err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	got, err := executeAction(ctx, Request{Command: "eval", Args: map[string]string{"script": "const f = document.getElementById('f').files; f.length + ':' + f[0].name"}})
	if err != nil || got != "1:a,b.txt" {
		t.Fatalf("files = %q (err %v), want 1:a,b.txt", got, err)
	}
}
