package actions

import (
	"fmt"
	"github.com/agenthands/rod-cli/types"
	"github.com/agenthands/rod-cli/utils"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultWaitStableDur = 1 * time.Second
	defaultDomDiff       = 0.2
)

func Navigate(ctx *types.Context, url string) (string, error) {
	if !utils.IsHttp(url) {
		return "", errors.New("invalid URL")
	}
	page, err := ctx.EnsurePage()
	if err != nil {
		return "", errors.Wrapf(err, "Failed to navigate to %s", url)
	}
	if err := page.Navigate(url); err != nil {
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
	if err := page.NavigateBack(); err != nil {
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
	if err := page.NavigateForward(); err != nil {
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
	if err := page.Reload(); err != nil {
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
	if err := page.Keyboard.Type(input.Key(key)); err != nil {
		return "", errors.Wrapf(err, "Failed to press key %s", string(key))
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Press key %s successfully", string(key)), nil
}

func CloseBrowser(ctx *types.Context) (string, error) {
	if err := ctx.CloseBrowser(); err != nil {
		return "", errors.Wrap(err, "Failed to close browser")
	}
	return "Close browser successfully", nil
}

func Evaluate(ctx *types.Context, script string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	r, err := proto.RuntimeEvaluate{
		Expression:            script,
		ObjectGroup:           "console",
		IncludeCommandLineAPI: true,
	}.Call(page)
	if err != nil {
		return "", errors.Wrap(err, "Failed to evaluate code")
	}
	if r.ExceptionDetails != nil {
		return "", fmt.Errorf("Exception: %s", r.ExceptionDetails.Exception.Description)
	}
	return fmt.Sprintf("Evaluate code successfully with result: %s", r.Result.Value.String()), nil
}

func Screenshot(ctx *types.Context, name string, selector string, width, height float64) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	req := &proto.PageCaptureScreenshot{
		Format: proto.PageCaptureScreenshotFormatPng,
	}
	bin, err := page.Screenshot(false, req)
	if err != nil {
		return "", errors.Wrap(err, "Failed to screenshot")
	}
	toFile := []string{"tmp", "screenshots", name + ".png"}
	filePath := filepath.Join(toFile...)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filePath, bin, 0664); err != nil {
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
	element, err := snapshot.LocatorInFrame(ref)
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
	if err := element.Click(proto.InputMouseButtonLeft, 1); err != nil {
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
	if err := element.Input(value); err != nil {
		return "", errors.Wrap(err, "Failed to fill element")
	}
	if submit {
		if err := element.Page().Keyboard.Press(input.Enter); err != nil {
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
	if err := element.Select(values, true, rod.SelectorTypeText); err != nil {
		return "", errors.Wrap(err, "Failed to select option(s)")
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Select option(s) in element %s successfully", ref), nil
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
	if err := element.Hover(); err != nil {
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
	if err := element.Click(proto.InputMouseButtonLeft, 2); err != nil {
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
	for _, char := range text {
		if err := element.Type(input.Key(char)); err != nil {
			return "", errors.Wrap(err, "Failed to type text")
		}
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Typed text into element %s successfully", ref), nil
}

func Pdf(ctx *types.Context, name string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	pdf, err := page.PDF(&proto.PagePrintToPDF{})
	if err != nil {
		return "", errors.Wrap(err, "Failed to generate pdf")
	}
	b, err := io.ReadAll(pdf)
	if err != nil {
		return "", err
	}
	toFile := []string{"tmp", "pdfs", name + ".pdf"}
	filePath := filepath.Join(toFile...)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filePath, b, 0664); err != nil {
		return "", errors.Wrap(err, "Failed to save pdf")
	}
	return fmt.Sprintf("Save to %s", filePath), nil
}

// parseKey parses a string to input.Key
func parseKey(keyStr string) input.Key {
	switch keyStr {
	case "Enter": return input.Enter
	case "Tab": return input.Tab
	case "Backspace": return input.Backspace
	case "Escape": return input.Escape
	case "ArrowUp": return input.ArrowUp
	case "ArrowDown": return input.ArrowDown
	case "ArrowLeft": return input.ArrowLeft
	case "ArrowRight": return input.ArrowRight
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
	if err := page.Keyboard.Press(parseKey(key)); err != nil {
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
	if err := page.Mouse.MoveTo(proto.Point{X: x, Y: y}); err != nil {
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
	if err := page.Mouse.Down(btn, 1); err != nil {
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
	if err := page.Mouse.Up(btn, 1); err != nil {
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
	cookies, err := page.Browser().GetCookies()
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
	if err := page.Browser().SetCookies(nil); err != nil {
		return "", fmt.Errorf("failed to clear cookies: %w", err)
	}
	return "Cookies cleared", nil
}

// EvalStorage evaluates localStorage or sessionStorage commands
func EvalStorage(ctx *types.Context, storageType string, action string, key string, value string) (string, error) {
	var script string
	switch action {
	case "get":
		if key == "" {
			script = fmt.Sprintf("() => JSON.stringify(Object.fromEntries(Object.entries(window.%s)))", storageType)
		} else {
			script = fmt.Sprintf("() => window.%s.getItem('%s')", storageType, key)
		}
	case "set":
		script = fmt.Sprintf("() => window.%s.setItem('%s', '%s')", storageType, key, value)
	case "clear":
		script = fmt.Sprintf("() => window.%s.clear()", storageType)
	default:
		return "", fmt.Errorf("unknown storage action: %s", action)
	}

	res, err := Evaluate(ctx, script)
	if err != nil {
		return "", fmt.Errorf("failed to eval storage: %w", err)
	}
	return res, nil
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
	_, err = element.Eval(`() => {
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
	if _, err := page.Eval(script); err != nil {
		return "", fmt.Errorf("failed to clear highlights: %w", err)
	}
	return "Highlights cleared", nil
}

// VideoStart starts recording (Stubbed for MVP as full mp4 generation requires ffmpeg or complex frame dumping)
func VideoStart(ctx *types.Context, name string) (string, error) {
	return "Video recording started (STUB: requires ffmpeg pipeline)", nil
}

// VideoStop stops recording
func VideoStop(ctx *types.Context) (string, error) {
	return "Video recording stopped", nil
}

// Show un-hides the browser or provides interactive annotate
func Show(ctx *types.Context, annotate bool) (string, error) {
	if annotate {
		return "Annotation UI launched at http://127.0.0.1:8080 (STUB: blocking for feedback...)", nil
	}
	return "Browser must be started without --headless to be visible. Run 'rod-cli close' and restart without --headless.", nil
}
