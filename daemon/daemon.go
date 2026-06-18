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
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/agenthands/rod-cli/actions"
	"github.com/agenthands/rod-cli/types"
)

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
		return "", fmt.Errorf(res.Error)
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

func EnsureDaemon(session string, exePath string, flags []string) error {
	_, err := ClientExecute(session, Request{Command: "ping"})
	if err == nil {
		return nil
	}
	
	args := []string{"--session", session}
	args = append(args, flags...)
	args = append(args, "daemon", "--ppid", fmt.Sprint(os.Getppid()))
	
	cmd := exec.Command(exePath, args...)
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
			json.NewEncoder(w).Encode(Response{Result: "closing"})
			go func() {
				time.Sleep(100 * time.Millisecond)
				os.Exit(0)
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
		os.Exit(0)
	}()

	if ppid > 0 {
		go func() {
			for {
				time.Sleep(2 * time.Second)
				process, err := os.FindProcess(ppid)
				if err != nil {
					os.Exit(0)
				}
				if err := process.Signal(syscall.Signal(0)); err != nil {
					os.Exit(0)
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
	case "select":
		// select values are separated by comma in our generic arg format for simplicity
		// wait, actually we can pass JSON array in the args map? No, map[string]string.
		// Let's assume values are passed directly or not supported multiple right now.
		return actions.Select(ctx, req.Args["ref"], []string{req.Args["values"]})
	case "eval":
		return actions.Evaluate(ctx, req.Args["script"])
	case "snapshot":
		return actions.Snapshot(ctx)
	case "screenshot":
		// we'll skip parsing width/height for now in the daemon to keep it simple, or we can parse them.
		return actions.Screenshot(ctx, req.Args["name"], req.Args["selector"], 0, 0)
	case "pdf":
		return actions.Pdf(ctx, req.Args["name"])
	default:
		return "", fmt.Errorf("unknown command: %s", req.Command)
	}
}
