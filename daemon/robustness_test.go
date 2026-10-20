package daemon

import (
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
