package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/coverage"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/agenthands/rod-cli/actions"
	"github.com/agenthands/rod-cli/types"
)

// exitDaemon terminates the daemon process. If the binary was built with
// `-cover` and GOCOVERDIR is set (i.e. during a coverage run), it flushes the
// accumulated coverage counters first, since os.Exit otherwise skips the
// coverage runtime's atexit writer. In a normal build WriteCountersDir returns
// an error and is ignored, so this is a no-op in production.
func exitDaemon(code int) {
	if dir := os.Getenv("GOCOVERDIR"); dir != "" {
		_ = coverage.WriteCountersDir(dir)
	}
	os.Exit(code)
}

type Request struct {
	Command string            `json:"command"`
	Args    map[string]string `json:"args"`
}

type Response struct {
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func portFilePath(session string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("rod-cli-%s.port", session))
}

func ClientExecute(session string, req Request) (string, error) {
	portFile := portFilePath(session)
	portData, err := os.ReadFile(portFile)
	if err != nil {
		return "", fmt.Errorf("daemon not running")
	}
	port := string(portData)

	body, _ := json.Marshal(req)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%s/execute", port), "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res Response
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	if res.Error != "" {
		return "", fmt.Errorf("%s", res.Error)
	}
	return res.Result, nil
}

func ListSessions() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(os.TempDir(), "rod-cli-*.port"))
	if err != nil {
		return nil, err
	}
	var active []string
	for _, f := range files {
		base := filepath.Base(f)
		session := strings.TrimSuffix(strings.TrimPrefix(base, "rod-cli-"), ".port")
		_, err := ClientExecute(session, Request{Command: "ping"})
		if err == nil {
			active = append(active, session)
		} else {
			os.Remove(f)
		}
	}
	return active, nil
}

// EnsureDaemon spawns the per-session daemon if it is not already running.
// Non-secret config is forwarded as CLI flags (visible in the daemon argv);
// secrets MUST be passed via extraEnv ("KEY=value" entries) instead, since argv
// is world-readable through /proc/<pid>/cmdline and `ps`. extraEnv is appended
// to the inherited environment for the spawned daemon only.
func EnsureDaemon(session string, exePath string, flags []string, extraEnv []string) error {
	_, err := ClientExecute(session, Request{Command: "ping"})
	if err == nil {
		return nil
	}

	args := []string{"--session", session}
	args = append(args, flags...)
	args = append(args, "daemon")

	cmd := exec.Command(exePath, args...)
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	// 0600: the daemon log may capture proxy/CONNECT diagnostics from the godoll
	// relay — keep it owner-only, never world-readable.
	logFile, _ := os.OpenFile(filepath.Join(os.TempDir(), "rod-cli-daemon.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return err
	}

	for i := 0; i < 40; i++ {
		time.Sleep(250 * time.Millisecond)
		_, err := ClientExecute(session, Request{Command: "ping"})
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("failed to start daemon")
}

func StartServer(session string, ppid int, rodCtx *types.Context) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	portFile := portFilePath(session)
	if err := os.WriteFile(portFile, []byte(fmt.Sprint(port)), 0644); err != nil {
		return err
	}
	defer os.Remove(portFile)

	mux := http.NewServeMux()
	idleTimer := time.NewTimer(15 * time.Minute)

	mux.HandleFunc("/execute", func(w http.ResponseWriter, r *http.Request) {
		idleTimer.Reset(15 * time.Minute)
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(Response{Error: err.Error()})
			return
		}

		if req.Command == "ping" {
			json.NewEncoder(w).Encode(Response{Result: "pong"})
			return
		}
		if req.Command == "close" {
			os.Remove(portFile)
			json.NewEncoder(w).Encode(Response{Result: "closing"})
			go func() {
				time.Sleep(10 * time.Millisecond)
				exitDaemon(0)
			}()
			return
		}

		res, err := executeAction(rodCtx, req)
		if err != nil {
			json.NewEncoder(w).Encode(Response{Error: err.Error()})
			return
		}
		json.NewEncoder(w).Encode(Response{Result: res})
	})

	go func() {
		<-idleTimer.C
		log.Info("Idle timeout reached, shutting down daemon")
		exitDaemon(0)
	}()

	if ppid > 0 {
		go func() {
			for {
				time.Sleep(2 * time.Second)
				process, err := os.FindProcess(ppid)
				if err != nil {
					exitDaemon(0)
				}
				if err := process.Signal(syscall.Signal(0)); err != nil {
					exitDaemon(0)
				}
			}
		}()
	}

	return http.Serve(listener, mux)
}

func executeAction(ctx *types.Context, req Request) (string, error) {
	switch req.Command {
	case "open", "goto":
		return actions.Navigate(ctx, req.Args["url"])
	case "go-back":
		return actions.GoBack(ctx)
	case "go-forward":
		return actions.GoForward(ctx)
	case "reload":
		return actions.Reload(ctx)
	case "click":
		return actions.Click(ctx, req.Args["ref"])
	case "dblclick":
		return actions.DblClick(ctx, req.Args["ref"])
	case "type":
		return actions.Type(ctx, req.Args["ref"], req.Args["text"])
	case "fill":
		submit := req.Args["submit"] == "true"
		return actions.Fill(ctx, req.Args["ref"], req.Args["text"], submit)
	case "hover":
		return actions.Hover(ctx, req.Args["ref"])
	case "check":
		return actions.Check(ctx, req.Args["ref"])
	case "uncheck":
		return actions.Uncheck(ctx, req.Args["ref"])
	case "plugin-load":
		return actions.PluginLoad(ctx, req.Args["path"])
	case "plugin-list":
		return actions.PluginList(ctx)
	case "plugin-run":
		return actions.PluginRun(ctx, req.Args["name"])
	case "upload":
		files := strings.Split(req.Args["files"], ",")
		return actions.Upload(ctx, req.Args["ref"], files)
	case "select":
		// select values are separated by comma in our generic arg format for simplicity
		// wait, actually we can pass JSON array in the args map? No, map[string]string.
		// Let's assume values are passed directly or not supported multiple right now.
		return actions.Select(ctx, req.Args["ref"], []string{req.Args["values"]})
	case "eval":
		return actions.Evaluate(ctx, req.Args["script"], req.Args["ref"])
	case "snapshot":
		return actions.Snapshot(ctx)
	case "stealth-check":
		return actions.StealthCheck(ctx, req.Args["url"], req.Args["raw"] == "true", req.Args["json"] == "true")
	case "screenshot":
		return actions.Screenshot(ctx, req.Args["name"], req.Args["selector"], 0, 0)
	case "pdf":
		return actions.Pdf(ctx, req.Args["name"])
	case "press":
		return actions.Press(ctx, req.Args["key"])
	case "mousemove":
		var x, y float64
		fmt.Sscanf(req.Args["x"], "%f", &x)
		fmt.Sscanf(req.Args["y"], "%f", &y)
		return actions.MouseMove(ctx, x, y)
	case "mousedown":
		return actions.MouseDown(ctx, req.Args["button"])
	case "mouseup":
		return actions.MouseUp(ctx, req.Args["button"])
	case "mousewheel":
		var dx, dy float64
		fmt.Sscanf(req.Args["dx"], "%f", &dx)
		fmt.Sscanf(req.Args["dy"], "%f", &dy)
		return actions.MouseWheel(ctx, dx, dy)
	case "resize":
		var w, h int
		fmt.Sscanf(req.Args["width"], "%d", &w)
		fmt.Sscanf(req.Args["height"], "%d", &h)
		return actions.Resize(ctx, w, h)
	case "tab-list":
		return actions.TabList(ctx)
	case "tab-new":
		return actions.TabNew(ctx, req.Args["url"])
	case "tab-close":
		var index int
		fmt.Sscanf(req.Args["index"], "%d", &index)
		return actions.TabClose(ctx, index)
	case "tab-select":
		var index int
		fmt.Sscanf(req.Args["index"], "%d", &index)
		return actions.TabSelect(ctx, index)
	case "dialog-accept":
		return actions.HandleDialog(ctx, true, req.Args["promptText"])
	case "dialog-dismiss":
		return actions.HandleDialog(ctx, false, "")
	case "cookie-get":
		return actions.GetCookies(ctx)
	case "cookie-set":
		return actions.SetCookie(ctx, req.Args["name"], req.Args["value"])
	case "cookie-delete":
		return actions.DeleteCookie(ctx, req.Args["name"])
	case "cookie-clear":
		return actions.ClearCookies(ctx)
	case "localstorage-get":
		return actions.EvalStorage(ctx, "localStorage", "get", req.Args["key"], "")
	case "localstorage-set":
		return actions.EvalStorage(ctx, "localStorage", "set", req.Args["key"], req.Args["value"])
	case "localstorage-delete":
		return actions.EvalStorage(ctx, "localStorage", "delete", req.Args["key"], "")
	case "localstorage-clear":
		return actions.EvalStorage(ctx, "localStorage", "clear", "", "")
	case "sessionstorage-get":
		return actions.EvalStorage(ctx, "sessionStorage", "get", req.Args["key"], "")
	case "sessionstorage-set":
		return actions.EvalStorage(ctx, "sessionStorage", "set", req.Args["key"], req.Args["value"])
	case "sessionstorage-delete":
		return actions.EvalStorage(ctx, "sessionStorage", "delete", req.Args["key"], "")
	case "sessionstorage-clear":
		return actions.EvalStorage(ctx, "sessionStorage", "clear", "", "")
	case "route":
		return actions.Route(ctx, req.Args["pattern"], req.Args["body"])
	case "unroute":
		return actions.Unroute(ctx, req.Args["pattern"])
	case "route-list":
		return actions.RouteList(ctx)
	case "console":
		return actions.ConsoleLogs(ctx)
	case "requests":
		return actions.NetworkRequests(ctx)
	case "request":
		var index int
		fmt.Sscanf(req.Args["index"], "%d", &index)
		return actions.NetworkRequest(ctx, index)
	case "drag":
		return actions.Drag(ctx, req.Args["start"], req.Args["end"])
	case "drop":
		return actions.Drop(ctx, req.Args["ref"], req.Args["path"])
	case "state-save":
		return actions.StateSave(ctx, req.Args["path"])
	case "state-load":
		return actions.StateLoad(ctx, req.Args["path"])
	case "highlight":
		return actions.Highlight(ctx, req.Args["ref"])
	case "highlight-clear":
		return actions.ClearHighlights(ctx)

	case "show":
		annotate := req.Args["annotate"] == "true"
		return actions.Show(ctx, annotate)
	default:
		return "", fmt.Errorf("unknown command: %s", req.Command)
	}
}
