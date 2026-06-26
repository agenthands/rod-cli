// Package cdpproxy implements a pass-through CDP WebSocket proxy that sits
// between go-rod's cdp.Client and Chrome's debugging WebSocket.
//
// In pass-through mode (v1), the proxy forwards all messages unchanged while
// logging them to an in-memory ring buffer. Future phases will add Runtime
// domain normalization and timing jitter.
//
// The proxy implements cdp.WebSocketable so it can be dropped into go-rod's
// Client.Start() in place of a raw WebSocket connection.
package cdpproxy

import (
	"encoding/json"
	"sync"

	"github.com/go-rod/rod/lib/cdp"
)

// Proxy wraps a cdp.WebSocketable and logs all CDP traffic to an in-memory
// ring buffer. In v1 (pass-through), Send and Read forward to the inner
// WebSocketable unchanged — the proxy is transparent to go-rod and Chrome.
type Proxy struct {
	inner cdp.WebSocketable

	mu  sync.Mutex
	log []CDPMessage
	cap int // max log entries
}

// CDPMessage is a logged CDP protocol message with direction.
type CDPMessage struct {
	Direction string          `json:"direction"` // "send" or "recv"
	Raw       json.RawMessage `json:"raw"`
}

// New creates a pass-through proxy wrapping the given WebSocketable.
// cap sets the maximum number of logged messages (ring buffer).
func New(inner cdp.WebSocketable, cap int) *Proxy {
	if cap <= 0 {
		cap = 256
	}
	return &Proxy{inner: inner, log: make([]CDPMessage, 0, cap), cap: cap}
}

// Send forwards data to Chrome and logs it.
func (p *Proxy) Send(data []byte) error {
	p.logMessage("send", data)
	return p.inner.Send(data)
}

// Read reads from Chrome and logs it.
func (p *Proxy) Read() ([]byte, error) {
	data, err := p.inner.Read()
	if err != nil {
		return data, err
	}
	p.logMessage("recv", data)
	return data, nil
}

// Traffic returns a copy of the logged CDP messages since the proxy started.
func (p *Proxy) Traffic() []CDPMessage {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]CDPMessage, len(p.log))
	copy(out, p.log)
	return out
}

// logMessage appends a message to the ring buffer.
func (p *Proxy) logMessage(dir string, data []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.log) >= p.cap {
		p.log = p.log[1:] // drop oldest
	}
	p.log = append(p.log, CDPMessage{
		Direction: dir,
		Raw:       append(json.RawMessage(nil), data...),
	})
}

// Ensure Proxy implements cdp.WebSocketable.
var _ cdp.WebSocketable = (*Proxy)(nil)
