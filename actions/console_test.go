package actions

import (
	"strings"
	"testing"
	"time"

	"github.com/agenthands/rod-cli/types"
)

func newConsoleCtx(t *testing.T) (*types.Context, func()) {
	on := true
	ctx, cleanup := newLiveCtxCfg(t, func(c *types.Config) { c.Stealth.ConsoleCapture = &on })
	srv := newFixtureServer()
	navFixture(t, ctx, srv)
	return ctx, func() { srv.Close(); cleanup() }
}

func TestConsoleRecordsUncaughtException(t *testing.T) {
	ctx, cleanup := newConsoleCtx(t)
	defer cleanup()
	if _, err := Evaluate(ctx, "setTimeout(()=>{throw new Error('boom')},0)", ""); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	out, err := ConsoleLogs(ctx, false)
	if err != nil || !strings.Contains(out, "[exception]") || !strings.Contains(out, "boom") {
		t.Errorf("ConsoleLogs = %q, err=%v; want [exception] ... boom", out, err)
	}
}
