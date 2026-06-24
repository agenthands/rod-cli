// Package proxyfixture provides an OFFLINE, loopback-only HTTP forward proxy
// for end-to-end proxy tests of the rod-cli binary. It mirrors the
// internal/plugin/scanner/testserver New/Start/Close/URL shape so tests can
// stand one (or several) up on 127.0.0.1:0, hand the URL to the live binary
// via --proxy, and read back which fixture served a given request.
//
// Two variants are exposed:
//
//   - New(id)              — an UNAUTHENTICATED forward proxy.
//   - NewAuth(id, u, p)    — the same, but every CONNECT/forward request must
//     carry Proxy-Authorization: Basic <b64(u:p)>, else
//     the fixture answers 407 Proxy Authentication
//     Required. This is the upstream the godoll relay
//     tunnels to in the rod-cli --proxy-auth path.
//
// Egress identity: each fixture is constructed with a distinguishing id. When
// Chrome (routed through this fixture via Chrome's --proxy-server) issues a
// plain-HTTP forward request, the fixture SERVES ITS OWN tiny target page
// containing that id and increments a per-fixture request counter. A test
// running two fixtures can therefore tell which proxy served a navigation by
// reading the id echoed in the page (or by comparing the two counters) without
// needing any real public IP — the whole thing is offline and deterministic.
//
// WARNING: test-only. This is a deliberately minimal forward proxy with no
// upstream forwarding and no production hardening. Never expose it to untrusted
// networks or use it outside tests.
package proxyfixture

import (
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ProxyFixture is an offline HTTP forward proxy bound to a loopback port.
type ProxyFixture struct {
	id       string
	username string // empty => no auth required
	password string

	listener net.Listener
	server   *http.Server

	requests  atomic.Int64 // total forward/CONNECT requests served (auth-accepted)
	authFails atomic.Int64 // requests rejected with 407
	startOnce sync.Once
	closeOnce sync.Once
}

// New creates an UNAUTHENTICATED offline forward proxy on 127.0.0.1:0.
// id is the distinguishing tag echoed back in served pages so a test can tell
// which fixture handled a request.
func New(id string) (*ProxyFixture, error) {
	return newFixture(id, "", "")
}

// NewAuth creates an offline forward proxy that REQUIRES
// Proxy-Authorization: Basic <b64(username:password)> on every request and
// answers 407 Proxy Authentication Required otherwise.
func NewAuth(id, username, password string) (*ProxyFixture, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("proxyfixture: NewAuth requires non-empty username and password")
	}
	return newFixture(id, username, password)
}

func newFixture(id, username, password string) (*ProxyFixture, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	f := &ProxyFixture{
		id:       id,
		username: username,
		password: password,
		listener: listener,
	}
	f.server = &http.Server{
		Handler:           http.HandlerFunc(f.handle),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return f, nil
}

// Start begins serving in the background. Safe to call once.
func (f *ProxyFixture) Start() {
	f.startOnce.Do(func() {
		go f.server.Serve(f.listener)
	})
}

// Close shuts the fixture down. Safe to call multiple times.
func (f *ProxyFixture) Close() {
	f.closeOnce.Do(func() {
		if f.server != nil {
			f.server.Close()
		}
		if f.listener != nil {
			f.listener.Close()
		}
	})
}

// URL returns the proxy URL suitable for rod-cli's --proxy flag
// (http://host:port). Use Addr() for a bare host:port.
func (f *ProxyFixture) URL() string {
	return "http://" + f.listener.Addr().String()
}

// Addr returns the bare host:port the fixture is listening on.
func (f *ProxyFixture) Addr() string {
	return f.listener.Addr().String()
}

// ID returns the fixture's distinguishing tag.
func (f *ProxyFixture) ID() string { return f.id }

// RequestCount returns the number of auth-accepted forward/CONNECT requests
// this fixture has served. A test asserts a session's traffic went through the
// EXPECTED fixture (count > 0) and zero through the other (count == 0).
func (f *ProxyFixture) RequestCount() int64 { return f.requests.Load() }

// AuthFailureCount returns how many requests this fixture rejected with 407.
func (f *ProxyFixture) AuthFailureCount() int64 { return f.authFails.Load() }

// requiresAuth reports whether this fixture enforces Proxy-Authorization.
func (f *ProxyFixture) requiresAuth() bool { return f.username != "" }

// checkAuth validates the Proxy-Authorization header against the configured
// credentials. Returns true when auth is satisfied (or not required).
func (f *ProxyFixture) checkAuth(h http.Header) bool {
	if !f.requiresAuth() {
		return true
	}
	const prefix = "Basic "
	val := h.Get("Proxy-Authorization")
	if !strings.HasPrefix(val, prefix) {
		return false
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(val, prefix))
	if err != nil {
		return false
	}
	user, pass, ok := strings.Cut(string(raw), ":")
	return ok && user == f.username && pass == f.password
}

// writeProxyAuthRequired emits a 407 with a Proxy-Authenticate challenge.
func (f *ProxyFixture) writeProxyAuthRequired(w http.ResponseWriter) {
	f.authFails.Add(1)
	w.Header().Set("Proxy-Authenticate", `Basic realm="proxyfixture"`)
	w.WriteHeader(http.StatusProxyAuthRequired) // 407
	io.WriteString(w, "407 Proxy Authentication Required\n")
}

// handle dispatches CONNECT (tunnel) vs plain-HTTP forward requests.
func (f *ProxyFixture) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		f.handleConnect(w, r)
		return
	}
	f.handleForward(w, r)
}

// handleForward serves Chrome's absolute-form plain-HTTP forward requests
// (GET http://target/...). Rather than forwarding upstream (which would require
// a real network), the fixture SERVES ITS OWN page tagged with its id — this is
// the egress marker the isolation test reads back. The served page also exposes
// the id at #egress so an `eval` read can pull it from the DOM.
func (f *ProxyFixture) handleForward(w http.ResponseWriter, r *http.Request) {
	if !f.checkAuth(r.Header) {
		f.writeProxyAuthRequired(w)
		return
	}
	f.requests.Add(1)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>proxyfixture</title></head>`+
		`<body><h1 id="egress" data-proxy-id="%s">%s</h1>`+
		`<p>served by proxyfixture egress id=%s for %s</p></body></html>`,
		f.id, f.id, f.id, r.URL.String())
}

// handleConnect tunnels an HTTPS CONNECT. After auth, it accepts the tunnel and
// connects the client to a tiny in-process TLS-less echo is not possible over a
// raw CONNECT (the client speaks TLS), so the fixture simply establishes the
// tunnel to a loopback sink: the counter increment is the egress proof. The
// auth test, however, drives plain-HTTP targets so the SUCCESS path is observed
// through handleForward; CONNECT support exists so Chrome never errors when it
// decides to tunnel.
func (f *ProxyFixture) handleConnect(w http.ResponseWriter, r *http.Request) {
	if !f.checkAuth(r.Header) {
		f.writeProxyAuthRequired(w)
		return
	}
	f.requests.Add(1)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack unsupported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer clientConn.Close()

	// Dial the requested loopback host so the tunnel is real for loopback
	// targets; if it can't be reached, we still acknowledged the CONNECT for
	// counting purposes but close the tunnel.
	upstream, err := net.DialTimeout("tcp", r.Host, 5*time.Second)
	if err != nil {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer upstream.Close()

	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	done := make(chan struct{}, 2)
	go func() { io.Copy(upstream, clientConn); done <- struct{}{} }()
	go func() { io.Copy(clientConn, upstream); done <- struct{}{} }()
	<-done
}
