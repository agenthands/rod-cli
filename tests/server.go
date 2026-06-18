package tests

import (
	"net/http"
	"net/http/httptest"
)

// SetupTestServer spins up a local web server with various DOM complexities
// (forms, dialogs, canvases, storage scripts) to act as our test fixture.
func SetupTestServer() *httptest.Server {
	mux := http.NewServeMux()

	// 1. Basic Navigation & Links
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
			<h1 id="header">Home</h1>
			<a id="link-forms" href="/forms">Forms</a>
			<a id="link-dialogs" href="/dialogs">Dialogs</a>
			<a id="link-storage" href="/storage">Storage</a>
			<a id="link-page1" href="/page1">Page 1</a>
		</body></html>`))
	})

	mux.HandleFunc("/page1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><h1 id="p1">Page 1</h1><a href="/page2">Page 2</a></body></html>`))
	})

	mux.HandleFunc("/page2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><h1 id="p2">Page 2</h1></body></html>`))
	})

	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		// No sleep for test stability, but simulate a large DOM or delayed rendering
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><div id="slow-div">Slow</div></body></html>`))
	})

	// 2. Forms & Inputs
	mux.HandleFunc("/forms", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
			<input type="text" id="username" />
			<input type="password" id="password" />
			<select id="dropdown"><option value="1">One</option><option value="2">Two</option></select>
			<button id="submit">Submit</button>
			<div id="output"></div>
			<script>
				document.getElementById('submit').addEventListener('click', () => {
					document.getElementById('output').innerText = 'Submitted ' + document.getElementById('username').value;
				});
			</script>
		</body></html>`))
	})

	// 3. Dialogs
	mux.HandleFunc("/dialogs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
			<button id="alertBtn" onclick="alert('Hello')">Alert</button>
			<button id="confirmBtn" onclick="window.res = confirm('Accept?')">Confirm</button>
			<div id="res"></div>
		</body></html>`))
	})

	// 4. Storage & Networking
	mux.HandleFunc("/storage", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
			<script>
				document.cookie = "testcookie=123";
				localStorage.setItem("localtest", "abc");
				sessionStorage.setItem("sessiontest", "xyz");
			</script>
			<div id="storage-ready">Ready</div>
		</body></html>`))
	})

	return httptest.NewServer(mux)
}
