package daemon

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/agenthands/rod-cli/types"
)

func TestPanickingActionDoesNotWedgeSession(t *testing.T) {
	privateTmp(t)
	origRun, origAcq := runAction, execAcquireTimeout
	t.Cleanup(func() { runAction, execAcquireTimeout = origRun, origAcq })
	execAcquireTimeout = 500 * time.Millisecond
	runAction = func(ctx *types.Context, req Request) (string, error) {
		if req.Command == "boom" {
			panic("test-only panicking action")
		}
		return executeAction(ctx, req)
	}
	session, _ := startBareServer(t)

	_, _ = ClientExecute(session, Request{Command: "boom"})
	_, err := ClientExecute(session, Request{Command: "bogus-cmd"})
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("command after a panic: err = %v, want unknown command (session wedged)", err)
	}
}

func TestExecuteActionRejectsNonNumericArgs(t *testing.T) {
	ctx := types.NewContext(context.Background(), types.Config{CDPEndpoint: "ws://127.0.0.1:1/bogus"})
	cases := []struct {
		cmd  string
		args map[string]string
		bad  string
	}{
		{"request", map[string]string{"index": "x"}, `"x"`},
		{"resize", map[string]string{"width": "abc", "height": "600"}, `"abc"`},
		{"mousemove", map[string]string{"x": "1", "y": "y"}, `"y"`},
		{"mousewheel", map[string]string{"dx": "0", "dy": "abc"}, `"abc"`},
	}
	for _, c := range cases {
		_, err := executeAction(ctx, Request{Command: c.cmd, Args: c.args})
		if err == nil || !strings.Contains(err.Error(), c.bad) {
			t.Errorf("%s %v: err = %v, want an error naming %s", c.cmd, c.args, err, c.bad)
		}
	}
}
