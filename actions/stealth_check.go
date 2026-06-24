package actions

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/agenthands/rod-cli/internal/detect"
	"github.com/agenthands/rod-cli/types"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
)

// signalVerdict is the per-signal result read from the LIVE page. Value is the
// string-coerced live value; Pass/Reason are the codified VALIDATE-01 verdict.
type signalVerdict struct {
	Verdict string `json:"verdict"`          // "PASS" or "FAIL"
	Value   string `json:"value"`            // the live window.__detect.<signal> value
	Reason  string `json:"reason,omitempty"` // failure reason (only when FAIL)
}

// stealthSignalOrder is the stable display/iteration order for the per-signal
// table and json. The names MUST match the shared probe (internal/detect/probe.js)
// and the detect.js harness — same source of truth, no divergence.
var stealthSignalOrder = []string{
	"webdriver",
	"pluginsLength",
	"userAgent",
	"webglVendor",
	"webglRenderer",
	"permissionsConsistent",
	"languages",
	"screen",
	"windowChrome",
	"chromeRuntime",
	"timezone",
}

// StealthCheck injects the shared table-stakes probe, reads each signal back
// from the LIVE page (never from a Go config field), applies the codified
// VALIDATE-01 thresholds, and formats the result per the requested mode:
//
//   - jsonOut: a structured object {signal -> {verdict, value, reason}} + overall.
//   - raw:     a single line — "PASS", or "FAIL" followed by ONLY the failing
//     signals as "name=FAIL(reason)" (never a full-signal dump).
//   - default: a human-readable aligned per-signal table with an overall verdict.
//
// If url != "" the action navigates there first; otherwise it checks the
// current page. The returned string is exactly what should reach stdout for the
// requested mode; warnings (e.g. a signal read error) are folded into the
// per-signal value/reason, never into a separate stdout line.
func StealthCheck(ctx *types.Context, url string, raw bool, jsonOut bool) (string, error) {
	// (1) Navigate first only when a url is supplied; empty checks current page.
	if url != "" {
		if _, err := Navigate(ctx, url); err != nil {
			return "", err
		}
	}

	page, err := ctx.ControlledPage()
	if err != nil {
		return "", err
	}

	// (2) Inject the shared probe and wait for window.__detect.ready (so async
	// permissions has settled). Mirror the harness waitForDetectReady cadence
	// (~30 x 300ms). Inject every poll attempt is unnecessary; inject once.
	if err := evalProbe(page, detect.Probe); err != nil {
		return "", errors.Wrap(err, "Failed to inject stealth probe")
	}
	if err := waitProbeReady(page); err != nil {
		return "", err
	}

	// (3) Read each signal back from the LIVE page (String-coerced) and (4) apply
	// the codified VALIDATE-01 thresholds.
	verdicts := make(map[string]signalVerdict, len(stealthSignalOrder))
	for _, name := range stealthSignalOrder {
		val := readSignal(page, name)
		verdicts[name] = computeVerdict(name, val)
	}

	allPass := true
	for _, name := range stealthSignalOrder {
		if verdicts[name].Verdict != "PASS" {
			allPass = false
			break
		}
	}

	// (5) Format per requested mode.
	switch {
	case jsonOut:
		return formatJSON(verdicts, allPass)
	case raw:
		return formatRaw(verdicts, allPass), nil
	default:
		return formatHuman(verdicts, allPass), nil
	}
}

// evalProbe injects the probe IIFE into the live page. detect.Probe is a fixed
// embedded string with NO user-input interpolation — the only attacker-reachable
// inputs (url, signal values) never reach this eval.
func evalProbe(page *rod.Page, probe string) error {
	_, err := runtimeEvaluateCall(proto.RuntimeEvaluate{
		Expression:   probe,
		AwaitPromise: true,
	}, page)
	return err
}

// waitProbeReady polls window.__detect.ready until the async permissions probe
// has settled into the global. Bounded retry; mirrors the harness cadence.
func waitProbeReady(page *rod.Page) error {
	for i := 0; i < 30; i++ {
		r, err := runtimeEvaluateCall(proto.RuntimeEvaluate{
			Expression:   "!!(window.__detect && window.__detect.ready === true)",
			AwaitPromise: true,
		}, page)
		if err == nil && r != nil && r.Result != nil {
			if b, ok := r.Result.Value.Val().(bool); ok && b {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return errors.New("stealth probe did not become ready (window.__detect.ready never true)")
}

// readSignal reads a single window.__detect.<name> back from the LIVE page,
// String-coerced so booleans/numbers/objects all serialize comparably. A read
// error is folded into the returned value (prefixed "read-error:") so one bad
// read never blanks the whole verdict set.
func readSignal(page *rod.Page, name string) string {
	r, err := runtimeEvaluateCall(proto.RuntimeEvaluate{
		Expression:   "String(window.__detect." + name + ")",
		AwaitPromise: true,
	}, page)
	if err != nil {
		return "read-error: " + err.Error()
	}
	if r == nil || r.Result == nil {
		return "read-error: no-result"
	}
	if r.ExceptionDetails != nil {
		return "read-error: exception"
	}
	return strings.TrimSpace(fmt.Sprintf("%v", r.Result.Value.Val()))
}

// computeVerdict applies the codified VALIDATE-01 threshold for a single signal.
// Thresholds (all read from the live page):
//
//	webdriver             -> must be "false" or "undefined" (masked/deleted by
//	                         stealth); only "true" is the automation tell
//	pluginsLength         -> integer > 0
//	userAgent             -> must NOT contain "HeadlessChrome"
//	webglVendor           -> not software (swiftshader/llvmpipe/software) and not
//	                         a no-context/no-extension sentinel
//	webglRenderer         -> same as webglVendor
//	permissionsConsistent -> must be "true"
//	languages             -> non-empty
//	screen                -> "WxH" with both > 0
//	windowChrome          -> must be "true"
//	chromeRuntime         -> must be "true"
//	timezone              -> non-empty IANA zone containing "/"
func computeVerdict(name, val string) signalVerdict {
	pass := func() signalVerdict { return signalVerdict{Verdict: "PASS", Value: val} }
	fail := func(reason string) signalVerdict {
		return signalVerdict{Verdict: "FAIL", Value: val, Reason: reason}
	}

	// A read error always fails the signal.
	if strings.HasPrefix(val, "read-error:") || strings.HasPrefix(val, "error:") {
		return fail(val)
	}

	switch name {
	case "webdriver":
		// godoll's scriptHideAutomation masks/deletes navigator.webdriver, so on a
		// correctly-stealthed page it reads back as `undefined` (String-coerced to
		// "undefined"). A genuine human browser reads "false". Both are non-tells;
		// only "true" (or a read-error, handled above) is the automation tell.
		if val == "false" || val == "undefined" {
			return pass()
		}
		return fail(val) // only "true" (the automation tell) fails here

	case "pluginsLength":
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			return pass()
		}
		return fail("0")

	case "userAgent":
		if strings.Contains(val, "HeadlessChrome") {
			return fail("HeadlessChrome")
		}
		return pass()

	case "webglVendor", "webglRenderer":
		lower := strings.ToLower(val)
		if val == "no-context" || val == "no-extension" {
			return fail(val)
		}
		for _, bad := range []string{"swiftshader", "llvmpipe", "software"} {
			if strings.Contains(lower, bad) {
				return fail(val)
			}
		}
		return pass()

	case "permissionsConsistent":
		if val == "true" {
			return pass()
		}
		return fail(val)

	case "languages":
		if val != "" {
			return pass()
		}
		return fail("empty")

	case "screen":
		if w, h, ok := parseScreen(val); ok && w > 0 && h > 0 {
			return pass()
		}
		return fail(val)

	case "windowChrome":
		if val == "true" {
			return pass()
		}
		return fail("absent")

	case "chromeRuntime":
		if val == "true" {
			return pass()
		}
		return fail("absent")

	case "timezone":
		if val != "" && strings.Contains(val, "/") {
			return pass()
		}
		return fail(val)
	}

	return pass()
}

// parseScreen parses a "WxH" dimension string into integers.
func parseScreen(val string) (int, int, bool) {
	parts := strings.SplitN(val, "x", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	w, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil {
		return 0, 0, false
	}
	return w, h, true
}

// formatRaw emits the single-line --raw form: "PASS", or "FAIL" followed by ONLY
// the failing signals as "name=FAIL(reason)" — never a full-signal dump.
func formatRaw(verdicts map[string]signalVerdict, allPass bool) string {
	if allPass {
		return "PASS"
	}
	tokens := []string{"FAIL"}
	for _, name := range stealthSignalOrder {
		v := verdicts[name]
		if v.Verdict != "PASS" {
			tokens = append(tokens, fmt.Sprintf("%s=FAIL(%s)", name, v.Reason))
		}
	}
	return strings.Join(tokens, " ")
}

// formatHuman emits an aligned per-signal table with an overall verdict line.
func formatHuman(verdicts map[string]signalVerdict, allPass bool) string {
	width := 0
	for _, name := range stealthSignalOrder {
		if len(name) > width {
			width = len(name)
		}
	}
	var b strings.Builder
	overall := "FAIL"
	if allPass {
		overall = "PASS"
	}
	fmt.Fprintf(&b, "stealth-check: %s\n", overall)
	for _, name := range stealthSignalOrder {
		v := verdicts[name]
		line := fmt.Sprintf("  %-*s  %-4s  %s", width, name, v.Verdict, v.Value)
		if v.Verdict != "PASS" && v.Reason != "" {
			line += fmt.Sprintf("  (%s)", v.Reason)
		}
		b.WriteString(strings.TrimRight(line, " "))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// formatJSON emits a structured object: per-signal verdicts + overall.
func formatJSON(verdicts map[string]signalVerdict, allPass bool) (string, error) {
	overall := "FAIL"
	if allPass {
		overall = "PASS"
	}
	// Stable signal ordering in the marshaled object is not guaranteed by Go's
	// map marshaling, so include the order explicitly for deterministic readers.
	out := map[string]interface{}{
		"overall": overall,
		"signals": verdicts,
		"order":   append([]string(nil), stealthSignalOrder...),
	}
	bytes, err := json.Marshal(out)
	if err != nil {
		return "", errors.Wrap(err, "Failed to marshal stealth-check json")
	}
	return string(bytes), nil
}
