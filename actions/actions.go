package actions

import (
	"bytes"
	"context"
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
	"image/png"
	"io"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	defaultWaitStableDur = 1 * time.Second
	defaultDomDiff       = 0.2
	// defaultNavigateTimeout bounds a single page.Navigate attempt. A target that
	// connects but never finishes loading (e.g. a stalled proxy CONNECT tunnel, a
	// hung server) must never hang the daemon forever — a CLI command must always
	// return. On timeout the navigation is treated as best-effort (the request was
	// issued; proceed with whatever loaded), mirroring the crawler's bounded-wait
	// philosophy. Generous enough that a normal slow page never trips it.
	defaultNavigateTimeout = 30 * time.Second
	// defaultWaitStableTimeout bounds the post-navigation DOM-settle so a page
	// that never stops mutating (or never loads) cannot hang the command either.
	defaultWaitStableTimeout = 15 * time.Second
	// defaultCaptureTimeout bounds Screenshot and Pdf's underlying CDP calls
	// (Page.captureScreenshot / Page.printToPDF). Unlike Navigate, neither
	// carried a timeout — confirmed live: rod.Page.Screenshot hung
	// indefinitely (>20s, verified via a SIGQUIT goroutine dump blocked
	// inside the CDP client waiting on an unresponsive browser) after a
	// normal goto+eval sequence, with no crash and no bound. Pdf shares the
	// identical unguarded shape (same seam-wrapper pattern, same rendering
	// CDP round-trip class) so gets the same bound preemptively. Generous
	// enough that a normal capture — typically well under a second — never
	// trips it.
	defaultCaptureTimeout = 30 * time.Second
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
	pagePDF          = func(p *rod.Page, req *proto.PagePrintToPDF) (*rod.StreamReader, error) { return p.PDF(req) }
	osMkdirAll       = os.MkdirAll
	osWriteFile      = os.WriteFile
	clickWithMouse   = humanize.ClickWithMouse
	typeWithHumanize = humanize.TypeWithHumanize
	moveToElement    = humanize.MoveTo
	elementClick     = func(el *rod.Element, button proto.InputMouseButton, n int) error { return el.Click(button, n) }
	elementSelect    = func(el *rod.Element, selectors []string, selected bool, t rod.SelectorType) error {
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
	pageMouseDown        = func(p *rod.Page, btn proto.InputMouseButton, clicks int) error { return p.Mouse.Down(btn, clicks) }
	pageMouseUp          = func(p *rod.Page, btn proto.InputMouseButton, clicks int) error { return p.Mouse.Up(btn, clicks) }
	ioReadAll            = io.ReadAll
	mouseScroll          = func(p *rod.Page, dx, dy float64) error { return p.Mouse.Scroll(dx, dy, 1) }
	moveToPoint          = humanize.MoveToPoint
	measureScroll        = measureScrollBoxes
	wheelSettle          = 400 * time.Millisecond
	wheelEndBudget       = 10 * time.Second
	queryElementDirect   = utils.QueryEleByAria
)

// --- Phase 28: humanize tuning option builders (HUMANIZE-01) ---
//
// Each builder translates the session's resolved StealthConfig humanize knobs
// (pointer-typed; nil = unset) into godoll option slices. The load-bearing
// invariant: an option is emitted ONLY when its value(s) are non-nil, and a
// min/max-pair option ONLY when BOTH ends are set. When every knob is nil the
// returned slice is empty ⇒ the variadic call applies godoll's own defaults ⇒
// behavior is byte-for-byte the current default (zero regression).

// typingOpts builds the []humanize.TypingOption for TypeWithHumanize from the
// resolved tuning. WithTypingSpeed needs BOTH ends; the spread between them IS
// the "delay jitter" (godoll has no separate jitter option).
func typingOpts(s types.StealthConfig) []humanize.TypingOption {
	var opts []humanize.TypingOption
	if s.TypingSpeedMin != nil && s.TypingSpeedMax != nil {
		opts = append(opts, humanize.WithTypingSpeed(*s.TypingSpeedMin, *s.TypingSpeedMax))
	}
	if s.TypoRate != nil {
		opts = append(opts, humanize.WithTypoRate(*s.TypoRate))
	}
	return opts
}

// mouseOpts builds the []humanize.MouseOption for ClickWithMouse from the
// resolved tuning. WithMouseSpeed needs BOTH ends.
func mouseOpts(s types.StealthConfig) []humanize.MouseOption {
	var opts []humanize.MouseOption
	if s.MouseTremor != nil {
		opts = append(opts, humanize.WithMouseTremor(*s.MouseTremor))
	}
	if s.MouseSteps != nil {
		opts = append(opts, humanize.WithMouseSteps(*s.MouseSteps))
	}
	if s.MouseSpeedMin != nil && s.MouseSpeedMax != nil {
		opts = append(opts, humanize.WithMouseSpeed(*s.MouseSpeedMin, *s.MouseSpeedMax))
	}
	if s.MouseDeviation != nil {
		opts = append(opts, humanize.WithMouseDeviation(*s.MouseDeviation))
	}
	return opts
}

func Navigate(ctx *types.Context, url string) (string, error) {
	if !utils.IsHttp(url) {
		return "", errors.New("invalid URL")
	}
	page, err := ctx.EnsurePage()
	if err != nil {
		return "", errors.Wrapf(err, "Failed to navigate to %s", url)
	}
	// Each Navigate attempt is bounded so a target that connects but never
	// finishes (stalled proxy tunnel, hung server) cannot hang the daemon. A
	// deadline-exceeded attempt is NOT retried (retrying an unresponsive target
	// just multiplies the wait) and is treated as best-effort success — the
	// navigation was issued; proceed with whatever is loaded. Genuine transient
	// errors still retry.
	err = retry.Retry(func() error {
		nerr := page.Timeout(defaultNavigateTimeout).Navigate(url)
		if errors.Is(nerr, context.DeadlineExceeded) {
			return nil // best-effort: stop retrying, proceed
		}
		return nerr
	}, retry.WithMaxRetries(3), retry.WithExponentialBackoff(), retry.WithDelay(time.Second))
	if err != nil {
		return "", errors.Wrapf(err, "Failed to navigate to %s", url)
	}
	// Bounded DOM-settle: a page that never quiesces (or never loads) must not
	// hang the command; best-effort, discard the timeout.
	_ = rod.Try(func() {
		page.Timeout(defaultWaitStableTimeout).WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	})
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
		return valStr, nil
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

	return valStr, nil
}

func Screenshot(ctx *types.Context, name, selector, out string, fullPage bool) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	var bin []byte
	if selector != "" {
		el, err := getElementByRef(ctx, selector)
		if err != nil {
			return "", err
		}
		bin, err = el.Timeout(defaultCaptureTimeout).Screenshot(proto.PageCaptureScreenshotFormatPng, 0)
		if err != nil {
			return "", errors.Wrap(err, "Failed to screenshot")
		}
	} else {
		req := &proto.PageCaptureScreenshot{
			Format: proto.PageCaptureScreenshotFormatPng,
		}
		bin, err = pageScreenshot(page.Timeout(defaultCaptureTimeout), fullPage, req)
		if err != nil {
			return "", errors.Wrap(err, "Failed to screenshot")
		}
	}
	abs, err := saveArtifact(out, filepath.Join("tmp", "screenshots"), name, ".png", bin)
	if err != nil {
		return "", err
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(bin))
	if err != nil {
		return "", errors.Wrap(err, "Failed to read screenshot size")
	}
	return fmt.Sprintf("saved %s %dx%d", abs, cfg.Width, cfg.Height), nil
}

var (
	snapRefRe  = regexp.MustCompile(`^s(\d+)e(\d+)$`)
	frameRefRe = regexp.MustCompile(`^f\d+s\d+e\d+$`)
)

// snapshotRootExpr turns a snapshot positional arg (a ref or a CSS selector)
// into the JS expression the snapshotter evaluates as its root element.
func snapshotRootExpr(root string) (string, error) {
	if root == "" {
		return "", nil
	}
	if frameRefRe.MatchString(root) {
		return "", errors.Errorf("snapshot %s does not support iframe refs yet", root)
	}
	if m := snapRefRe.FindStringSubmatch(root); m != nil {
		return fmt.Sprintf(`(()=>{const s=snapshotEngine._lastAriaSnapshot;if(!s||s.generation!==%s)throw new Error("stale ref %s: run snapshot");const e=s.elements.get(%s);if(!e)throw new Error("no element for ref %s");return e})()`, m[1], root, m[2], root), nil
	}
	sel, _ := json.Marshal(root)
	msg, _ := json.Marshal("no element matches " + root)
	return fmt.Sprintf(`document.querySelector(%s)||(()=>{throw new Error(%s)})()`, sel, msg), nil
}

// Snapshot captures the page, or only the subtree under root (a ref or CSS
// selector) when given; interactive keeps only controls and their ancestors.
func Snapshot(ctx *types.Context, root string, interactive bool) (string, error) {
	expr, err := snapshotRootExpr(root)
	if err != nil {
		return "", err
	}
	snapshot, err := ctx.BuildSnapshot(types.SnapshotOpts{Root: expr, Scope: root, Interactive: interactive})
	if err != nil {
		return "", errors.Wrap(err, "Failed to capture snapshot")
	}
	return snapshot, nil
}

func getElementByRef(ctx *types.Context, ref string) (*rod.Element, error) {
	var element *rod.Element
	err := retry.Retry(func() error {
		var inErr error
		if snapshot, snapErr := ctx.LatestSnapshot(); snapErr == nil {
			element, inErr = snapshot.LocatorInFrame(ref)
			return inErr
		}
		// No snapshot taken yet in this session -- fall back to resolving ref
		// directly against the current page. A bare "sNeM" snapshot ref is
		// meaningless without the JS-side snapshot state a prior `snapshot`
		// call establishes, and correctly still fails as "not found" below;
		// a plain CSS selector resolves fine, so click/fill/type/etc. work
		// standalone with no snapshot dependency.
		page, pageErr := ctx.ControlledPage()
		if pageErr != nil {
			return pageErr
		}
		element, inErr = queryElementDirect(page, ref)
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
	if err := clickWithMouse(page, element, mouseOpts(ctx.HumanizeTuning())...); err != nil {
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
	if err := typeWithHumanize(element, value, typingOpts(ctx.HumanizeTuning())...); err != nil {
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
	for _, v := range values {
		if err := selectOne(element, v); err != nil {
			return "", err
		}
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	res, err := elementEval(element, `() => Array.from(this.selectedOptions, o => o.text.trim())`)
	if err != nil {
		return "", errors.Wrap(err, "Failed to read selected options")
	}
	labels := make([]string, 0, len(res.Value.Arr()))
	for _, l := range res.Value.Arr() {
		labels = append(labels, strconv.Quote(l.Str()))
	}
	return fmt.Sprintf("Selected %s in %s", strings.Join(labels, ", "), ref), nil
}

// selectOne selects the option matching v: option text, then value, then
// case-insensitive substring of the text.
func selectOne(el *rod.Element, v string) error {
	if elementSelect(el, []string{v}, true, rod.SelectorTypeText) == nil || selectByValue(el, v) == nil {
		return nil
	}
	res, err := elementEval(el, `(v) => {
		const o = Array.from(this.options).find(o => o.text.toLowerCase().includes(v.toLowerCase()));
		return o ? o.value : null;
	}`, v)
	if err != nil {
		return errors.Wrap(err, "Failed to select option(s)")
	}
	if res.Value.Nil() {
		return fmt.Errorf("no option matches %q", v)
	}
	if err := selectByValue(el, res.Value.Str()); err != nil {
		return errors.Wrap(err, "Failed to select option(s)")
	}
	return nil
}

func selectByValue(el *rod.Element, v string) error {
	return elementSelect(el, []string{"[value=" + strconv.Quote(v) + "]"}, true, rod.SelectorTypeCSSSector)
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
	if err := moveToElement(page, element, mouseOpts(ctx.HumanizeTuning())...); err != nil {
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
	if err := typeWithHumanize(element, text, typingOpts(ctx.HumanizeTuning())...); err != nil {
		return "", errors.Wrap(err, "Failed to type text")
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Typed text into element %s successfully", ref), nil
}

func Pdf(ctx *types.Context, name, out string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	pdf, err := pagePDF(page.Timeout(defaultCaptureTimeout), &proto.PagePrintToPDF{})
	if err != nil {
		return "", errors.Wrap(err, "Failed to generate pdf")
	}
	b, err := ioReadAll(pdf)
	if err != nil {
		return "", err
	}
	abs, err := saveArtifact(out, filepath.Join("tmp", "pdfs"), name, ".pdf", b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("saved %s", abs), nil
}

// saveArtifact writes data to out when set, else to dir/<base of name>+ext,
// and returns the absolute path. The base-name rule keeps a hostile --name
// inside dir; out is an explicit caller-chosen path and is used as given.
func saveArtifact(out, dir, name, ext string, data []byte) (string, error) {
	filePath := out
	if filePath == "" {
		base := filepath.Base(name)
		if base == "." || base == ".." || base == "" || base == string(filepath.Separator) {
			return "", fmt.Errorf("invalid name %q", name)
		}
		filePath = filepath.Join(dir, base+ext)
	}
	if err := osMkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", err
	}
	if err := osWriteFile(filePath, data, 0664); err != nil {
		return "", errors.Wrap(err, "Failed to save "+ext[1:])
	}
	return filepath.Abs(filePath)
}

// scrollBox is one scroll container's position, limits and viewport size.
type scrollBox struct{ X, Y, MaxX, MaxY, VW, VH float64 }

// measureScrollBoxes returns el's scrollable ancestors, innermost first: each
// document's boxes end with its scrollingElement, then continue from the
// frame element that holds it, so the list always ends with the top
// document.scrollingElement. A cross-origin parent hides frameElement (null),
// so measuring stops at the frame's own document.
func measureScrollBoxes(el *rod.Element) ([]scrollBox, error) {
	res, err := elementEval(el, `function(){
  const box = e => ({X:e.scrollLeft, Y:e.scrollTop, MaxX:e.scrollWidth-e.clientWidth, MaxY:e.scrollHeight-e.clientHeight, VW:e.clientWidth, VH:e.clientHeight});
  const out = [];
  for (let e = this, doc = this.ownerDocument; doc; ) {
    const root = doc.scrollingElement;
    for (; e; e = e.parentElement || (e.getRootNode && e.getRootNode().host)) {
      if (e.nodeType !== 1 || e === root) continue;
      const s = doc.defaultView.getComputedStyle(e);
      if (/auto|scroll|overlay/.test(s.overflowX + s.overflowY) && (e.scrollHeight > e.clientHeight || e.scrollWidth > e.clientWidth)) out.push(box(e));
    }
    if (root) out.push(box(root));
    e = doc.defaultView.frameElement;
    doc = e && e.ownerDocument;
  }
  return out;
}`)
	if err != nil {
		return nil, err
	}
	var boxes []scrollBox
	if err := res.Value.Unmarshal(&boxes); err != nil {
		return nil, err
	}
	return boxes, nil
}

// wheelAnchorJS(x,y) starts at this (a ref element) or else at the element
// under top-viewport point x,y. While that element is a frame whose document
// is readable (same origin), it descends into the frame's document at the
// same point. A cross-origin frame cannot be read from the parent, so the
// frame element itself is the anchor.
const wheelAnchorJS = `function(x,y){
  let e = this && this.nodeType === 1 ? this : null;
  if (e) { // a ref may live in a frame: move x,y into its document
    for (let f = e.ownerDocument.defaultView.frameElement; f; f = f.ownerDocument.defaultView.frameElement) {
      const r = f.getBoundingClientRect();
      x -= r.left + f.clientLeft; y -= r.top + f.clientTop;
    }
  } else e = document.elementFromPoint(x,y) || document.scrollingElement;
  while (e && (e.tagName === 'IFRAME' || e.tagName === 'FRAME')) {
    let d = null;
    try { d = e.contentDocument } catch (_) {}
    if (!d) break;
    const r = e.getBoundingClientRect();
    x -= r.left + e.clientLeft; y -= r.top + e.clientTop;
    e = d.elementFromPoint(x,y) || d.scrollingElement;
  }
  return e;
}`

// MouseWheel scrolls with real wheel input and reports what actually moved.
// dx=dy=0 → one viewport down. ref=="" → page; ref → the ref's nearest
// scrollable ancestor. maxSteps 0 = one scroll; n>0 = until end, at most n
// scrolls or wheelEndBudget (then " cap").
func MouseWheel(ctx *types.Context, dx, dy float64, ref string, maxSteps int) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	// ponytail: moving the pointer onto the target fires hover effects (menus,
	// tooltips) like a real user would; add a no-move mode if that bites.
	var anchor *rod.Element
	if ref != "" {
		if anchor, err = getElementByRef(ctx, ref); err != nil {
			return "", err
		}
		// MoveTo scrolls the element into view and lands inside it.
		if err := humanize.MoveTo(page, anchor, mouseOpts(ctx.HumanizeTuning())...); err != nil {
			return "", errors.Wrap(err, "Failed to move pointer")
		}
		pt := humanize.GetMousePos(page)
		if anchor, err = anchor.ElementByJS(rod.Eval(wheelAnchorJS, pt.X, pt.Y)); err != nil {
			return "", errors.Wrap(err, "Failed to find scroll target")
		}
	} else {
		m, err := proto.PageGetLayoutMetrics{}.Call(page)
		if err != nil {
			return "", errors.Wrap(err, "Failed to read viewport")
		}
		w, h := float64(m.CSSLayoutViewport.ClientWidth), float64(m.CSSLayoutViewport.ClientHeight)
		pt := humanize.Point{X: w * (0.4 + 0.2*rand.Float64()), Y: h * (0.4 + 0.2*rand.Float64())}
		if err := moveToPoint(page, pt, mouseOpts(ctx.HumanizeTuning())...); err != nil {
			return "", errors.Wrap(err, "Failed to move pointer")
		}
		if anchor, err = page.ElementByJS(rod.Eval(wheelAnchorJS, pt.X, pt.Y)); err != nil {
			return "", errors.Wrap(err, "Failed to find scroll target")
		}
	}

	first, err := measureScroll(anchor)
	if err != nil || len(first) == 0 {
		return "", errors.Wrap(err, "Failed to measure scroll")
	}
	dxReq := dx != 0
	if dx == 0 && dy == 0 {
		dy = first[0].VH
	}
	// atLimit: the requested direction sits at its limit. atEnd: or moved 0 px.
	atLimit := func(a scrollBox) bool {
		if dy == 0 {
			return dx > 0 && a.X >= a.MaxX || dx < 0 && a.X <= 0
		}
		return dy > 0 && a.Y >= a.MaxY || dy < 0 && a.Y <= 0
	}
	atEnd := func(b, a scrollBox) bool {
		moved := a.Y - b.Y
		if dy == 0 {
			moved = a.X - b.X
		}
		return moved == 0 || atLimit(a)
	}

	before, i, suffix := first, -1, ""
	var a scrollBox
	deadline := time.Now().Add(wheelEndBudget)
	for step := 1; ; step++ {
		after, err := wheelScroll(ctx, page, anchor, dx, dy)
		if err != nil {
			return "", err
		}
		if i < 0 { // report the innermost box that moved, else the innermost
			i = 0
			for j := 0; j < len(before) && j < len(after); j++ {
				if before[j].X != after[j].X || before[j].Y != after[j].Y {
					i = j
					break
				}
			}
		}
		if i >= len(after) || i >= len(before) {
			return "", errors.New("Failed to measure scroll")
		}
		a = after[i]
		end := atEnd(before[i], a)
		if end && maxSteps > 0 {
			// Lazy loads grow the page after the limit: settle and re-measure once.
			time.Sleep(wheelSettle)
			if again, err := measureScroll(anchor); err == nil && i < len(again) {
				end = again[i] == a || atLimit(again[i])
				a, after[i] = again[i], again[i]
			}
		}
		if end {
			suffix = " end"
			break
		}
		if maxSteps == 0 {
			break
		}
		if step >= maxSteps || time.Now().After(deadline) {
			suffix = " cap"
			break
		}
		before = after
	}

	b := first[min(i, len(first)-1)]
	movedX, movedY := a.X-b.X, a.Y-b.Y
	out := fmt.Sprintf("dy=%.0f y=%.0f/%.0f", movedY, a.Y, a.MaxY)
	if dxReq || movedX != 0 {
		out = fmt.Sprintf("dx=%.0f x=%.0f/%.0f ", movedX, a.X, a.MaxX) + out
	}
	return "scrolled " + out + suffix, nil
}

// wheelScroll sends dx,dy as ~100px wheel notches, then re-measures until two
// reads agree (smooth scrolling animates; max 1s).
func wheelScroll(ctx *types.Context, page *rod.Page, anchor *rod.Element, dx, dy float64) ([]scrollBox, error) {
	n := int(math.Ceil(math.Max(math.Abs(dx), math.Abs(dy)) / 100))
	for i := 0; i < n; i++ {
		if i > 0 {
			gap := time.Duration(20+rand.IntN(41)) * time.Millisecond
			if d := ctx.HumanizeTuning().ScrollDuration; d != nil {
				gap = time.Duration(*d) * time.Millisecond / time.Duration(n)
			}
			time.Sleep(gap)
		}
		if err := mouseScroll(page, dx/float64(n), dy/float64(n)); err != nil {
			return nil, errors.Wrap(err, "Failed to scroll mouse wheel")
		}
	}
	after, err := measureScroll(anchor)
	for deadline := time.Now().Add(time.Second); err == nil && time.Now().Before(deadline); {
		time.Sleep(50 * time.Millisecond)
		var next []scrollBox
		if next, err = measureScroll(anchor); err == nil && slices.Equal(next, after) {
			break
		}
		after = next
	}
	if err != nil {
		return nil, errors.Wrap(err, "Failed to measure scroll")
	}
	return after, nil
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
	var cur proto.TargetTargetID
	if p, err := ctx.ControlledPage(); err == nil {
		cur = p.TargetID
	}
	var res strings.Builder
	for i, p := range pages {
		mark := " "
		if p.TargetID == cur {
			mark = "*"
		}
		fmt.Fprintf(&res, "%s [%d] %s\n", mark, i, tabDesc(p))
	}
	return res.String(), nil
}

func TabNew(ctx *types.Context, url string) (string, error) {
	browser := ctx.GetBrowser()
	if browser == nil {
		return "", errors.New("No active browser")
	}
	page, err := browserPage(browser, proto.TargetCreateTarget{})
	if err != nil {
		return "", errors.Wrap(err, "Failed to create new tab")
	}
	ctx.SwitchTo(page)
	if url != "" && url != "about:blank" {
		if _, err := Navigate(ctx, url); err != nil {
			return "", err
		}
	}
	return "New tab created: " + tabDesc(page), nil
}

// shortID is the first 8 characters of p's TargetID: the tab id users type.
func shortID(p *rod.Page) string {
	id := string(p.TargetID)
	if len(id) > 8 {
		id = id[:8]
	}
	return id
}

// tabDesc is "<ID> <Title> (<url>)"; title and url are empty when Info fails.
func tabDesc(p *rod.Page) string {
	var title, url string
	if info, err := p.Info(); err == nil && info != nil {
		title, url = info.Title, info.URL
	}
	return fmt.Sprintf("%s %s (%s)", shortID(p), title, url)
}

func TabClose(ctx *types.Context, sel string) (string, error) {
	browser := ctx.GetBrowser()
	if browser == nil {
		return "", errors.New("No active browser")
	}
	pages, err := browserPages(browser)
	if err != nil {
		return "", err
	}
	i, err := resolveTab(pages, sel)
	if err != nil {
		return "", err
	}
	target := pages[i]
	tail := ""
	if cur, err := ctx.ControlledPage(); err == nil && cur.TargetID == target.TargetID {
		rest := append(pages[:i:i], pages[i+1:]...)
		if len(rest) == 0 {
			ctx.SwitchTo(nil)
			tail = "; no tabs left (goto opens one)"
		} else {
			j := max(i-1, 0)
			ctx.SwitchTo(rest[j])
			tail = fmt.Sprintf("; current: [%d] %s", j, tabDesc(rest[j]))
		}
	}
	if err := pageClose(target); err != nil {
		return "", errors.Wrap(err, "Failed to close tab")
	}
	return fmt.Sprintf("Closed tab %d %s%s", i, shortID(target), tail), nil
}

var tabIndexRe = regexp.MustCompile(`^[0-9]{1,7}$`)

// resolveTab maps a tab selector to an index in pages: fewer than 8 digits is
// an index; anything else is a unique, case-insensitive TargetID prefix.
func resolveTab(pages rod.Pages, sel string) (int, error) {
	if tabIndexRe.MatchString(sel) {
		i, _ := strconv.Atoi(sel)
		if i >= len(pages) {
			return 0, errors.Errorf("Tab index out of range: %d", i)
		}
		return i, nil
	}
	match := -1
	for i, p := range pages {
		if sel != "" && strings.HasPrefix(strings.ToUpper(string(p.TargetID)), strings.ToUpper(sel)) {
			if match >= 0 {
				return 0, errors.Errorf("Tab id %q is ambiguous", sel)
			}
			match = i
		}
	}
	if match < 0 {
		return 0, errors.Errorf("No tab matches %q", sel)
	}
	return match, nil
}

func TabSelect(ctx *types.Context, sel string) (string, error) {
	browser := ctx.GetBrowser()
	if browser == nil {
		return "", errors.New("No active browser")
	}
	pages, err := browserPages(browser)
	if err != nil {
		return "", err
	}
	i, err := resolveTab(pages, sel)
	if err != nil {
		return "", err
	}
	page := pages[i]
	if _, err := pageActivate(page); err != nil {
		return "", errors.Wrap(err, "Failed to activate tab")
	}
	out := fmt.Sprintf("Selected tab %d %s", i, tabDesc(page))
	if ctx.SwitchTo(page) {
		out += "; adopted, stealth applies from next navigation (reload to apply)"
	}
	return out, nil
}

// namedKeys maps `press` key names to rod keys (skills/rod-cli/SKILL.md lists them).
var namedKeys = map[string]input.Key{
	"Enter": input.Enter, "Tab": input.Tab, "Backspace": input.Backspace, "Escape": input.Escape,
	"Space": input.Space, "Delete": input.Delete, "Insert": input.Insert,
	"Home": input.Home, "End": input.End, "PageUp": input.PageUp, "PageDown": input.PageDown,
	"ArrowUp": input.ArrowUp, "ArrowDown": input.ArrowDown, "ArrowLeft": input.ArrowLeft, "ArrowRight": input.ArrowRight,
	"Shift": input.ShiftLeft, "Control": input.ControlLeft, "Alt": input.AltLeft, "Meta": input.MetaLeft,
	"CapsLock": input.CapsLock, "ContextMenu": input.ContextMenu,
	"F1": input.F1, "F2": input.F2, "F3": input.F3, "F4": input.F4, "F5": input.F5, "F6": input.F6,
	"F7": input.F7, "F8": input.F8, "F9": input.F9, "F10": input.F10, "F11": input.F11, "F12": input.F12,
}

// parseKey resolves a key name, or a single character on the US keyboard
// layout, to an input.Key; anything else is an `unknown key` error.
func parseKey(keyStr string) (input.Key, error) {
	if k, ok := namedKeys[keyStr]; ok {
		return k, nil
	}
	if r := []rune(keyStr); len(r) == 1 && keyDefined(input.Key(r[0])) {
		return input.Key(r[0]), nil
	}
	return 0, fmt.Errorf("unknown key %q", keyStr)
}

// keyDefined reports whether rod's keymap has k (Key.Info panics otherwise).
func keyDefined(k input.Key) (ok bool) {
	defer func() { ok = recover() == nil }()
	k.Info()
	return
}

// Press triggers a raw keyboard key press
func Press(ctx *types.Context, key string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	k, err := parseKey(key)
	if err != nil {
		return "", err
	}
	if err := keyboardPress(page.Keyboard, k); err != nil {
		return "", fmt.Errorf("failed to press key %s: %w", key, err)
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
	return fmt.Sprintf("Pressed key: %s", key), nil
}

// MouseMove moves the pointer along a humanized curve (honours --mouse-*).
func MouseMove(ctx *types.Context, x, y float64) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	if err := moveToPoint(page, humanize.Point{X: x, Y: y}, mouseOpts(ctx.HumanizeTuning())...); err != nil {
		return "", fmt.Errorf("failed to move mouse: %w", err)
	}
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

// GetCookies returns the current cookies, or only those named name when name is non-empty.
func GetCookies(ctx *types.Context, name string) (string, error) {
	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}
	cookies, err := browserGetCookies(page.Browser())
	if err != nil {
		return "", fmt.Errorf("failed to get cookies: %w", err)
	}

	res := "Cookies:\n"
	found := false
	for _, c := range cookies {
		if name != "" && c.Name != name {
			continue
		}
		found = true
		res += fmt.Sprintf("- %s: %s (domain: %s)\n", c.Name, c.Value, c.Domain)
	}
	if name != "" && !found {
		return "", fmt.Errorf("no cookie named %q", name)
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
	info, err := page.Info()
	if err != nil {
		return "", errors.Wrap(err, "failed to read page info")
	}
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
	info, err := page.Info()
	if err != nil {
		return "", errors.Wrap(err, "failed to read page info")
	}
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

func ConsoleLogs(ctx *types.Context, clear bool) (string, error) {
	logs, dropped := ctx.GetConsoleLogs()
	if clear {
		ctx.ClearConsoleLogs()
	}
	if len(logs) == 0 {
		return "No console logs", nil
	}
	return droppedLine(dropped) + strings.Join(logs, "\n"), nil
}

func NetworkRequests(ctx *types.Context, clear bool) (string, error) {
	reqs, dropped := ctx.GetRequests()
	if clear {
		ctx.ClearRequests()
	}
	if len(reqs) == 0 {
		return "No network requests", nil
	}
	var res strings.Builder
	res.WriteString(droppedLine(dropped))
	for i, req := range reqs {
		res.WriteString(fmt.Sprintf("[%d] %s\n", i, req))
	}
	return res.String(), nil
}

// droppedLine is the first output line telling the agent the capture cap
// evicted older entries; empty when nothing was dropped.
func droppedLine(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("[dropped %d older entries]\n", n)
}

func NetworkRequest(ctx *types.Context, index int) (string, error) {
	reqs, _ := ctx.GetRequests()
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
	for _, r := range routes {
		res.WriteString(fmt.Sprintf("- %s\n", r.Pattern))
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

	opts := mouseOpts(ctx.HumanizeTuning())
	if err := moveToElement(page, el1, opts...); err != nil {
		return "", errors.Wrap(err, "Failed to move to drag source")
	}
	if err := pageMouseDown(page, proto.InputMouseButtonLeft, 1); err != nil {
		return "", errors.Wrap(err, "Failed to press mouse on drag source")
	}
	held := true
	defer func() {
		if held { // never leave the button stuck down on an error path
			_ = pageMouseUp(page, proto.InputMouseButtonLeft, 1)
		}
	}()
	if err := moveToElement(page, el2, opts...); err != nil {
		return "", errors.Wrap(err, "Failed to move to drag target")
	}
	held = false
	if err := pageMouseUp(page, proto.InputMouseButtonLeft, 1); err != nil {
		return "", errors.Wrap(err, "Failed to release mouse on drag target")
	}
	page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
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
