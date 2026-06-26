package actions

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agenthands/rod-cli/types"
	"github.com/pkg/errors"
)

// CDPTraffic reads the CDP proxy's traffic log and formats it for display.
// Requires the CDP proxy to be enabled (--cdp-proxy). Returns a descriptive
// error if the proxy is not active.
func CDPTraffic(ctx *types.Context, jsonOut bool) (string, error) {
	proxy := ctx.GetCDPProxy()
	if proxy == nil {
		return "", errors.New("CDP proxy is not enabled — start a session with --cdp-proxy")
	}

	msgs := proxy.Traffic()
	if jsonOut {
		out, err := json.Marshal(msgs)
		if err != nil {
			return "", errors.Wrap(err, "marshal cdp traffic")
		}
		return string(out), nil
	}

	if len(msgs) == 0 {
		return "No CDP traffic logged yet. Navigate a page first.", nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "CDP traffic log (%d messages):\n", len(msgs))
	for i, msg := range msgs {
		// Truncate long messages for readability
		preview := string(msg.Raw)
		if len(preview) > 120 {
			preview = preview[:120] + "..."
		}
		fmt.Fprintf(&b, "  %3d  %-4s  %s\n", i+1, strings.ToUpper(msg.Direction), preview)
	}
	return b.String(), nil
}
