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
