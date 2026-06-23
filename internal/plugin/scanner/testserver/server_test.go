package testserver

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func newRunning(t *testing.T) *VulnServer {
	t.Helper()
	s, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	s.Start()
	t.Cleanup(s.Close)
	return s
}

func get(t *testing.T, rawURL string) (int, string) {
	t.Helper()
	resp, err := http.Get(rawURL)
	if err != nil {
		t.Fatalf("GET %s failed: %v", rawURL, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func TestServer_URLAndIndex(t *testing.T) {
	s := newRunning(t)

	if !strings.HasPrefix(s.URL(), "http://127.0.0.1:") {
		t.Fatalf("unexpected URL: %s", s.URL())
	}

	code, body := get(t, s.URL()+"/")
	if code != http.StatusOK {
		t.Fatalf("expected 200 on /, got %d", code)
	}
	if !strings.Contains(body, "Vulnerable Test Application") {
		t.Fatal("index page missing expected heading")
	}
}

func TestServer_IndexNotFoundForUnknownPath(t *testing.T) {
	s := newRunning(t)
	// "/" handler also serves unknown paths and must 404 them.
	code, _ := get(t, s.URL()+"/no-such-page")
	if code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown path, got %d", code)
	}
}

func TestServer_ReflectedSearch(t *testing.T) {
	s := newRunning(t)

	// Without query: form rendered, no results div.
	_, body := get(t, s.URL()+"/search")
	if !strings.Contains(body, `name="q"`) {
		t.Fatal("search form missing q input")
	}
	if strings.Contains(body, `id="results"`) {
		t.Fatal("expected no results div when q is empty")
	}

	// With query: unsanitized echo.
	_, body = get(t, s.URL()+"/search?q="+url.QueryEscape("<script>x</script>"))
	if !strings.Contains(body, "<script>x</script>") {
		t.Fatal("expected reflected unsanitized query")
	}
	if !strings.Contains(body, `id="results"`) {
		t.Fatal("expected results div when q is present")
	}
}

func TestServer_SafeSearch(t *testing.T) {
	s := newRunning(t)

	_, body := get(t, s.URL()+"/safe-search?q="+url.QueryEscape("<script>x</script>"))
	if strings.Contains(body, "<script>x</script>") {
		t.Fatal("safe-search must escape the query")
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Fatal("expected escaped query in safe-search output")
	}

	// Empty query branch.
	_, body = get(t, s.URL()+"/safe-search")
	if strings.Contains(body, `id="results"`) {
		t.Fatal("expected no results div for empty safe-search query")
	}
}

func TestServer_StoredGuestbook(t *testing.T) {
	s := newRunning(t)

	// GET shows the empty guestbook form.
	_, body := get(t, s.URL()+"/guestbook")
	if !strings.Contains(body, "Guestbook") {
		t.Fatal("guestbook page missing heading")
	}

	// POST a comment, then confirm it is stored and reflected unsanitized.
	form := url.Values{"name": {"alice"}, "comment": {"<b>hi</b>"}}
	resp, err := http.PostForm(s.URL()+"/guestbook", form)
	if err != nil {
		t.Fatalf("POST guestbook failed: %v", err)
	}
	postBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(postBody), "<b>hi</b>") {
		t.Fatal("expected stored comment reflected unsanitized after POST")
	}

	// A subsequent GET still shows the stored entry.
	_, body = get(t, s.URL()+"/guestbook")
	if !strings.Contains(body, "alice") {
		t.Fatal("expected stored name on subsequent GET")
	}

	// POST with both fields empty must NOT store anything new.
	before := len(s.stored)
	resp2, err := http.PostForm(s.URL()+"/guestbook", url.Values{"name": {""}, "comment": {""}})
	if err != nil {
		t.Fatalf("POST empty failed: %v", err)
	}
	resp2.Body.Close()
	if len(s.stored) != before {
		t.Fatalf("empty POST should not store an entry; before=%d after=%d", before, len(s.stored))
	}

	// ResetStored clears entries.
	s.ResetStored()
	if len(s.stored) != 0 {
		t.Fatalf("ResetStored should empty stored, got %d", len(s.stored))
	}
}

func TestServer_DOMXSS(t *testing.T) {
	s := newRunning(t)
	code, body := get(t, s.URL()+"/dom-xss")
	if code != http.StatusOK {
		t.Fatalf("expected 200 on /dom-xss, got %d", code)
	}
	if !strings.Contains(body, "URLSearchParams") {
		t.Fatal("expected DOM XSS sink script in page")
	}
}

func TestServer_Contact(t *testing.T) {
	s := newRunning(t)

	// No subject: form only.
	_, body := get(t, s.URL()+"/contact")
	if !strings.Contains(body, `name="subject"`) {
		t.Fatal("contact form missing subject input")
	}
	if strings.Contains(body, `id="confirm"`) {
		t.Fatal("expected no confirm block without subject")
	}

	// With subject (unescaped) and email (escaped).
	_, body = get(t, s.URL()+"/contact?subject="+url.QueryEscape("<i>s</i>")+"&email="+url.QueryEscape("<e>@x"))
	if !strings.Contains(body, "<i>s</i>") {
		t.Fatal("expected unescaped subject reflected")
	}
	if !strings.Contains(body, `id="confirm"`) {
		t.Fatal("expected confirm block when subject present")
	}
	if strings.Contains(body, "<e>@x") {
		t.Fatal("expected email value to be escaped")
	}
}

func TestServer_About(t *testing.T) {
	s := newRunning(t)
	code, body := get(t, s.URL()+"/about")
	if code != http.StatusOK {
		t.Fatalf("expected 200 on /about, got %d", code)
	}
	if !strings.Contains(body, "About Us") {
		t.Fatal("about page missing heading")
	}
}

func TestServer_Close(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	s.Start()
	addr := s.URL()
	s.Close()

	// After Close the listener is shut; a request should fail to connect.
	if _, err := http.Get(addr + "/"); err == nil {
		t.Fatal("expected request to fail after Close()")
	}
}
