// Package testserver provides an intentionally vulnerable web application
// for testing DAST scanner plugins. This server contains known XSS
// vulnerabilities that mirror real-world patterns.
//
// WARNING: This server is for testing purposes ONLY. Never expose it
// to untrusted networks.
package testserver

import (
	"fmt"
	"html"
	"net"
	"net/http"
	"sync"
)

// StoredEntry represents a stored comment in the guestbook
type StoredEntry struct {
	Name    string
	Comment string
}

// VulnServer is an intentionally vulnerable web server for DAST testing
type VulnServer struct {
	listener net.Listener
	mux      *http.ServeMux
	mu       sync.Mutex
	stored   []StoredEntry
}

// New creates a new vulnerable test server on a random port
func New() (*VulnServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	s := &VulnServer{
		listener: listener,
		mux:      http.NewServeMux(),
		stored:   []StoredEntry{},
	}
	s.registerRoutes()
	return s, nil
}

// Start begins serving HTTP requests in the background
func (s *VulnServer) Start() {
	go http.Serve(s.listener, s.mux)
}

// Close shuts down the server
func (s *VulnServer) Close() {
	s.listener.Close()
}

// URL returns the base URL of the running server
func (s *VulnServer) URL() string {
	return fmt.Sprintf("http://%s", s.listener.Addr().String())
}

func (s *VulnServer) registerRoutes() {
	// Landing page with links to all test pages
	s.mux.HandleFunc("/", s.handleIndex)

	// Reflected XSS: search query echoed without sanitization
	s.mux.HandleFunc("/search", s.handleReflectedXSS)

	// Stored XSS: guestbook that stores and displays comments without sanitization
	s.mux.HandleFunc("/guestbook", s.handleStoredXSS)

	// Safe page: properly escaped output
	s.mux.HandleFunc("/safe-search", s.handleSafeSearch)

	// DOM-based XSS: page that reads from URL fragment
	s.mux.HandleFunc("/dom-xss", s.handleDOMXSS)

	// Multiple input form: contact page with several fields
	s.mux.HandleFunc("/contact", s.handleContact)

	// No-form page: static content with no inputs
	s.mux.HandleFunc("/about", s.handleAbout)
}

func (s *VulnServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>Test Application</title></head>
<body>
<h1>Vulnerable Test Application</h1>
<ul>
  <li><a href="/search">Search (Reflected XSS)</a></li>
  <li><a href="/guestbook">Guestbook (Stored XSS)</a></li>
  <li><a href="/safe-search">Safe Search (Escaped)</a></li>
  <li><a href="/dom-xss">DOM XSS</a></li>
  <li><a href="/contact">Contact Form</a></li>
  <li><a href="/about">About (No Forms)</a></li>
</ul>
</body></html>`)
}

// handleReflectedXSS echoes user input without sanitization (reflected XSS)
func (s *VulnServer) handleReflectedXSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	query := r.URL.Query().Get("q")

	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>Search</title></head>
<body>
<h1>Search</h1>
<form method="GET" action="/search">
  <input type="text" name="q" value="%s" />
  <button type="submit">Search</button>
</form>`, query)

	if query != "" {
		// VULNERABLE: query is injected directly into HTML without escaping
		fmt.Fprintf(w, `<div id="results"><p>Results for: %s</p><p>No results found.</p></div>`, query)
	}

	fmt.Fprint(w, `</body></html>`)
}

// handleStoredXSS stores and displays comments without sanitization (stored XSS)
func (s *VulnServer) handleStoredXSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if r.Method == http.MethodPost {
		r.ParseForm()
		name := r.FormValue("name")
		comment := r.FormValue("comment")
		if name != "" || comment != "" {
			s.mu.Lock()
			s.stored = append(s.stored, StoredEntry{Name: name, Comment: comment})
			s.mu.Unlock()
		}
	}

	fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>Guestbook</title></head>
<body>
<h1>Guestbook</h1>
<form method="POST" action="/guestbook">
  <input type="text" name="name" placeholder="Your name" />
  <textarea name="comment" placeholder="Your comment"></textarea>
  <button type="submit">Post</button>
</form>
<h2>Comments</h2>
<div id="comments">`)

	s.mu.Lock()
	for _, entry := range s.stored {
		// VULNERABLE: stored content is displayed without escaping
		fmt.Fprintf(w, `<div class="comment"><strong>%s</strong>: %s</div>`, entry.Name, entry.Comment)
	}
	s.mu.Unlock()

	fmt.Fprint(w, `</div></body></html>`)
}

// handleSafeSearch properly escapes user input (NOT vulnerable)
func (s *VulnServer) handleSafeSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	query := r.URL.Query().Get("q")

	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>Safe Search</title></head>
<body>
<h1>Safe Search</h1>
<form method="GET" action="/safe-search">
  <input type="text" name="q" value="%s" />
  <button type="submit">Search</button>
</form>`, html.EscapeString(query))

	if query != "" {
		// SAFE: query is properly escaped
		fmt.Fprintf(w, `<div id="results"><p>Results for: %s</p><p>No results found.</p></div>`, html.EscapeString(query))
	}

	fmt.Fprint(w, `</body></html>`)
}

// handleDOMXSS serves a page vulnerable to DOM-based XSS
func (s *VulnServer) handleDOMXSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>DOM XSS</title></head>
<body>
<h1>Welcome</h1>
<div id="greeting"></div>
<script>
  var name = new URLSearchParams(window.location.search).get('name');
  if (name) {
    document.getElementById('greeting').innerHTML = 'Hello, ' + name + '!';
  }
</script>
</body></html>`)
}

// handleContact renders a multi-field form (reflected XSS on 'subject')
func (s *VulnServer) handleContact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	subject := r.URL.Query().Get("subject")
	email := r.URL.Query().Get("email")

	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>Contact</title></head>
<body>
<h1>Contact Us</h1>
<form method="GET" action="/contact">
  <input type="text" name="subject" placeholder="Subject" value="%s" />
  <input type="email" name="email" placeholder="Email" value="%s" />
  <textarea name="message" placeholder="Message"></textarea>
  <button type="submit">Send</button>
</form>`, subject, html.EscapeString(email))

	if subject != "" {
		// VULNERABLE: subject is not escaped
		fmt.Fprintf(w, `<div id="confirm"><p>Subject: %s</p><p>Thank you for your message!</p></div>`, subject)
	}

	fmt.Fprint(w, `</body></html>`)
}

// handleAbout serves a static page with no forms
func (s *VulnServer) handleAbout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>About</title></head>
<body>
<h1>About Us</h1>
<p>This is a test application for DAST scanning.</p>
</body></html>`)
}

// ResetStored clears all stored XSS entries (useful between tests)
func (s *VulnServer) ResetStored() {
	s.mu.Lock()
	s.stored = nil
	s.mu.Unlock()
}
