package scanner

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/agenthands/rod-cli/internal/plugin/scanner/testserver"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

var testBrowser *rod.Browser
var testServer *testserver.VulnServer

func TestMain(m *testing.M) {
	// Start vulnerable test server
	var err error
	testServer, err = testserver.New()
	if err != nil {
		panic("failed to start test server: " + err.Error())
	}
	testServer.Start()
	defer testServer.Close()

	// Launch headless browser
	path, _ := launcher.LookPath()
	u := launcher.New().Bin(path).Headless(true).NoSandbox(true).MustLaunch()
	testBrowser = rod.New().ControlURL(u).MustConnect()
	defer testBrowser.MustClose()

	os.Exit(m.Run())
}

func openPage(t *testing.T, targetURL string) *rod.Page {
	t.Helper()
	page, err := testBrowser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		t.Fatalf("failed to create page: %v", err)
	}
	err = page.Navigate(targetURL)
	if err != nil {
		t.Fatalf("failed to navigate to %s: %v", targetURL, err)
	}
	_ = page.Timeout(5 * time.Second).WaitLoad()
	time.Sleep(200 * time.Millisecond)
	return page
}

// --- Positive Tests ---

func TestDiscoverForms_ReflectedPage(t *testing.T) {
	page := openPage(t, testServer.URL()+"/search")
	defer page.MustClose()

	forms, err := DiscoverForms(page)
	if err != nil {
		t.Fatalf("DiscoverForms failed: %v", err)
	}
	if len(forms) != 1 {
		t.Fatalf("expected 1 form, got %d", len(forms))
	}
	if len(forms[0].Fields) == 0 {
		t.Fatal("expected at least 1 field in search form")
	}

	found := false
	for _, f := range forms[0].Fields {
		if f.Name == "q" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected to find field 'q' in search form")
	}
}

func TestDiscoverForms_GuestbookPage(t *testing.T) {
	page := openPage(t, testServer.URL()+"/guestbook")
	defer page.MustClose()

	forms, err := DiscoverForms(page)
	if err != nil {
		t.Fatalf("DiscoverForms failed: %v", err)
	}
	if len(forms) != 1 {
		t.Fatalf("expected 1 form, got %d", len(forms))
	}
	if forms[0].Method != "POST" {
		t.Fatalf("expected POST method, got %s", forms[0].Method)
	}

	fieldNames := map[string]bool{}
	for _, f := range forms[0].Fields {
		fieldNames[f.Name] = true
	}
	if !fieldNames["name"] || !fieldNames["comment"] {
		t.Fatalf("expected fields 'name' and 'comment', got %v", fieldNames)
	}
}

func TestDiscoverForms_MultiFieldPage(t *testing.T) {
	page := openPage(t, testServer.URL()+"/contact")
	defer page.MustClose()

	forms, err := DiscoverForms(page)
	if err != nil {
		t.Fatalf("DiscoverForms failed: %v", err)
	}
	if len(forms) == 0 {
		t.Fatal("expected at least 1 form on contact page")
	}
	if len(forms[0].Fields) < 2 {
		t.Fatalf("expected at least 2 fields, got %d", len(forms[0].Fields))
	}
}

func TestDiscoverForms_NoFormsPage(t *testing.T) {
	page := openPage(t, testServer.URL()+"/about")
	defer page.MustClose()

	forms, err := DiscoverForms(page)
	if err != nil {
		t.Fatalf("DiscoverForms failed: %v", err)
	}
	if len(forms) != 0 {
		t.Fatalf("expected 0 forms on about page, got %d", len(forms))
	}
}

func TestReflectedXSS_Vulnerable(t *testing.T) {
	payload := `<script>alert('XSS')</script>`
	form := Form{Action: "/search", Method: "GET", Fields: []FormField{{Name: "q", Type: "text"}}}
	field := form.Fields[0]

	finding, err := TestReflectedXSS(testBrowser, testServer.URL(), form, field, payload)
	if err != nil {
		t.Fatalf("TestReflectedXSS failed: %v", err)
	}
	if finding == nil {
		t.Fatal("expected to find reflected XSS on /search, got nil")
	}
	if finding.Type != "reflected" {
		t.Fatalf("expected type 'reflected', got '%s'", finding.Type)
	}
	if finding.Payload != payload {
		t.Fatalf("expected payload to match, got '%s'", finding.Payload)
	}
	if finding.Evidence == "" {
		t.Fatal("expected non-empty evidence")
	}
}

func TestReflectedXSS_Safe(t *testing.T) {
	payload := `<script>alert('XSS')</script>`
	form := Form{Action: "/safe-search", Method: "GET", Fields: []FormField{{Name: "q", Type: "text"}}}
	field := form.Fields[0]

	finding, err := TestReflectedXSS(testBrowser, testServer.URL(), form, field, payload)
	if err != nil {
		t.Fatalf("TestReflectedXSS failed: %v", err)
	}
	if finding != nil {
		t.Fatalf("expected no finding on safe page, got: %+v", finding)
	}
}

func TestStoredXSS_Vulnerable(t *testing.T) {
	testServer.ResetStored()

	payload := `<script>alert('XSS')</script>`
	form := Form{Action: "/guestbook", Method: "POST", Fields: []FormField{
		{Name: "name", Type: "text"},
		{Name: "comment", Type: "text", TagName: "textarea"},
	}}

	finding, err := TestStoredXSS(testBrowser, testServer.URL()+"/guestbook", form, form.Fields[1], payload)
	if err != nil {
		t.Fatalf("TestStoredXSS failed: %v", err)
	}
	if finding == nil {
		t.Fatal("expected to find stored XSS on /guestbook, got nil")
	}
	if finding.Type != "stored" {
		t.Fatalf("expected type 'stored', got '%s'", finding.Type)
	}
}

func TestStoredXSS_GetFormSkipped(t *testing.T) {
	form := Form{Action: "/search", Method: "GET", Fields: []FormField{{Name: "q", Type: "text"}}}
	field := form.Fields[0]

	finding, err := TestStoredXSS(testBrowser, testServer.URL(), form, field, "<script>alert(1)</script>")
	if err != nil {
		t.Fatalf("TestStoredXSS failed: %v", err)
	}
	if finding != nil {
		t.Fatal("expected nil finding for GET form stored XSS test")
	}
}

func TestScanPage_ReflectedPage(t *testing.T) {
	result, err := ScanPage(testBrowser, testServer.URL()+"/search")
	if err != nil {
		t.Fatalf("ScanPage failed: %v", err)
	}
	if result.FormsFound == 0 {
		t.Fatal("expected at least 1 form")
	}
	if result.FieldsTested == 0 {
		t.Fatal("expected at least 1 field tested")
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected at least 1 finding on vulnerable /search page")
	}

	hasReflected := false
	for _, f := range result.Findings {
		if f.Type == "reflected" {
			hasReflected = true
			break
		}
	}
	if !hasReflected {
		t.Fatal("expected reflected XSS finding")
	}
}

func TestScanPage_SafePage(t *testing.T) {
	result, err := ScanPage(testBrowser, testServer.URL()+"/safe-search")
	if err != nil {
		t.Fatalf("ScanPage failed: %v", err)
	}
	// The safe page escapes HTML special chars, so script/img/svg payloads
	// should not be found. However, payloads without special chars (like
	// "javascript:alert('XSS')") may appear in input value attributes.
	for _, f := range result.Findings {
		if strings.Contains(f.Payload, "<") || strings.Contains(f.Payload, ">") {
			t.Fatalf("found HTML-injection finding on safe page: %+v", f)
		}
	}
}

func TestScanPage_NoFormsPage(t *testing.T) {
	result, err := ScanPage(testBrowser, testServer.URL()+"/about")
	if err != nil {
		t.Fatalf("ScanPage failed: %v", err)
	}
	if result.FormsFound != 0 {
		t.Fatalf("expected 0 forms, got %d", result.FormsFound)
	}
	if result.FieldsTested != 0 {
		t.Fatalf("expected 0 fields tested, got %d", result.FieldsTested)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(result.Findings))
	}
}

func TestScanPage_ContactPage(t *testing.T) {
	result, err := ScanPage(testBrowser, testServer.URL()+"/contact")
	if err != nil {
		t.Fatalf("ScanPage failed: %v", err)
	}

	hasSubjectFinding := false
	for _, f := range result.Findings {
		if f.Field == "subject" && f.Type == "reflected" {
			hasSubjectFinding = true
		}
	}
	if !hasSubjectFinding {
		t.Fatal("expected reflected XSS finding for 'subject' field on contact page")
	}
}

// --- Negative Tests ---

func TestReflectedXSS_EmptyPayload(t *testing.T) {
	form := Form{Action: "/search", Method: "GET", Fields: []FormField{{Name: "q", Type: "text"}}}
	field := form.Fields[0]

	finding, err := TestReflectedXSS(testBrowser, testServer.URL(), form, field, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = finding // should not crash
}

func TestReflectedXSS_InvalidURL(t *testing.T) {
	form := Form{Action: "/search", Method: "GET", Fields: []FormField{{Name: "q", Type: "text"}}}
	field := form.Fields[0]

	_, err := TestReflectedXSS(testBrowser, "://invalid-url", form, field, "<script>alert(1)</script>")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestStoredXSS_EmptyPayload(t *testing.T) {
	testServer.ResetStored()
	form := Form{Action: "/guestbook", Method: "POST", Fields: []FormField{
		{Name: "name", Type: "text"},
		{Name: "comment", Type: "text"},
	}}
	field := form.Fields[1]

	_, err := TestStoredXSS(testBrowser, testServer.URL()+"/guestbook", form, field, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Edge Case Tests ---

func TestPayloads_NotEmpty(t *testing.T) {
	p := Payloads()
	if len(p) == 0 {
		t.Fatal("Payloads() should return at least one payload")
	}
	for i, payload := range p {
		if payload == "" {
			t.Fatalf("payload at index %d is empty", i)
		}
	}
}

func TestBuildTestURL_SimpleCase(t *testing.T) {
	form := Form{Action: "/search", Method: "GET", Fields: []FormField{{Name: "q", Type: "text"}}}
	field := form.Fields[0]
	payload := "<script>alert(1)</script>"

	u, err := buildTestURL("http://localhost:8080", form, field, payload)
	if err != nil {
		t.Fatalf("buildTestURL failed: %v", err)
	}
	if u == "" {
		t.Fatal("expected non-empty URL")
	}
}

func TestBuildTestURL_EmptyAction(t *testing.T) {
	form := Form{Action: "", Method: "GET", Fields: []FormField{{Name: "q", Type: "text"}}}
	field := form.Fields[0]

	u, err := buildTestURL("http://localhost:8080/page", form, field, "test")
	if err != nil {
		t.Fatalf("buildTestURL failed: %v", err)
	}
	if u == "" {
		t.Fatal("expected non-empty URL")
	}
}

func TestResolveURL_Absolute(t *testing.T) {
	result := resolveURL("http://example.com/page", "http://other.com/target")
	if result != "http://other.com/target" {
		t.Fatalf("expected absolute URL to pass through, got %s", result)
	}
}

func TestResolveURL_Relative(t *testing.T) {
	result := resolveURL("http://example.com/dir/page", "/search")
	if result != "http://example.com/search" {
		t.Fatalf("expected resolved URL, got %s", result)
	}
}

func TestExtractEvidence_Found(t *testing.T) {
	html := `<div>prefix <script>alert('XSS')</script> suffix</div>`
	evidence := extractEvidence(html, `<script>alert('XSS')</script>`)
	if evidence == "" {
		t.Fatal("expected non-empty evidence")
	}
	if len(evidence) > len(html) {
		t.Fatal("evidence should not be longer than the full HTML")
	}
}

func TestExtractEvidence_NotFound(t *testing.T) {
	html := `<div>safe content</div>`
	evidence := extractEvidence(html, `<script>alert('XSS')</script>`)
	if evidence != "" {
		t.Fatalf("expected empty evidence, got '%s'", evidence)
	}
}

func TestExtractEvidence_AtStart(t *testing.T) {
	html := `<script>alert('XSS')</script> trailing content`
	evidence := extractEvidence(html, `<script>alert('XSS')</script>`)
	if evidence == "" {
		t.Fatal("expected non-empty evidence when payload is at start")
	}
}

func TestExtractEvidence_AtEnd(t *testing.T) {
	html := `leading content <script>alert('XSS')</script>`
	evidence := extractEvidence(html, `<script>alert('XSS')</script>`)
	if evidence == "" {
		t.Fatal("expected non-empty evidence when payload is at end")
	}
}

func TestScanResult_EmptyInitialization(t *testing.T) {
	result := &ScanResult{TargetURL: "http://example.com"}
	if result.FormsFound != 0 {
		t.Fatal("expected FormsFound to be 0")
	}
	if result.FieldsTested != 0 {
		t.Fatal("expected FieldsTested to be 0")
	}
	if len(result.Findings) != 0 {
		t.Fatal("expected no findings")
	}
}

func TestVulnServer_ResetStored(t *testing.T) {
	testServer.ResetStored()

	page := openPage(t, testServer.URL()+"/guestbook")
	defer page.MustClose()

	html, _ := page.HTML()
	for _, p := range Payloads() {
		if indexOf(html, p) >= 0 {
			t.Fatal("expected no payloads after reset")
		}
	}
}

// --- Helpers ---

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
