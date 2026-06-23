package actions

import (
	"encoding/json"
	"fmt"
	"github.com/agenthands/godoll/humanize"
	"github.com/agenthands/godoll/retry"
	"github.com/agenthands/rod-cli/types"
	"github.com/agenthands/rod-cli/types/js"
	"github.com/agenthands/rod-cli/utils"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultWaitStableDur = 1 * time.Second
	defaultDomDiff       = 0.2
)

// seams for testing — overridden in *_test.go to exercise otherwise-unreachable
// error branches. Each defaults to the real rod/humanize/io operation, so
// production behavior is unchanged.
var (
	pageNavigateBack    = (*rod.Page).NavigateBack
	pageNavigateForward = (*rod.Page).NavigateForward
	pageReload          = (*rod.Page).Reload
	keyboardType        = func(kb *rod.Keyboard, keys ...input.Key) error { return kb.Type(keys...) }
	keyboardPress       = func(kb *rod.Keyboard, key input.Key) error { return kb.Press(key) }
	ctxCloseBrowser     = (*types.Context).CloseBrowser
	elementEval         = func(el *rod.Element, js string, params ...interface{}) (*proto.RuntimeRemoteObject, error) {
		return el.Eval(js, params...)
	}
	pageEval = func(p *rod.Page, js string, args ...interface{}) (*proto.RuntimeRemoteObject, error) {
		return p.Eval(js, args...)
	}
	runtimeEvaluateCall = func(req proto.RuntimeEvaluate, page *rod.Page) (*proto.RuntimeEvaluateResult, error) {
		return req.Call(page)
	}
	pageScreenshot = func(p *rod.Page, fullPage bool, req *proto.PageCaptureScreenshot) ([]byte, error) {
		return p.Screenshot(fullPage, req)
	}
	pagePDF             = func(p *rod.Page, req *proto.PagePrintToPDF) (*rod.StreamReader, error) { return p.PDF(req) }
	osMkdirAll          = os.MkdirAll
	osWriteFile         = os.WriteFile
	clickWithMouse      = humanize.ClickWithMouse
	typeWithHumanize    = humanize.TypeWithHumanize
	humanizeHover       = humanize.Hover
	humanizeScrollBy    = humanize.ScrollBy
	humanizeDragAndDrop = humanize.DragAndDrop
	elementClick        = func(el *rod.Element, button proto.InputMouseButton, n int) error { return el.Click(button, n) }
	elementSelect       = func(el *rod.Element, selectors []string, selected bool, t rod.SelectorType) error {
		return el.Select(selectors, selected, t)
	}
	elementSetFiles      = func(el *rod.Element, paths []string) error { return el.SetFiles(paths) }
	pageSetViewport      = func(p *rod.Page, params *proto.EmulationSetDeviceMetricsOverride) error { return p.SetViewport(params) }
	pageSetCookies       = func(p *rod.Page, cookies []*proto.NetworkCookieParam) error { return p.SetCookies(cookies) }
	browserGetCookies    = func(b *rod.Browser) ([]*proto.NetworkCookie, error) { return b.GetCookies() }
	browserSetCookies    = func(b *rod.Browser, cookies []*proto.NetworkCookieParam) error { return b.SetCookies(cookies) }
	networkDeleteCookies = func(req proto.NetworkDeleteCookies, page *rod.Page) error { return req.Call(page) }
	browserPages         = func(b *rod.Browser) (rod.Pages, error) { return b.Pages() }
	browserPage          = func(b *rod.Browser, opts proto.TargetCreateTarget) (*rod.Page, error) { return b.Page(opts) }
	pageActivate         = func(p *rod.Page) (*rod.Page, error) { return p.Activate() }
	pageClose            = func(p *rod.Page) error { return p.Close() }
	pageMouseMoveTo      = func(p *rod.Page, pt proto.Point) error { return p.Mouse.MoveTo(pt) }
	pageMouseDown        = func(p *rod.Page, btn proto.InputMouseButton, clicks int) error { return p.Mouse.Down(btn, clicks) }
	pageMouseUp          = func(p *rod.Page, btn proto.InputMouseButton, clicks int) error { return p.Mouse.Up(btn, clicks) }
	ioReadAll            = io.ReadAll
)

func Navigate(ctx *types.Context, url string) (string, error) {
	if !utils.IsHttp(url) {
		return "", errors.New("invalid URL")
	}
	page, err := ctx.EnsurePage()
	if err != nil {
		return "", errors.Wrapf(err, "Failed to navigate to %s", url)
	}
	err = retry.Retry(func() error {
		return page.Navigate(url)
	}, retry.WithMaxRetries(3), retry.WithExponentialBackoff(), retry.WithDelay(time.Second))
	if err != nil {
		return "", errors.Wrapf(err, "Failed to navigate to %s", url)
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Navigated to %s", url), nil
}

func GoBack(ctx *types.Context) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	err = retry.Retry(func() error {
		return pageNavigateBack(page)
	}, retry.WithMaxRetries(3), retry.WithExponentialBackoff(), retry.WithDelay(time.Second))
	if err != nil {
		return "", errors.Wrap(err, "Failed to go back")
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return "Go back successfully", nil
}

func GoForward(ctx *types.Context) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	err = retry.Retry(func() error {
		return pageNavigateForward(page)
	}, retry.WithMaxRetries(3), retry.WithExponentialBackoff(), retry.WithDelay(time.Second))
	if err != nil {
		return "", errors.Wrap(err, "Failed to go forward")
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return "Go forward successfully", nil
}

func Reload(ctx *types.Context) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	err = retry.Retry(func() error {
		return pageReload(page)
	}, retry.WithMaxRetries(3), retry.WithExponentialBackoff(), retry.WithDelay(time.Second))
	if err != nil {
		return "", errors.Wrap(err, "Failed to reload")
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return "Reload current page successfully", nil
}

func PressKey(ctx *types.Context, key rune) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	if err := keyboardType(page.Keyboard, input.Key(key)); err != nil {
		return "", errors.Wrapf(err, "Failed to press key %s", string(key))
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Press key %s successfully", string(key)), nil
}

func CloseBrowser(ctx *types.Context) (string, error) {
	if err := ctxCloseBrowser(ctx); err != nil {
		return "", errors.Wrap(err, "Failed to close browser")
	}
	return "Close browser successfully", nil
}

func Evaluate(ctx *types.Context, script string, ref string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}

	var valStr string
	scriptTrimmed := strings.TrimSpace(script)

	// If a ref is provided, evaluate on that specific element
	if ref != "" {
		element, err := getElementByRef(ctx, ref)
		if err != nil {
			return "", err
		}
		res, err := elementEval(element, scriptTrimmed)
		if err != nil {
			return "", errors.Wrap(err, "Failed to evaluate code on element")
		}
		if res != nil && !res.Value.Nil() {
			valStr = res.Value.String()
		} else {
			valStr = "null"
		}
		return fmt.Sprintf("Evaluate code successfully with result: %s", valStr), nil
	}

	// Global page evaluation
	isFunc := strings.HasPrefix(scriptTrimmed, "function") ||
		strings.HasPrefix(scriptTrimmed, "()") ||
		strings.HasPrefix(scriptTrimmed, "async ()") ||
		strings.HasPrefix(scriptTrimmed, "el =>") ||
		strings.HasPrefix(scriptTrimmed, "(el) =>") ||
		strings.HasPrefix(scriptTrimmed, "e =>") ||
		strings.HasPrefix(scriptTrimmed, "() =>") ||
		strings.HasPrefix(scriptTrimmed, "async () =>")

	if isFunc {
		res, err := pageEval(page, scriptTrimmed)
		if err != nil {
			return "", errors.Wrap(err, "Failed to evaluate code")
		}
		if res != nil {
			if !res.Value.Nil() {
				valStr = res.Value.String()
			} else {
				valStr = "null"
			}
		}
	} else {
		r, err := runtimeEvaluateCall(proto.RuntimeEvaluate{
			Expression:            scriptTrimmed,
			ObjectGroup:           "console",
			IncludeCommandLineAPI: true,
			AwaitPromise:          true,
		}, page)
		if err != nil {
			return "", errors.Wrap(err, "Failed to evaluate code")
		}
		if r.ExceptionDetails != nil {
			return "", fmt.Errorf("Exception: %s", r.ExceptionDetails.Exception.Description)
		}
		if r.Result != nil && !r.Result.Value.Nil() {
			valStr = r.Result.Value.String()
		} else if r.Result != nil && r.Result.Type == "string" {
			valStr = fmt.Sprintf("%v", r.Result.Value)
		} else if r.Result != nil {
			valStr = fmt.Sprintf("%v", r.Result.Value)
		} else {
			valStr = "null"
		}
	}

	return fmt.Sprintf("Evaluate code successfully with result: %s", valStr), nil
}

func Screenshot(ctx *types.Context, name string, selector string, width, height float64) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	req := &proto.PageCaptureScreenshot{
		Format: proto.PageCaptureScreenshotFormatPng,
	}
	bin, err := pageScreenshot(page, false, req)
	if err != nil {
		return "", errors.Wrap(err, "Failed to screenshot")
	}
	toFile := []string{"tmp", "screenshots", name + ".png"}
	filePath := filepath.Join(toFile...)
	if err := osMkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", err
	}
	if err := osWriteFile(filePath, bin, 0664); err != nil {
		return "", errors.Wrap(err, "Failed to save screenshot")
	}
	return fmt.Sprintf("Save to %s", filePath), nil
}

func Snapshot(ctx *types.Context) (string, error) {
	snapshot, err := ctx.BuildSnapshot()
	if err != nil {
		return "", errors.Wrap(err, "Failed to capture snapshot")
	}
	return snapshot, nil
}

func getElementByRef(ctx *types.Context, ref string) (*rod.Element, error) {
	snapshot, err := ctx.LatestSnapshot()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get latest snapshot (try running snapshot first)")
	}
	var element *rod.Element
	err = retry.Retry(func() error {
		var inErr error
		element, inErr = snapshot.LocatorInFrame(ref)
		return inErr
	}, retry.WithMaxRetries(3), retry.WithExponentialBackoff(), retry.WithDelay(500*time.Millisecond))

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to find element by ref %s", ref)
	}
	return element, nil
}

func Click(ctx *types.Context, ref string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	element, err := getElementByRef(ctx, ref)
	if err != nil {
		return "", err
	}
	if err := clickWithMouse(page, element); err != nil {
		return "", errors.Wrap(err, "Failed to click element")
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Click element %s successfully", ref), nil
}

func Fill(ctx *types.Context, ref string, value string, submit bool) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	element, err := getElementByRef(ctx, ref)
	if err != nil {
		return "", err
	}
	// Clear input first
	_ = element.SelectAllText()
	_ = page.Keyboard.Press(input.Backspace)
	if err := typeWithHumanize(element, value); err != nil {
		return "", errors.Wrap(err, "Failed to fill element")
	}
	if submit {
		if err := keyboardPress(element.Page().Keyboard, input.Enter); err != nil {
			return "", errors.Wrap(err, "Failed to submit element")
		}
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Fill out element %s successfully", ref), nil
}

func Select(ctx *types.Context, ref string, values []string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	element, err := getElementByRef(ctx, ref)
	if err != nil {
		return "", err
	}
	if err := elementSelect(element, values, true, rod.SelectorTypeText); err != nil {
		return "", errors.Wrap(err, "Failed to select option(s)")
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Select option(s) in element %s successfully", ref), nil
}

func Check(ctx *types.Context, ref string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	element, err := getElementByRef(ctx, ref)
	if err != nil {
		return "", err
	}
	_, err = elementEval(element, `() => {
		if (!this.checked) {
			this.checked = true;
			this.dispatchEvent(new Event('change', { bubbles: true }));
			this.dispatchEvent(new Event('input', { bubbles: true }));
		}
	}`)
	if err != nil {
		return "", errors.Wrap(err, "Failed to check element")
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Checked element %s successfully", ref), nil
}

func Uncheck(ctx *types.Context, ref string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	element, err := getElementByRef(ctx, ref)
	if err != nil {
		return "", err
	}
	_, err = elementEval(element, `() => {
		if (this.checked) {
			this.checked = false;
			this.dispatchEvent(new Event('change', { bubbles: true }));
			this.dispatchEvent(new Event('input', { bubbles: true }));
		}
	}`)
	if err != nil {
		return "", errors.Wrap(err, "Failed to uncheck element")
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Unchecked element %s successfully", ref), nil
}

func Upload(ctx *types.Context, ref string, filePaths []string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	element, err := getElementByRef(ctx, ref)
	if err != nil {
		return "", err
	}

	// Convert relative to absolute paths as needed by the browser
	absPaths := make([]string, len(filePaths))
	for i, p := range filePaths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			return "", errors.Wrap(err, "Failed to resolve absolute path for upload")
		}
		absPaths[i] = absPath
	}

	if err := elementSetFiles(element, absPaths); err != nil {
		return "", errors.Wrap(err, "Failed to upload files")
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Uploaded files to element %s successfully", ref), nil
}

func Hover(ctx *types.Context, ref string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	element, err := getElementByRef(ctx, ref)
	if err != nil {
		return "", err
	}
	if err := humanizeHover(element); err != nil {
		return "", errors.Wrap(err, "Failed to hover element")
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Hovered element %s successfully", ref), nil
}

func DblClick(ctx *types.Context, ref string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	element, err := getElementByRef(ctx, ref)
	if err != nil {
		return "", err
	}
	if err := elementClick(element, proto.InputMouseButtonLeft, 2); err != nil {
		return "", errors.Wrap(err, "Failed to double click")
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Double clicked element %s successfully", ref), nil
}

func Type(ctx *types.Context, ref string, text string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	element, err := getElementByRef(ctx, ref)
	if err != nil {
		return "", err
	}
	if err := typeWithHumanize(element, text); err != nil {
		return "", errors.Wrap(err, "Failed to type text")
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Typed text into element %s successfully", ref), nil
}

func Pdf(ctx *types.Context, name string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	pdf, err := pagePDF(page, &proto.PagePrintToPDF{})
	if err != nil {
		return "", errors.Wrap(err, "Failed to generate pdf")
	}
	b, err := ioReadAll(pdf)
	if err != nil {
		return "", err
	}
	toFile := []string{"tmp", "pdfs", name + ".pdf"}
	filePath := filepath.Join(toFile...)
	if err := osMkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", err
	}
	if err := osWriteFile(filePath, b, 0664); err != nil {
		return "", errors.Wrap(err, "Failed to save pdf")
	}
	return fmt.Sprintf("Save to %s", filePath), nil
}

func MouseWheel(ctx *types.Context, dx, dy float64) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}

	// Handle Y axis scrolling with godoll physics
	if dy > 0 {
		if err := humanizeScrollBy(page, humanize.ScrollDown, int(dy)); err != nil {
			return "", errors.Wrap(err, "Failed to scroll mouse wheel down")
		}
	} else if dy < 0 {
		if err := humanizeScrollBy(page, humanize.ScrollUp, int(-dy)); err != nil {
			return "", errors.Wrap(err, "Failed to scroll mouse wheel up")
		}
	}

	// Handle X axis scrolling with godoll physics
	if dx > 0 {
		if err := humanizeScrollBy(page, humanize.ScrollRight, int(dx)); err != nil {
			return "", errors.Wrap(err, "Failed to scroll mouse wheel right")
		}
	} else if dx < 0 {
		if err := humanizeScrollBy(page, humanize.ScrollLeft, int(-dx)); err != nil {
			return "", errors.Wrap(err, "Failed to scroll mouse wheel left")
		}
	}

	return "Mouse wheel scrolled", nil
}

func Resize(ctx *types.Context, width, height int) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	if err := pageSetViewport(page, &proto.EmulationSetDeviceMetricsOverride{Width: width, Height: height, DeviceScaleFactor: 1, Mobile: false}); err != nil {
		return "", errors.Wrap(err, "Failed to resize viewport")
	}
	return fmt.Sprintf("Viewport resized to %dx%d", width, height), nil
}

func TabList(ctx *types.Context) (string, error) {
	browser := ctx.GetBrowser()
	if browser == nil {
		return "", errors.New("No active browser")
	}
	pages, err := browserPages(browser)
	if err != nil {
		return "", errors.Wrap(err, "Failed to list tabs")
	}
	var res strings.Builder
	for i, p := range pages {
		info, _ := p.Info()
		res.WriteString(fmt.Sprintf("[%d] %s (%s)\n", i, info.Title, info.URL))
	}
	return res.String(), nil
}

func TabNew(ctx *types.Context, url string) (string, error) {
	browser := ctx.GetBrowser()
	if browser == nil {
		return "", errors.New("No active browser")
	}
	var targetURL string
	if url != "" {
		targetURL = url
	} else {
		targetURL = "about:blank"
	}
	page, err := browserPage(browser, proto.TargetCreateTarget{URL: targetURL})
	if err != nil {
		return "", errors.Wrap(err, "Failed to create new tab")
	}
	ctx.SetPage(page)
	return fmt.Sprintf("New tab created: %s", targetURL), nil
}

func TabClose(ctx *types.Context, index int) (string, error) {
	browser := ctx.GetBrowser()
	if browser == nil {
		return "", errors.New("No active browser")
	}
	pages, err := browserPages(browser)
	if err != nil {
		return "", err
	}
	if index < 0 || index >= len(pages) {
		return "", errors.Errorf("Tab index out of range: %d", index)
	}
	if err := pageClose(pages[index]); err != nil {
		return "", errors.Wrap(err, "Failed to close tab")
	}
	return fmt.Sprintf("Closed tab %d", index), nil
}

func TabSelect(ctx *types.Context, index int) (string, error) {
	browser := ctx.GetBrowser()
	if browser == nil {
		return "", errors.New("No active browser")
	}
	pages, err := browserPages(browser)
	if err != nil {
		return "", err
	}
	if index < 0 || index >= len(pages) {
		return "", errors.Errorf("Tab index out of range: %d", index)
	}
	page := pages[index]
	if _, err := pageActivate(page); err != nil {
		return "", errors.Wrap(err, "Failed to activate tab")
	}
	ctx.SetPage(page)
	return fmt.Sprintf("Selected tab %d", index), nil
}

// parseKey parses a string to input.Key
func parseKey(keyStr string) input.Key {
	switch keyStr {
	case "Enter":
		return input.Enter
	case "Tab":
		return input.Tab
	case "Backspace":
		return input.Backspace
	case "Escape":
		return input.Escape
	case "ArrowUp":
		return input.ArrowUp
	case "ArrowDown":
		return input.ArrowDown
	case "ArrowLeft":
		return input.ArrowLeft
	case "ArrowRight":
		return input.ArrowRight
	}
	if len(keyStr) > 0 {
		return input.Key(rune(keyStr[0]))
	}
	return input.Key(0)
}

// Press triggers a raw keyboard key press
func Press(ctx *types.Context, key string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	if err := keyboardPress(page.Keyboard, parseKey(key)); err != nil {
		return "", fmt.Errorf("failed to press key %s: %w", key, err)
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Pressed key: %s", key), nil
}

// MouseMove triggers a raw mouse move
func MouseMove(ctx *types.Context, x, y float64) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	if err := pageMouseMoveTo(page, proto.Point{X: x, Y: y}); err != nil {
		return "", fmt.Errorf("failed to move mouse: %w", err)
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Moved mouse to %f, %f", x, y), nil
}

// MouseDown triggers a raw mouse down
func MouseDown(ctx *types.Context, button string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	btn := proto.InputMouseButtonLeft
	if button == "right" {
		btn = proto.InputMouseButtonRight
	} else if button == "middle" {
		btn = proto.InputMouseButtonMiddle
	}
	if err := pageMouseDown(page, btn, 1); err != nil {
		return "", fmt.Errorf("failed to mousedown: %w", err)
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return "Mouse down", nil
}

// MouseUp triggers a raw mouse up
func MouseUp(ctx *types.Context, button string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	btn := proto.InputMouseButtonLeft
	if button == "right" {
		btn = proto.InputMouseButtonRight
	} else if button == "middle" {
		btn = proto.InputMouseButtonMiddle
	}
	if err := pageMouseUp(page, btn, 1); err != nil {
		return "", fmt.Errorf("failed to mouseup: %w", err)
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return "Mouse up", nil
}

// HandleDialog handles javascript alerts/confirms
func HandleDialog(ctx *types.Context, accept bool, promptText string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	wait, handle := page.MustHandleDialog()
	go func() {
		wait()
		handle(accept, promptText)
	}()
	return "Set up dialog handler for next dialog", nil
}

// GetCookies returns the current cookies
func GetCookies(ctx *types.Context) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	cookies, err := browserGetCookies(page.Browser())
	if err != nil {
		return "", fmt.Errorf("failed to get cookies: %w", err)
	}

	// Format cookies into a string representation instead of JSON marshaling the Rod types directly,
	// or we can marshal if json import is present. json is not imported, let's just format it simple.
	// Wait, we can just use Evaluate for localStorage, but cookies are browser level.
	// Let's just return length or basic info, or we can use standard json import.
	// We'll return a simple summary since JSON is not imported yet, or we'll add json import later.
	// Rod's cookies are []*proto.NetworkCookie.
	res := "Cookies:\n"
	for _, c := range cookies {
		res += fmt.Sprintf("- %s: %s (domain: %s)\n", c.Name, c.Value, c.Domain)
	}
	return res, nil
}

// ClearCookies clears all cookies
func ClearCookies(ctx *types.Context) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	if err := browserSetCookies(page.Browser(), nil); err != nil {
		return "", fmt.Errorf("failed to clear cookies: %w", err)
	}
	return "Cookies cleared", nil
}

func SetCookie(ctx *types.Context, name, value string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	info, _ := page.Info()
	err = pageSetCookies(page, []*proto.NetworkCookieParam{{
		Name:  name,
		Value: value,
		URL:   info.URL,
	}})
	if err != nil {
		return "", errors.Wrap(err, "Failed to set cookie")
	}
	return fmt.Sprintf("Set cookie %s", name), nil
}

func DeleteCookie(ctx *types.Context, name string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	info, _ := page.Info()
	err = networkDeleteCookies(proto.NetworkDeleteCookies{Name: name, URL: info.URL}, page)
	if err != nil {
		return "", errors.Wrap(err, "Failed to delete cookie")
	}
	return fmt.Sprintf("Deleted cookie %s", name), nil
}

// EvalStorage evaluates localStorage or sessionStorage commands
func EvalStorage(ctx *types.Context, storageType string, action string, key string, value string) (string, error) {
	var script string
	switch action {
	case "get":
		if key == "" {
			script = fmt.Sprintf("JSON.stringify(Object.fromEntries(Object.entries(window.%s)))", storageType)
		} else {
			keyBytes, _ := json.Marshal(key)
			script = fmt.Sprintf("window.%s.getItem(%s)", storageType, string(keyBytes))
		}
	case "set":
		keyBytes, _ := json.Marshal(key)
		valBytes, _ := json.Marshal(value)
		script = fmt.Sprintf("window.%s.setItem(%s, %s)", storageType, string(keyBytes), string(valBytes))
	case "delete":
		keyBytes, _ := json.Marshal(key)
		script = fmt.Sprintf("window.%s.removeItem(%s)", storageType, string(keyBytes))
	case "clear":
		script = fmt.Sprintf("window.%s.clear()", storageType)
	default:
		return "", fmt.Errorf("unknown storage action: %s", action)
	}

	res, err := Evaluate(ctx, script, "")
	if err != nil {
		return "", fmt.Errorf("failed to eval storage: %w", err)
	}
	return res, nil
}

func ConsoleLogs(ctx *types.Context) (string, error) {
	logs := ctx.GetConsoleLogs()
	if len(logs) == 0 {
		return "No console logs", nil
	}
	return strings.Join(logs, "\n"), nil
}

func NetworkRequests(ctx *types.Context) (string, error) {
	reqs := ctx.GetRequests()
	if len(reqs) == 0 {
		return "No network requests", nil
	}
	var res strings.Builder
	for i, req := range reqs {
		res.WriteString(fmt.Sprintf("[%d] %s\n", i, req))
	}
	return res.String(), nil
}

func NetworkRequest(ctx *types.Context, index int) (string, error) {
	reqs := ctx.GetRequests()
	if index < 0 || index >= len(reqs) {
		return "", fmt.Errorf("Request index out of bounds")
	}
	return reqs[index], nil
}

func Route(ctx *types.Context, pattern string, body string) (string, error) {
	ctx.AddRoute(pattern, body)
	return fmt.Sprintf("Added route for %s", pattern), nil
}

func Unroute(ctx *types.Context, pattern string) (string, error) {
	ctx.RemoveRoute(pattern)
	return fmt.Sprintf("Removed route for %s", pattern), nil
}

func RouteList(ctx *types.Context) (string, error) {
	routes := ctx.GetRoutes()
	if len(routes) == 0 {
		return "No active routes", nil
	}
	var res strings.Builder
	for pattern := range routes {
		res.WriteString(fmt.Sprintf("- %s\n", pattern))
	}
	return res.String(), nil
}

// Highlight adds a highly visible red outline to the target element
func Highlight(ctx *types.Context, ref string) (string, error) {
	_, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	element, err := getElementByRef(ctx, ref)
	if err != nil {
		return "", err
	}

	// We use Eval on the specific element
	_, err = elementEval(element, `() => {
		this.style.outline = "5px solid red";
		this.style.boxShadow = "0 0 10px red";
		this.classList.add("rod-cli-highlighted");
	}`)
	if err != nil {
		return "", fmt.Errorf("failed to highlight: %w", err)
	}
	return fmt.Sprintf("Highlighted element: %s", ref), nil
}

// ClearHighlights removes all highlights
func ClearHighlights(ctx *types.Context) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	script := `() => {
		document.querySelectorAll(".rod-cli-highlighted").forEach(el => {
			el.style.outline = "";
			el.style.boxShadow = "";
			el.classList.remove("rod-cli-highlighted");
		});
	}`
	if _, err := pageEval(page, script); err != nil {
		return "", fmt.Errorf("failed to clear highlights: %w", err)
	}
	return "Highlights cleared", nil
}

func Drag(ctx *types.Context, startRef, endRef string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	el1, err := getElementByRef(ctx, startRef)
	if err != nil {
		return "", err
	}
	el2, err := getElementByRef(ctx, endRef)
	if err != nil {
		return "", err
	}

	if err := humanizeDragAndDrop(page, el1, el2); err != nil {
		return "", err
	}
	return fmt.Sprintf("Dragged from %s to %s", startRef, endRef), nil
}

func Drop(ctx *types.Context, ref string, path string) (string, error) {
	el, err := getElementByRef(ctx, ref)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if err := elementSetFiles(el, []string{absPath}); err != nil {
		return "", err
	}
	return fmt.Sprintf("Dropped file(s) onto %s", ref), nil
}

func StateSave(ctx *types.Context, path string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	cookies, err := browserGetCookies(page.Browser())
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(cookies)
	if err != nil {
		return "", err
	}
	if err := osWriteFile(path, b, 0644); err != nil {
		return "", err
	}
	return fmt.Sprintf("Saved state to %s", path), nil
}

func StateLoad(ctx *types.Context, path string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var cookies []*proto.NetworkCookieParam
	if err := json.Unmarshal(b, &cookies); err != nil {
		return "", err
	}
	if err := pageSetCookies(page, cookies); err != nil {
		return "", err
	}
	return fmt.Sprintf("Loaded state from %s", path), nil
}

// Show un-hides the browser or provides interactive annotate
func Show(ctx *types.Context, annotate bool) (string, error) {
	if annotate {
		page, err := ctx.ControlledPage()
		if err != nil {
			return "", err
		}

		res, err := pageEval(page, js.AnnotatorUI)
		if err != nil {
			return "", errors.Wrap(err, "Failed to launch annotation UI")
		}

		if res.Value.Get("cancelled").Bool() {
			return "Annotation cancelled by user.", nil
		}

		annotations := res.Value.Get("annotations")
		return fmt.Sprintf("Annotations saved:\n%s", annotations.JSON("", "  ")), nil
	}
	return "Browser must be started without --headless to be visible. Run 'rod-cli close' and restart without --headless.", nil
}
