// Package detect provides a deterministic, offline, self-authored bot-detection
// fixture server for the rod-cli stealth harness.
//
// DetectServer binds an ephemeral loopback port (127.0.0.1:0) and serves a
// //go:embed-bundled detection page (detect.html + detect.js). Navigating the
// real rod-cli browser to the served page runs the table-stakes detection
// probes and writes per-signal verdicts into window.__detect, which the e2e
// harness reads back via the `eval` command.
//
// All responses are static embedded assets; there is no reflected/untrusted
// input and the listener binds loopback only — never a public interface.
package detect

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
)

// Scorecard is an optional per-run result the detection page may POST to
// /result. The e2e harness reads window.__detect directly via `eval`, so this
// sink is a convenience only and is never required for a test to pass.
type Scorecard struct {
	Signals map[string]interface{} `json:"signals"`
}

// DetectServer is an offline detection fixture bound to an ephemeral loopback
// port. Use New to construct, Start to serve, Close to shut down.
type DetectServer struct {
	listener net.Listener
	mux      *http.ServeMux

	mu        sync.Mutex
	scorecard *Scorecard
}

// New creates a detection fixture server bound to 127.0.0.1:0 (loopback only,
// ephemeral port). It registers the static routes but does not begin serving;
// call Start for that.
func New() (*DetectServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &DetectServer{
		listener: listener,
		mux:      http.NewServeMux(),
	}
	s.registerRoutes()
	return s, nil
}

// Start begins serving HTTP requests in the background.
func (s *DetectServer) Start() {
	go http.Serve(s.listener, s.mux)
}

// Close shuts down the server by closing its listener.
func (s *DetectServer) Close() {
	s.listener.Close()
}

// URL returns the base URL of the running server (http://127.0.0.1:<port>).
func (s *DetectServer) URL() string {
	return fmt.Sprintf("http://%s", s.listener.Addr().String())
}

// LastScorecard returns the most recent scorecard POSTed to /result, or nil if
// none has been received.
func (s *DetectServer) LastScorecard() *Scorecard {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scorecard
}

func (s *DetectServer) registerRoutes() {
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/detect.js", s.handleScript)
	s.mux.HandleFunc("/result", s.handleResult)
}

// handleIndex serves the embedded detection page at the root path.
func (s *DetectServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, detectHTML)
}

// handleScript serves the embedded detection JavaScript.
func (s *DetectServer) handleScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	fmt.Fprint(w, detectJS)
}

// handleResult is an optional sink for a POSTed scorecard. It stores the most
// recent payload under a mutex; the e2e harness does not rely on it.
func (s *DetectServer) handleResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var sc Scorecard
	if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	s.scorecard = &sc
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}
