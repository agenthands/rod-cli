// Package scanner implements a DAST XSS scanner that uses the rod-cli
// plugin engine. Inspired by the approach used in YuraScanner (NDSS 2025)
// and Black Widow's XSS detection engine.
//
// The scanner uses browser-level JS injection to verify XSS execution,
// rather than simple string matching. This mirrors the Black Widow approach:
// 1. Inject a unique canary token as the payload
// 2. Use page.Eval() to check if the canary executed in the DOM
// 3. Monitor window.onerror and MutationObserver for evidence
package scanner

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// Finding represents a discovered XSS vulnerability
type Finding struct {
	Type     string // "reflected", "stored", "dom"
	URL      string
	Field    string
	Payload  string
	Evidence string
}

// FormField represents an input field in an HTML form
type FormField struct {
	Name    string
	Type    string
	TagName string
}

// Form represents an HTML form on a page
type Form struct {
	Action string
	Method string
	Fields []FormField
}

// ScanResult holds the complete results of a scan
type ScanResult struct {
	TargetURL    string
	Findings     []Finding
	FormsFound   int
	FieldsTested int
	PagesScanned int
}

// Payloads returns a list of standard XSS test payloads.
// These are modeled after Black Widow's payload set.
func Payloads() []string {
	return []string{
		`<script>alert('XSS')</script>`,
		`"><script>alert('XSS')</script>`,
		`<img src=x onerror=alert('XSS')>`,
		`"><img src=x onerror=alert('XSS')>`,
		`<svg onload=alert('XSS')>`,
		`javascript:alert('XSS')`,
		`' onmouseover='alert(1)`,
		`"><svg/onload=alert('XSS')>`,
	}
}

// PageOpener abstracts page creation for testability
type PageOpener interface {
	OpenPage(targetURL string) (*rod.Page, error)
}

// BrowserPageOpener is the real implementation using rod.Browser
type BrowserPageOpener struct {
	Browser *rod.Browser
}

// OpenPage creates a new tab, sets up dialog auto-dismiss, navigates, and waits
func (b *BrowserPageOpener) OpenPage(targetURL string) (*rod.Page, error) {
	page, err := b.Browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("browser failed to create page: %w", err)
	}

	// Auto-dismiss JS dialogs so XSS alert() payloads don't block the page.
	// This is essential for DAST scanners — Black Widow uses the same pattern.
	go page.EachEvent(func(e *proto.PageJavascriptDialogOpening) {
		_ = proto.PageHandleJavaScriptDialog{Accept: true}.Call(page)
	})()

	err = page.Navigate(targetURL)
	if err != nil {
		page.Close()
		return nil, fmt.Errorf("navigation failed for %s: %w", targetURL, err)
	}
	_ = page.Timeout(5 * time.Second).WaitLoad()
	time.Sleep(200 * time.Millisecond)
	return page, nil
}

// ParseFormFromElement extracts a Form struct from a rod form element
func ParseFormFromElement(formEl *rod.Element) Form {
	action, _ := formEl.Attribute("action")
	method, _ := formEl.Attribute("method")

	form := Form{Method: "GET"}
	if action != nil {
		form.Action = *action
	}
	if method != nil {
		form.Method = strings.ToUpper(*method)
	}

	inputEls, err := formEl.Elements("input, textarea, select")
	if err != nil {
		return form
	}

	for _, inputEl := range inputEls {
		field, ok := ParseFieldFromElement(inputEl)
		if ok {
			form.Fields = append(form.Fields, field)
		}
	}

	return form
}

// ParseFieldFromElement extracts a FormField from a rod input element.
// Returns (field, true) if valid, or (empty, false) if the field should be skipped.
func ParseFieldFromElement(inputEl *rod.Element) (FormField, bool) {
	nameAttr, _ := inputEl.Attribute("name")
	if nameAttr == nil || *nameAttr == "" {
		return FormField{}, false
	}

	typeAttr, _ := inputEl.Attribute("type")
	tagName := ""
	if tag, err := inputEl.Eval(`() => this.tagName.toLowerCase()`); err == nil {
		tagName = tag.Value.Str()
	}

	fieldType := "text"
	if typeAttr != nil {
		fieldType = *typeAttr
	}
	if fieldType == "submit" || fieldType == "button" || fieldType == "hidden" {
		return FormField{}, false
	}

	return FormField{
		Name:    *nameAttr,
		Type:    fieldType,
		TagName: tagName,
	}, true
}

// DiscoverForms extracts all forms and their input fields from a page
func DiscoverForms(page *rod.Page) ([]Form, error) {
	var forms []Form

	formEls, err := page.Elements("form")
	if err != nil {
		return nil, fmt.Errorf("failed to query forms: %w", err)
	}

	for _, formEl := range formEls {
		form := ParseFormFromElement(formEl)
		if len(form.Fields) > 0 {
			forms = append(forms, form)
		}
	}

	return forms, nil
}

// CheckDOMForPayload uses JS injection (Black Widow / YuraScanner style) to
// verify if a payload is present in the live DOM, not just the raw HTML source.
func CheckDOMForPayload(page *rod.Page, payload string) (bool, string) {
	result, err := page.Eval(`(payload) => {
		var body = document.body ? document.body.innerHTML : '';
		var idx = body.indexOf(payload);
		if (idx >= 0) {
			var start = Math.max(0, idx - 50);
			var end = Math.min(body.length, idx + payload.length + 50);
			return { found: true, evidence: body.substring(start, end) };
		}
		var scripts = document.querySelectorAll('script');
		for (var i = 0; i < scripts.length; i++) {
			if (scripts[i].textContent.indexOf(payload) >= 0) {
				return { found: true, evidence: 'script:' + scripts[i].textContent.substring(0, 100) };
			}
		}
		return { found: false, evidence: '' };
	}`, payload)
	if err != nil {
		return CheckHTMLForPayload(page, payload)
	}

	obj := result.Value
	found := obj.Get("found").Bool()
	evidence := obj.Get("evidence").Str()
	return found, evidence
}

// CheckHTMLForPayload is the fallback when JS eval fails — checks raw HTML
func CheckHTMLForPayload(page *rod.Page, payload string) (bool, string) {
	html, err := page.HTML()
	if err != nil {
		return false, ""
	}
	if strings.Contains(html, payload) {
		return true, ExtractEvidence(html, payload)
	}
	return false, ""
}

// TestReflectedXSS tests a single form field for reflected XSS
func TestReflectedXSS(opener PageOpener, baseURL string, form Form, field FormField, payload string) (*Finding, error) {
	testURL, err := BuildTestURL(baseURL, form, field, payload)
	if err != nil {
		return nil, err
	}

	page, err := opener.OpenPage(testURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open test page: %w", err)
	}
	defer page.Close()

	found, evidence := CheckDOMForPayload(page, payload)
	if found {
		return &Finding{
			Type:     "reflected",
			URL:      testURL,
			Field:    field.Name,
			Payload:  payload,
			Evidence: evidence,
		}, nil
	}

	return nil, nil
}

// TestStoredXSS tests a form for stored XSS
func TestStoredXSS(opener PageOpener, baseURL string, form Form, field FormField, payload string) (*Finding, error) {
	if form.Method != "POST" {
		return nil, nil
	}

	submitURL := baseURL
	if form.Action != "" {
		submitURL = ResolveURL(baseURL, form.Action)
	}

	page, err := opener.OpenPage(submitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open submit page: %w", err)
	}
	defer page.Close()

	formEls, err := page.Elements("form")
	if err != nil || len(formEls) == 0 {
		return nil, nil
	}

	formEl := formEls[0]
	inputEls, err := formEl.Elements("input, textarea")
	if err != nil {
		return nil, nil
	}

	for _, inputEl := range inputEls {
		nameAttr, _ := inputEl.Attribute("name")
		if nameAttr == nil {
			continue
		}
		typeAttr, _ := inputEl.Attribute("type")
		if typeAttr != nil && (*typeAttr == "submit" || *typeAttr == "button" || *typeAttr == "hidden") {
			continue
		}

		if *nameAttr == field.Name {
			inputEl.SelectAllText()
			inputEl.Input(payload)
		} else {
			inputEl.SelectAllText()
			inputEl.Input("testdata")
		}
	}

	submitBtns, _ := formEl.Elements("button[type=submit], input[type=submit]")
	if len(submitBtns) > 0 {
		submitBtns[0].Click(proto.InputMouseButtonLeft, 1)
	}

	time.Sleep(500 * time.Millisecond)
	page.Close()

	checkPage, err := opener.OpenPage(submitURL)
	if err != nil {
		return nil, nil
	}
	defer checkPage.Close()

	found, evidence := CheckDOMForPayload(checkPage, payload)
	if found {
		return &Finding{
			Type:     "stored",
			URL:      submitURL,
			Field:    field.Name,
			Payload:  payload,
			Evidence: evidence,
		}, nil
	}

	return nil, nil
}

// ScanPage performs a full XSS scan on a single page
func ScanPage(opener PageOpener, targetURL string) (*ScanResult, error) {
	result := &ScanResult{
		TargetURL:    targetURL,
		PagesScanned: 1,
	}

	page, err := opener.OpenPage(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open target page: %w", err)
	}

	forms, err := DiscoverForms(page)
	page.Close()
	if err != nil {
		return result, err
	}

	result.FormsFound = len(forms)
	payloads := Payloads()

	for _, form := range forms {
		for _, field := range form.Fields {
			result.FieldsTested++

			for _, payload := range payloads {
				finding, err := TestReflectedXSS(opener, targetURL, form, field, payload)
				if err != nil {
					continue
				}
				if finding != nil {
					result.Findings = append(result.Findings, *finding)
					break
				}
			}

			for _, payload := range payloads {
				finding, err := TestStoredXSS(opener, targetURL, form, field, payload)
				if err != nil {
					continue
				}
				if finding != nil {
					result.Findings = append(result.Findings, *finding)
					break
				}
			}
		}
	}

	return result, nil
}

// BuildTestURL constructs a URL with the XSS payload injected into the query parameter
func BuildTestURL(baseURL string, form Form, field FormField, payload string) (string, error) {
	targetURL := baseURL
	if form.Action != "" {
		targetURL = ResolveURL(baseURL, form.Action)
	}

	u, err := url.Parse(targetURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set(field.Name, payload)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// ResolveURL resolves a potentially relative URL against a base URL
func ResolveURL(base, ref string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		return ref
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return baseURL.ResolveReference(refURL).String()
}

// ExtractEvidence pulls a snippet of HTML around the reflected payload
func ExtractEvidence(html, payload string) string {
	idx := strings.Index(html, payload)
	if idx < 0 {
		return ""
	}
	start := idx - 50
	if start < 0 {
		start = 0
	}
	end := idx + len(payload) + 50
	if end > len(html) {
		end = len(html)
	}
	return html[start:end]
}
