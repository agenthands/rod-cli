// Package scanner implements a DAST XSS scanner that uses the rod-cli
// plugin engine. Inspired by the approach used in YuraScanner (NDSS 2025)
// and Black Widow's XSS detection engine.
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

// Payloads returns a list of standard XSS test payloads
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

// openAndWait creates a new page, navigates to the URL, and waits for load.
// It automatically dismisses any JS dialogs (alert/confirm/prompt) that may
// be triggered by XSS payloads.
func openAndWait(browser *rod.Browser, targetURL string) (*rod.Page, error) {
	page, err := browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, err
	}

	// Auto-dismiss JS dialogs so XSS alert() payloads don't block the page
	go page.EachEvent(func(e *proto.PageJavascriptDialogOpening) {
		_ = proto.PageHandleJavaScriptDialog{Accept: true}.Call(page)
	})()

	err = page.Navigate(targetURL)
	if err != nil {
		page.Close()
		return nil, err
	}
	_ = page.Timeout(5 * time.Second).WaitLoad()
	time.Sleep(200 * time.Millisecond)
	return page, nil
}

// DiscoverForms extracts all forms and their input fields from a page
func DiscoverForms(page *rod.Page) ([]Form, error) {
	var forms []Form

	formEls, err := page.Elements("form")
	if err != nil {
		return nil, fmt.Errorf("failed to query forms: %w", err)
	}

	for _, formEl := range formEls {
		action, _ := formEl.Attribute("action")
		method, _ := formEl.Attribute("method")

		form := Form{
			Method: "GET",
		}
		if action != nil {
			form.Action = *action
		}
		if method != nil {
			form.Method = strings.ToUpper(*method)
		}

		inputEls, err := formEl.Elements("input, textarea, select")
		if err != nil {
			continue
		}

		for _, inputEl := range inputEls {
			nameAttr, _ := inputEl.Attribute("name")
			typeAttr, _ := inputEl.Attribute("type")
			tagName := ""
			if tag, err := inputEl.Eval(`() => this.tagName.toLowerCase()`); err == nil {
				tagName = tag.Value.Str()
			}

			if nameAttr == nil || *nameAttr == "" {
				continue
			}

			fieldType := "text"
			if typeAttr != nil {
				fieldType = *typeAttr
			}
			// Skip submit/button/hidden types
			if fieldType == "submit" || fieldType == "button" || fieldType == "hidden" {
				continue
			}

			form.Fields = append(form.Fields, FormField{
				Name:    *nameAttr,
				Type:    fieldType,
				TagName: tagName,
			})
		}

		if len(form.Fields) > 0 {
			forms = append(forms, form)
		}
	}

	return forms, nil
}

// TestReflectedXSS tests a single form field for reflected XSS
func TestReflectedXSS(browser *rod.Browser, baseURL string, form Form, field FormField, payload string) (*Finding, error) {
	// Build the test URL with the payload
	testURL, err := buildTestURL(baseURL, form, field, payload)
	if err != nil {
		return nil, err
	}

	page, err := openAndWait(browser, testURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open test page: %w", err)
	}
	defer page.Close()

	// Check if the payload is reflected in the page source
	pageHTML, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("failed to get page HTML: %w", err)
	}

	if strings.Contains(pageHTML, payload) {
		return &Finding{
			Type:     "reflected",
			URL:      testURL,
			Field:    field.Name,
			Payload:  payload,
			Evidence: extractEvidence(pageHTML, payload),
		}, nil
	}

	return nil, nil
}

// TestStoredXSS tests a form for stored XSS by submitting a payload via POST
// and then reloading the page to check if the payload persists
func TestStoredXSS(browser *rod.Browser, baseURL string, form Form, field FormField, payload string) (*Finding, error) {
	if form.Method != "POST" {
		return nil, nil
	}

	submitURL := baseURL
	if form.Action != "" {
		submitURL = resolveURL(baseURL, form.Action)
	}

	// Step 1: Submit the payload via a new page
	page, err := openAndWait(browser, submitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open submit page: %w", err)
	}
	defer page.Close()

	// Fill the target field with payload, others with benign data
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

	// Submit the form
	submitBtns, _ := formEl.Elements("button[type=submit], input[type=submit]")
	if len(submitBtns) > 0 {
		submitBtns[0].Click(proto.InputMouseButtonLeft, 1)
	}

	time.Sleep(500 * time.Millisecond)
	page.Close()

	// Step 2: Reload the page and check for the stored payload
	checkPage, err := openAndWait(browser, submitURL)
	if err != nil {
		return nil, nil
	}
	defer checkPage.Close()

	pageHTML, err := checkPage.HTML()
	if err != nil {
		return nil, nil
	}

	if strings.Contains(pageHTML, payload) {
		return &Finding{
			Type:     "stored",
			URL:      submitURL,
			Field:    field.Name,
			Payload:  payload,
			Evidence: extractEvidence(pageHTML, payload),
		}, nil
	}

	return nil, nil
}

// ScanPage performs a full XSS scan on a single page
func ScanPage(browser *rod.Browser, targetURL string) (*ScanResult, error) {
	result := &ScanResult{
		TargetURL:    targetURL,
		PagesScanned: 1,
	}

	page, err := openAndWait(browser, targetURL)
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
				// Test reflected XSS
				finding, err := TestReflectedXSS(browser, targetURL, form, field, payload)
				if err != nil {
					continue
				}
				if finding != nil {
					result.Findings = append(result.Findings, *finding)
					break // One finding per field is sufficient
				}
			}

			for _, payload := range payloads {
				// Test stored XSS
				finding, err := TestStoredXSS(browser, targetURL, form, field, payload)
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

// buildTestURL constructs a URL with the XSS payload injected into the query parameter
func buildTestURL(baseURL string, form Form, field FormField, payload string) (string, error) {
	targetURL := baseURL
	if form.Action != "" {
		targetURL = resolveURL(baseURL, form.Action)
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

// resolveURL resolves a potentially relative URL against a base URL
func resolveURL(base, ref string) string {
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

// extractEvidence pulls a snippet of HTML around the reflected payload
func extractEvidence(html, payload string) string {
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
