package scanner

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// ===================== BrowserPageOpener.OpenPage browser error =====================

// OpenPage's first failure mode is browser.Page() returning an error (lines
// 82-84). We connect a dedicated browser, close it, then call OpenPage so the
// underlying CDP page-creation request fails.
func TestBrowserPageOpener_OpenPage_BrowserClosed(t *testing.T) {
	path, _ := launcher.LookPath()
	u := launcher.New().Bin(path).Headless(true).NoSandbox(true).MustLaunch()
	b := rod.New().ControlURL(u).MustConnect()
	b.MustClose()

	op := &BrowserPageOpener{Browser: b}
	_, err := op.OpenPage("http://127.0.0.1:0/")
	if err == nil {
		t.Fatal("expected error creating a page on a closed browser")
	}
}

// ===================== ParseFieldFromElement direct =====================

// fieldFixtureHTML hosts inputs covering each ParseFieldFromElement branch:
//   - unnamed input (nil/empty name -> skipped)
//   - submit / button / hidden types (-> skipped)
//   - a normal text input and a textarea (-> kept)
const fieldFixtureHTML = `<!DOCTYPE html>
<html><body>
<form id="f" method="GET" action="/x">
  <input type="text" id="unnamed" />
  <input type="submit" name="go" value="Go" />
  <input type="button" name="btn" value="Btn" />
  <input type="hidden" name="csrf" value="tok" />
  <input type="text" name="username" />
  <textarea name="bio"></textarea>
</form>
</body></html>`

func navFixture(t *testing.T, html string) *rod.Page {
	t.Helper()
	page, err := testBrowser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		t.Fatalf("create page: %v", err)
	}
	if err := page.SetDocumentContent(html); err != nil {
		t.Fatalf("set content: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	return page
}

func TestParseFieldFromElement_SkipsAndKeeps(t *testing.T) {
	page := navFixture(t, fieldFixtureHTML)
	defer page.MustClose()

	// Unnamed input -> skipped (nil/empty name branch).
	unnamed, err := page.Element("#unnamed")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ParseFieldFromElement(unnamed); ok {
		t.Fatal("expected unnamed input to be skipped")
	}

	// submit / button / hidden -> skipped (type branch).
	for _, sel := range []string{`input[name=go]`, `input[name=btn]`, `input[name=csrf]`} {
		el, err := page.Element(sel)
		if err != nil {
			t.Fatalf("element %s: %v", sel, err)
		}
		if _, ok := ParseFieldFromElement(el); ok {
			t.Fatalf("expected %s to be skipped", sel)
		}
	}

	// Normal text input -> kept.
	username, err := page.Element(`input[name=username]`)
	if err != nil {
		t.Fatal(err)
	}
	field, ok := ParseFieldFromElement(username)
	if !ok {
		t.Fatal("expected username field to be kept")
	}
	if field.Name != "username" || field.Type != "text" || field.TagName != "input" {
		t.Fatalf("unexpected field: %+v", field)
	}

	// Textarea -> kept with tagName "textarea" and default type "text".
	bio, err := page.Element(`textarea[name=bio]`)
	if err != nil {
		t.Fatal(err)
	}
	bf, ok := ParseFieldFromElement(bio)
	if !ok {
		t.Fatal("expected textarea field to be kept")
	}
	if bf.TagName != "textarea" {
		t.Fatalf("expected tagName textarea, got %q", bf.TagName)
	}
}

// ===================== ParseFormFromElement Elements error =====================

func TestParseFormFromElement_ClosedPage(t *testing.T) {
	page := navFixture(t, fieldFixtureHTML)
	formEl, err := page.Element("#f")
	if err != nil {
		t.Fatal(err)
	}
	// Close the page so formEl.Elements(...) errors, exercising the early
	// return in ParseFormFromElement (it still returns the partial form).
	page.MustClose()

	form := ParseFormFromElement(formEl)
	// Attributes were read pre-close; method defaults to GET. No fields since
	// the Elements() query failed.
	if len(form.Fields) != 0 {
		t.Fatalf("expected no fields after Elements() error, got %d", len(form.Fields))
	}
}

// ===================== TestReflectedXSS BuildTestURL error =====================

func TestReflectedXSS_BuildURLError(t *testing.T) {
	// An invalid form action combined with an unparsable resolved URL makes
	// BuildTestURL fail, so TestReflectedXSS returns that error before opening
	// any page.
	form := Form{Action: "://bad-action", Method: "GET", Fields: []FormField{{Name: "q", Type: "text"}}}
	_, err := TestReflectedXSS(&failOpener{}, "://bad-base", form, form.Fields[0], "p")
	if err == nil {
		t.Fatal("expected BuildTestURL error to propagate")
	}
}

// ===================== TestStoredXSS extra branches =====================

// TestStoredXSS where the tested field's name does not match any real input
// on the page: every input gets "testdata" instead of the payload, so the
// payload is never stored and CheckDOMForPayload returns false. This exercises
// the trailing "return nil, nil" (line 318) while still walking the full
// input-fill / submit / re-open / check flow. The unique payload guarantees no
// accidental reflection from earlier entries.
func TestStoredXSS_FieldNameNoMatch(t *testing.T) {
	testServer.ResetStored()
	form := Form{Action: "/guestbook", Method: "POST", Fields: []FormField{
		{Name: "nonexistent_field_xyz", Type: "text"},
	}}
	uniquePayload := "<script>alert('UNIQUE_NOMATCH_9f3a')</script>"
	finding, err := TestStoredXSS(opener, testServer.URL()+"/guestbook", form, form.Fields[0], uniquePayload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if finding != nil {
		t.Fatalf("expected nil finding when payload field never matches, got %+v", finding)
	}
}

// checkPageFailOpener returns a real page on the first OpenPage call (the
// submit page) and fails on the second (the verification reopen), exercising
// the "checkPage open error -> return nil, nil" branch (lines 302-304).
type checkPageFailOpener struct {
	real  PageOpener
	mu    sync.Mutex
	calls int
}

func (o *checkPageFailOpener) OpenPage(targetURL string) (*rod.Page, error) {
	o.mu.Lock()
	o.calls++
	n := o.calls
	o.mu.Unlock()
	if n == 1 {
		return o.real.OpenPage(targetURL)
	}
	return nil, fmt.Errorf("mock: second open fails")
}

func TestStoredXSS_CheckPageOpenFails(t *testing.T) {
	testServer.ResetStored()
	op := &checkPageFailOpener{real: opener}
	form := Form{Action: "/guestbook", Method: "POST", Fields: []FormField{
		{Name: "name", Type: "text"},
		{Name: "comment", Type: "text"},
	}}
	finding, err := TestStoredXSS(op, testServer.URL()+"/guestbook", form, form.Fields[1], "<script>alert(1)</script>")
	if err != nil {
		t.Fatalf("expected nil error when checkPage open fails, got %v", err)
	}
	if finding != nil {
		t.Fatalf("expected nil finding when verification page can't open, got %+v", finding)
	}
}

// contentFormOpener returns a page whose content is a fixed POST form built via
// SetDocumentContent. The form deliberately contains an unnamed input (nil-name
// continue branch, lines 276-277) and a hidden input (type-skip continue
// branch, lines 280-281) alongside the real "comment" field, so the
// TestStoredXSS input loop walks every branch.
type contentFormOpener struct{}

const storedFormHTML = `<!DOCTYPE html>
<html><body>
<form id="g" method="POST" action="">
  <input type="text" />
  <input type="hidden" name="csrf" value="tok" />
  <input type="text" name="comment" />
  <button type="submit">Post</button>
</form>
</body></html>`

func (contentFormOpener) OpenPage(targetURL string) (*rod.Page, error) {
	page, err := testBrowser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, err
	}
	if err := page.SetDocumentContent(storedFormHTML); err != nil {
		page.Close()
		return nil, err
	}
	time.Sleep(80 * time.Millisecond)
	return page, nil
}

func TestStoredXSS_InputLoopBranches(t *testing.T) {
	form := Form{Action: "", Method: "POST", Fields: []FormField{{Name: "comment", Type: "text"}}}
	// Payload won't reflect back (content is static), so the call walks the
	// fill loop (hitting unnamed-input + hidden-input continues), submits, and
	// returns no finding.
	finding, err := TestStoredXSS(contentFormOpener{}, "http://example.com/g", form, form.Fields[0], "<b>x</b>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if finding != nil {
		t.Fatalf("expected nil finding on static form page, got %+v", finding)
	}
}

// ===================== ScanPage error-continue branches =====================

// scanFailAfterFirstOpener serves a real page for the initial DiscoverForms
// open, then fails every later open. This drives the TestReflectedXSS and
// TestStoredXSS error returns inside ScanPage's loops, hitting the
// "if err != nil { continue }" branches (lines 348-349, 359-360).
type scanFailAfterFirstOpener struct {
	real  PageOpener
	mu    sync.Mutex
	calls int
}

func (o *scanFailAfterFirstOpener) OpenPage(targetURL string) (*rod.Page, error) {
	o.mu.Lock()
	o.calls++
	n := o.calls
	o.mu.Unlock()
	if n == 1 {
		return o.real.OpenPage(targetURL)
	}
	return nil, fmt.Errorf("mock: open fails after discovery")
}

func TestScanPage_FieldTestsError(t *testing.T) {
	op := &scanFailAfterFirstOpener{real: opener}
	// Use the guestbook (POST form) so both reflected and stored field tests
	// run; every per-field open fails and is skipped via continue.
	result, err := ScanPage(op, testServer.URL()+"/guestbook")
	if err != nil {
		t.Fatalf("ScanPage should not error when field tests fail individually: %v", err)
	}
	if result.FormsFound == 0 {
		t.Fatal("expected forms discovered before field tests")
	}
	if result.FieldsTested == 0 {
		t.Fatal("expected fields counted even though their tests errored")
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings when all field opens fail, got %d", len(result.Findings))
	}
}

// closedPageDiscoverOpener returns an already-closed page so ScanPage's
// DiscoverForms call errors, exercising the "forms err -> return result, err"
// branch (lines 335-337).
type closedPageDiscoverOpener struct {
	real PageOpener
}

func (o *closedPageDiscoverOpener) OpenPage(targetURL string) (*rod.Page, error) {
	p, err := o.real.OpenPage(targetURL)
	if err != nil {
		return nil, err
	}
	p.MustClose()
	return p, nil
}

func TestScanPage_DiscoverFormsError(t *testing.T) {
	op := &closedPageDiscoverOpener{real: opener}
	result, err := ScanPage(op, testServer.URL()+"/search")
	if err == nil {
		t.Fatal("expected DiscoverForms error to propagate from ScanPage")
	}
	if result == nil {
		t.Fatal("expected non-nil result alongside the error")
	}
}
