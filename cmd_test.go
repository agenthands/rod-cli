package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/launcher"
)

// newCDPEndpoint launches a headless browser and returns its CDP websocket URL.
func newCDPEndpoint(t *testing.T) string {
	t.Helper()
	return launcher.New().Headless(true).MustLaunch()
}

// TestMain lets a re-exec of the test binary invoke the real main() so that
// main()'s code (including its os.Exit/log.Fatal paths) is exercised in a child
// process. With `go test -cover`, Go propagates GOCOVERDIR to the child and
// merges its coverage counters into the profile.
func TestMain(m *testing.M) {
	if os.Getenv("ROD_CLI_RUN_MAIN") == "1" {
		// Rebuild os.Args from the controlled env so main() parses our args.
		argStr := os.Getenv("ROD_CLI_MAIN_ARGS")
		args := []string{"rod-cli"}
		if argStr != "" {
			args = append(args, strings.Split(argStr, "\x1f")...)
		}
		os.Args = args
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// runMainSubprocess re-execs the test binary so it calls main() with the given
// CLI args, returning combined output and the process error.
func runMainSubprocess(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(),
		"ROD_CLI_RUN_MAIN=1",
		"ROD_CLI_MAIN_ARGS="+strings.Join(args, "\x1f"),
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// TestMainErrorJSON covers main()'s --json error-formatting branch (a missing
// required arg makes app.Run return an error; --json triggers JSON output and
// os.Exit(1) in the child).
func TestMainErrorJSON(t *testing.T) {
	out, err := runMainSubprocess(t, "--json", "click")
	if err == nil {
		t.Fatal("expected non-zero exit from main on error")
	}
	if !strings.Contains(out, `"error"`) || !strings.Contains(out, "element ref is required") {
		t.Fatalf("expected JSON error output, got: %s", out)
	}
}

// TestMainErrorPlain covers main()'s non-json error branch (log.Fatal).
func TestMainErrorPlain(t *testing.T) {
	out, err := runMainSubprocess(t, "click")
	if err == nil {
		t.Fatal("expected non-zero exit from main on error")
	}
	if !strings.Contains(out, "element ref is required") {
		t.Fatalf("expected error output, got: %s", out)
	}
}

// TestMainSuccess covers main()'s happy path (app.Run returns nil).
func TestMainSuccess(t *testing.T) {
	clearPortFiles(t)
	out, err := runMainSubprocess(t, "--no-banner")
	if err != nil {
		t.Fatalf("expected success, got err %v out %s", err, out)
	}
}

// captureStdout runs fn while capturing everything written to os.Stdout and
// returns the captured text.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fn()

	w.Close()
	os.Stdout = old
	return <-done
}

// runApp builds a fresh app and runs it with the given args (rod-cli prepended).
func runApp(args ...string) error {
	app := getApp()
	full := append([]string{"rod-cli"}, args...)
	return app.Run(full)
}

// --- Missing-argument negative cases (no daemon/browser needed) ---

func TestMissingArgErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"open", []string{"open"}, "URL is required"},
		{"click", []string{"click"}, "element ref is required"},
		{"dblclick", []string{"dblclick"}, "element ref is required"},
		{"type-no-args", []string{"type"}, "element ref and text are required"},
		{"type-one-arg", []string{"type", "ref1"}, "element ref and text are required"},
		{"fill", []string{"fill", "ref1"}, "element ref and text are required"},
		{"hover", []string{"hover"}, "element ref is required"},
		{"check", []string{"check"}, "element ref is required"},
		{"uncheck", []string{"uncheck"}, "element ref is required"},
		{"upload-no-file", []string{"upload", "ref1"}, "element ref and at least one file are required"},
		{"select-no-vals", []string{"select", "ref1"}, "element ref and values are required"},
		{"eval", []string{"eval"}, "script is required"},
		{"press", []string{"press"}, "key is required"},
		{"mousemove", []string{"mousemove", "1"}, "x and y are required"},
		{"mousewheel", []string{"mousewheel", "1"}, "dx and dy are required"},
		{"resize", []string{"resize", "100"}, "width and height are required"},
		{"tab-close", []string{"tab-close"}, "tab index is required"},
		{"tab-select", []string{"tab-select"}, "tab index is required"},
		{"cookie-set", []string{"cookie-set", "n"}, "cookie name and value are required"},
		{"cookie-delete", []string{"cookie-delete"}, "cookie name is required"},
		{"route", []string{"route"}, "route pattern is required"},
		{"request", []string{"request"}, "request index is required"},
		{"drag", []string{"drag", "a"}, "start and end refs are required"},
		{"drop-no-ref", []string{"drop"}, "element ref is required"},
		{"drop-no-path", []string{"drop", "ref1"}, "--path is required"},
		{"state-save", []string{"state-save"}, "file path is required"},
		{"state-load", []string{"state-load"}, "file path is required"},
		{"highlight", []string{"highlight"}, "element ref is required"},
		{"plugin-load", []string{"plugin", "load"}, "plugin path is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := runApp(tc.args...)
			if err == nil {
				t.Fatalf("expected error for %v", tc.args)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %q", tc.want, err.Error())
			}
		})
	}
}

// --- install command (uses cached browser, no download) ---

func TestInstallPlain(t *testing.T) {
	out := captureStdout(t, func() {
		if err := runApp("install"); err != nil {
			t.Errorf("install failed: %v", err)
		}
	})
	if !strings.Contains(out, "installed successfully") {
		t.Fatalf("expected install success message, got: %s", out)
	}
}

func TestInstallJSON(t *testing.T) {
	out := captureStdout(t, func() {
		if err := runApp("--json", "install"); err != nil {
			t.Errorf("install --json failed: %v", err)
		}
	})
	var m map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Fatalf("expected JSON output, got %q (err %v)", out, err)
	}
	if m["status"] != "installed" || m["path"] == "" {
		t.Fatalf("unexpected install json: %v", m)
	}
}

// --- sessions command ---

func TestSessionsEmpty(t *testing.T) {
	clearPortFiles(t)
	out := captureStdout(t, func() {
		if err := runApp("sessions"); err != nil {
			t.Errorf("sessions failed: %v", err)
		}
	})
	if !strings.Contains(out, "No active sessions") {
		t.Fatalf("expected 'No active sessions', got: %s", out)
	}
}

func TestSessionsJSON(t *testing.T) {
	clearPortFiles(t)
	out := captureStdout(t, func() {
		if err := runApp("--json", "sessions"); err != nil {
			t.Errorf("sessions --json failed: %v", err)
		}
	})
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Fatalf("expected JSON output, got %q (err %v)", out, err)
	}
	if _, ok := m["sessions"]; !ok {
		t.Fatalf("expected 'sessions' key in json: %v", m)
	}
}

func TestSessionsListsLive(t *testing.T) {
	clearPortFiles(t)
	srv, session := startFakeDaemon(t)
	defer srv.Close()
	defer os.Remove(portFilePathForSession(session))

	out := captureStdout(t, func() {
		if err := runApp("sessions"); err != nil {
			t.Errorf("sessions failed: %v", err)
		}
	})
	if !strings.Contains(out, "Active sessions:") || !strings.Contains(out, session) {
		t.Fatalf("expected active session %s listed, got: %s", session, out)
	}
}

// --- default action / help / banner ---

func TestHelpWithBanner(t *testing.T) {
	out := captureStdout(t, func() {
		if err := runApp(); err != nil {
			t.Errorf("default action failed: %v", err)
		}
	})
	if !strings.Contains(out, "rod-cli") {
		t.Fatalf("expected help/banner output, got: %s", out)
	}
}

func TestHelpNoBanner(t *testing.T) {
	out := captureStdout(t, func() {
		if err := runApp("--no-banner"); err != nil {
			t.Errorf("default action failed: %v", err)
		}
	})
	if !strings.Contains(out, "COMMANDS") && !strings.Contains(out, "USAGE") {
		t.Fatalf("expected help output, got: %s", out)
	}
}

// --- runClientCommand happy paths via a fake daemon (json/raw/plain) ---

func TestRunClientCommandOutputBranches(t *testing.T) {
	clearPortFiles(t)
	srv, session := startFakeDaemon(t)
	defer srv.Close()
	defer os.Remove(portFilePathForSession(session))

	// plain output
	out := captureStdout(t, func() {
		if err := runApp("--session", session, "snapshot"); err != nil {
			t.Errorf("snapshot failed: %v", err)
		}
	})
	if !strings.Contains(out, "fake-result") {
		t.Fatalf("plain: expected fake-result, got %q", out)
	}

	// raw output
	out = captureStdout(t, func() {
		if err := runApp("--raw", "--session", session, "snapshot"); err != nil {
			t.Errorf("snapshot --raw failed: %v", err)
		}
	})
	if !strings.Contains(out, "fake-result") {
		t.Fatalf("raw: expected fake-result, got %q", out)
	}

	// json output
	out = captureStdout(t, func() {
		if err := runApp("--json", "--session", session, "snapshot"); err != nil {
			t.Errorf("snapshot --json failed: %v", err)
		}
	})
	var m map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Fatalf("json: expected JSON, got %q (err %v)", out, err)
	}
	if m["result"] != "fake-result" {
		t.Fatalf("json: expected result=fake-result, got %v", m)
	}
}

// TestRunClientCommandDaemonError exercises the path where ClientExecute
// returns an error after the daemon is already considered running (the fake
// daemon answers ping but returns an error for the real command).
func TestRunClientCommandDaemonError(t *testing.T) {
	clearPortFiles(t)
	srv, session := startErrorDaemon(t)
	defer srv.Close()
	defer os.Remove(portFilePathForSession(session))

	err := runApp("--session", session, "snapshot")
	if err == nil {
		t.Fatal("expected error from daemon")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected 'boom' error, got %v", err)
	}
}

// TestRunClientCommandFlagForwarding ensures the spawn-flag formatting block in
// runClientCommand runs (config/cdp/headless/vision set). The daemon already
// runs (fake) so EnsureDaemon short-circuits without spawning.
func TestRunClientCommandFlagForwarding(t *testing.T) {
	clearPortFiles(t)
	srv, session := startFakeDaemon(t)
	defer srv.Close()
	defer os.Remove(portFilePathForSession(session))

	out := captureStdout(t, func() {
		err := runApp("--config", "x.yaml", "--cdp-endpoint", "ws://x", "--headless", "--vision",
			"--session", session, "snapshot")
		if err != nil {
			t.Errorf("snapshot with flags failed: %v", err)
		}
	})
	if !strings.Contains(out, "fake-result") {
		t.Fatalf("expected fake-result, got %q", out)
	}
}

// TestAllCommandActionsHappyPath runs each command's Action closure with valid
// args against the fake daemon so the runClientCommand call inside every
// command body executes (covering getApp's command closures).
func TestAllCommandActionsHappyPath(t *testing.T) {
	clearPortFiles(t)
	srv, session := startFakeDaemon(t)
	defer srv.Close()
	defer os.Remove(portFilePathForSession(session))

	cmds := [][]string{
		{"open", "http://example.com"},
		{"goto", "http://example.com"},
		{"go-back"},
		{"go-forward"},
		{"reload"},
		{"click", "ref1"},
		{"dblclick", "ref1"},
		{"type", "ref1", "hello"},
		{"fill", "ref1", "hello"},
		{"fill", "--submit", "ref1", "hello"},
		{"hover", "ref1"},
		{"check", "ref1"},
		{"uncheck", "ref1"},
		{"upload", "ref1", "file1.txt"},
		{"select", "ref1", "optionA"},
		{"eval", "1+1"},
		{"eval", "1+1", "ref1"},
		{"snapshot"},
		{"screenshot"},
		{"screenshot", "--name", "shot", "--selector", "body"},
		{"pdf"},
		{"pdf", "--name", "doc"},
		{"press", "Enter"},
		{"mousemove", "1", "2"},
		{"mousedown", "left"},
		{"mouseup", "left"},
		{"mousewheel", "0", "10"},
		{"resize", "800", "600"},
		{"tab-list"},
		{"tab-new", "http://example.com"},
		{"tab-new"},
		{"tab-close", "1"},
		{"tab-select", "0"},
		{"plugin", "load", "/tmp/p.js"},
		{"plugin", "list"},
		{"plugin", "run", "p1"},
		{"plugin", "run"},
		{"dialog-accept"},
		{"dialog-accept", "--prompt", "txt"},
		{"dialog-dismiss"},
		{"cookie-get"},
		{"cookie-set", "k", "v"},
		{"cookie-delete", "k"},
		{"cookie-clear"},
		{"localstorage-get"},
		{"localstorage-get", "k"},
		{"localstorage-set", "k", "v"},
		{"localstorage-delete", "k"},
		{"localstorage-clear"},
		{"sessionstorage-get"},
		{"sessionstorage-get", "k"},
		{"sessionstorage-set", "k", "v"},
		{"sessionstorage-delete", "k"},
		{"sessionstorage-clear"},
		{"route", "--body", "b", "**/x"},
		{"route-list"},
		{"unroute", "**/x"},
		{"unroute"},
		{"console"},
		{"requests"},
		{"request", "0"},
		{"drag", "a", "b"},
		{"drop", "--path", "/tmp/x", "ref1"},
		{"state-save", "/tmp/state.json"},
		{"state-load", "/tmp/state.json"},
		{"highlight", "ref1"},
		{"highlight-clear"},
		{"show"},
		{"show", "--annotate"},
		{"close"},
	}
	for _, c := range cmds {
		t.Run(strings.Join(c, "_"), func(t *testing.T) {
			args := append([]string{"--session", session}, c...)
			out := captureStdout(t, func() {
				if err := runApp(args...); err != nil {
					t.Errorf("command %v failed: %v", c, err)
				}
			})
			if !strings.Contains(out, "fake-result") {
				t.Errorf("command %v: expected fake-result, got %q", c, out)
			}
		})
	}
}

// TestRunDaemonServerBadConfig covers runDaemonServer's early error return when
// the config file fails to parse. The file must be named like the expected
// config (rod-cli.yaml) and contain invalid YAML so LoadConfig returns an error.
func TestRunDaemonServerBadConfig(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "rod-cli.yaml")
	if err := os.WriteFile(bad, []byte("\tnot: [valid: yaml"), 0644); err != nil {
		t.Fatal(err)
	}
	err := runApp("--config", bad, "daemon")
	if err == nil {
		t.Fatal("expected error parsing invalid config")
	}
}

// TestRunDaemonServerStarts launches the hidden daemon command in-process
// (which calls runDaemonServer -> StartServer with a real headless browser),
// waits for it to answer ping, then abandons it. We never send "close" because
// that triggers os.Exit inside this test process.
func TestRunDaemonServerStarts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}
	clearPortFiles(t)
	session := uniqueSession("rds")
	pf := portFilePathForSession(session)

	go func() {
		// Blocks in http.Serve; will be abandoned when the test ends.
		_ = runApp("--headless", "--session", session, "daemon")
	}()

	defer os.Remove(pf)
	// runDaemonServer with no --config writes a default config in cwd.
	defer os.Remove("rod-cli.yaml")

	if !waitForPong(pf) {
		t.Fatal("runDaemonServer did not start a reachable daemon")
	}
}

// TestRunDaemonServerVisionAndCDP exercises the --vision and --cdp-endpoint
// config-override branches of runDaemonServer by connecting the daemon to an
// already-launched browser via CDP.
func TestRunDaemonServerVisionAndCDP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}
	clearPortFiles(t)

	u := newCDPEndpoint(t)
	session := uniqueSession("rdscdp")
	pf := portFilePathForSession(session)

	go func() {
		_ = runApp("--vision", "--cdp-endpoint", u, "--session", session, "daemon")
	}()
	defer os.Remove(pf)

	if !waitForPong(pf) {
		t.Fatal("runDaemonServer (vision/cdp) did not start a reachable daemon")
	}
}

func waitForPong(pf string) bool {
	for i := 0; i < 120; i++ {
		if _, err := os.Stat(pf); err == nil {
			portData, _ := os.ReadFile(pf)
			resp, err := http.Post(
				fmt.Sprintf("http://127.0.0.1:%s/execute", strings.TrimSpace(string(portData))),
				"application/json",
				strings.NewReader(`{"command":"ping"}`),
			)
			if err == nil {
				b, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				if strings.Contains(string(b), "pong") {
					return true
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// TestRunClientCommandEnsureDaemonError covers runClientCommand's branch where
// EnsureDaemon fails: no daemon is running and the executable used to spawn one
// does not exist, so cmd.Start fails immediately.
func TestRunClientCommandEnsureDaemonError(t *testing.T) {
	clearPortFiles(t)
	session := uniqueSession("noensure")
	_ = os.Remove(portFilePathForSession(session))

	// Make os.Args[0] (used as the daemon exe path) point at a missing binary
	// so EnsureDaemon's cmd.Start fails immediately instead of polling for 10s.
	oldArg0 := os.Args[0]
	os.Args[0] = filepath.Join(os.TempDir(), "rod-cli-definitely-missing-binary-xyz")
	defer func() { os.Args[0] = oldArg0 }()

	err := runApp("--session", session, "snapshot")
	if err == nil {
		t.Fatal("expected EnsureDaemon failure")
	}
	if !strings.Contains(err.Error(), "failed to ensure daemon") {
		t.Fatalf("expected 'failed to ensure daemon', got %v", err)
	}
}

// --- helpers for fake daemon over the port-file protocol ---

func portFilePathForSession(session string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("rod-cli-%s.port", session))
}

func clearPortFiles(t *testing.T) {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join(os.TempDir(), "rod-cli-*.port"))
	for _, f := range files {
		os.Remove(f)
	}
}

func uniqueSession(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// startFakeDaemon spins up an HTTP server speaking the daemon /execute protocol:
// it answers "pong" to ping (so EnsureDaemon short-circuits) and "fake-result"
// for any other command. It writes the port file so ClientExecute finds it.
func startFakeDaemon(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Command string `json:"command"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Command == "ping" {
			fmt.Fprint(w, `{"result":"pong"}`)
			return
		}
		fmt.Fprint(w, `{"result":"fake-result"}`)
	}))
	session := uniqueSession("fake")
	writePortFile(t, srv, session)
	return srv, session
}

// startErrorDaemon answers ping but returns an error for any other command.
func startErrorDaemon(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Command string `json:"command"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Command == "ping" {
			fmt.Fprint(w, `{"result":"pong"}`)
			return
		}
		fmt.Fprint(w, `{"error":"boom"}`)
	}))
	session := uniqueSession("err")
	writePortFile(t, srv, session)
	return srv, session
}

func writePortFile(t *testing.T, srv *httptest.Server, session string) {
	t.Helper()
	port := strings.TrimPrefix(srv.URL, "http://127.0.0.1:")
	if err := os.WriteFile(portFilePathForSession(session), []byte(port), 0644); err != nil {
		t.Fatal(err)
	}
}
