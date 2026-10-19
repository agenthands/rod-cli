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

func TestConsoleClear(t *testing.T) {
	ctx, cleanup := newConsoleCtx(t)
	defer cleanup()
	_, _ = Evaluate(ctx, "console.log('keep me')", "")
	time.Sleep(200 * time.Millisecond)
	if out, err := ConsoleLogs(ctx, true); err != nil || !strings.Contains(out, "[log] keep me") {
		t.Fatalf("ConsoleLogs(clear) = %q, err=%v; want the entries", out, err)
	}
	if out, err := ConsoleLogs(ctx, false); err != nil || out != "No console logs" {
		t.Errorf("ConsoleLogs after clear = %q, err=%v; want \"No console logs\"", out, err)
	}
}

func TestConsoleBufferCapped(t *testing.T) {
	ctx, cleanup := newConsoleCtx(t)
	defer cleanup()
	if _, err := Evaluate(ctx, "for (let i = 0; i < 2100; i++) console.log(i)", ""); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	var logs []string
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		logs, _ = ctx.GetConsoleLogs()
		if len(logs) > 0 && logs[len(logs)-1] == "[log] 2099" {
			break
		}
	}
	if len(logs) != 2000 || logs[0] != "[log] 100" {
		first := ""
		if len(logs) > 0 {
			first = logs[0]
		}
		t.Errorf("got %d entries, first %q; want 2000, first \"[log] 100\"", len(logs), first)
	}
}

func TestConsoleReportsDropped(t *testing.T) {
	ctx, cleanup := newConsoleCtx(t)
	defer cleanup()
	if _, err := Evaluate(ctx, "for (let i = 0; i < 2100; i++) console.log(i)", ""); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	var out string
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		out, _ = ConsoleLogs(ctx, false)
		if strings.HasSuffix(out, "[log] 2099") {
			break
		}
	}
	lines := strings.SplitN(out, "\n", 3)
	if len(lines) < 2 || lines[0] != "[dropped 100 older entries]" || lines[1] != "[log] 100" {
		t.Errorf("first lines = %q; want [\"[dropped 100 older entries]\" \"[log] 100\"]", lines[:min(2, len(lines))])
	}
}
