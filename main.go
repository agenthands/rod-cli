package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/agenthands/rod-cli/daemon"
	"github.com/agenthands/rod-cli/types"
	"github.com/urfave/cli/v2"
)

func runDaemonServer(c *cli.Context) error {
	cfg, err := types.LoadConfig(c.String("config"))
	if err != nil {
		return err
	}
	if c.Bool("headless") {
		cfg.Headless = true
	}
	if c.Bool("vision") {
		cfg.Mode = types.Vision
	}
	if cdp := c.String("cdp-endpoint"); cdp != "" {
		cfg.CDPEndpoint = cdp
	}

	// Resolve the stealth surface ONCE, before NewContext freezes Config, using
	// the precedence CLI flag > profile file > built-in default. The forwarded
	// --proxy/--proxy-auth/--profile are available on the same cli.Context because
	// EnsureDaemon spawned `rod-cli --session <s> --proxy ... daemon`. A bad
	// profile load fails the daemon loudly rather than shipping a half-resolved
	// identity. Credentials are never logged here (they would otherwise reach the
	// daemon log file).
	stealthFlags := types.StealthFlags{
		Proxy: c.String("proxy"),
		// proxy-auth is passed out-of-band via the environment (never the daemon
		// argv) so the credential is not exposed in /proc/<pid>/cmdline or `ps`.
		// The client sets ROD_CLI_PROXY_AUTH on the spawned daemon (see cmd.go).
		ProxyAuth: os.Getenv("ROD_CLI_PROXY_AUTH"),
		Profile:   c.String("profile"),
		// The 4 curated fingerprint pins arrive verbatim on the daemon argv.
		UserAgent: c.String("user-agent"),
		Locale:    c.String("locale"),
		Timezone:  c.String("timezone"),
		Platform:  c.String("platform"),
	}
	// Phase-27 hardening toggles: capture as *bool only when explicitly set so an
	// unset flag leaves the field nil = keep the default-true baseline.
	if c.IsSet("webrtc-protection") {
		v := c.Bool("webrtc-protection")
		stealthFlags.WebRTCLeakProtection = &v
	}
	if c.IsSet("canvas-noise") {
		v := c.Bool("canvas-noise")
		stealthFlags.CanvasNoise = &v
	}
	// Phase-28 humanize tuning: capture each as a non-nil pointer ONLY when the
	// flag is set on the daemon argv, so an unset knob stays nil (= emit no godoll
	// option = byte-for-byte default behavior).
	if c.IsSet("typing-speed-min") {
		v := c.Int("typing-speed-min")
		stealthFlags.TypingSpeedMin = &v
	}
	if c.IsSet("typing-speed-max") {
		v := c.Int("typing-speed-max")
		stealthFlags.TypingSpeedMax = &v
	}
	if c.IsSet("typo-rate") {
		v := float32(c.Float64("typo-rate"))
		stealthFlags.TypoRate = &v
	}
	if c.IsSet("mouse-tremor") {
		v := c.Bool("mouse-tremor")
		stealthFlags.MouseTremor = &v
	}
	if c.IsSet("mouse-steps") {
		v := c.Int("mouse-steps")
		stealthFlags.MouseSteps = &v
	}
	if c.IsSet("mouse-speed-min") {
		v := c.Int("mouse-speed-min")
		stealthFlags.MouseSpeedMin = &v
	}
	if c.IsSet("mouse-speed-max") {
		v := c.Int("mouse-speed-max")
		stealthFlags.MouseSpeedMax = &v
	}
	if c.IsSet("mouse-deviation") {
		v := c.Float64("mouse-deviation")
		stealthFlags.MouseDeviation = &v
	}
	if c.IsSet("scroll-duration") {
		v := c.Int("scroll-duration")
		stealthFlags.ScrollDuration = &v
	}
	if c.IsSet("scroll-physics") {
		v := c.Bool("scroll-physics")
		stealthFlags.ScrollPhysics = &v
	}
	if err := types.ResolveStealth(cfg, &stealthFlags); err != nil {
		return err
	}

	types.InitLogger(cfg.LoggerConfig)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rodCtx := types.NewContext(ctx, *cfg)
	defer rodCtx.Close()

	session := c.String("session")
	ppid := c.Int("ppid")

	log.Infof("Starting daemon for session %s (ppid: %d)", session, ppid)
	if err := daemon.StartServer(session, ppid, rodCtx); err != nil {
		log.Errorf("Daemon error: %s", err)
		return err
	}
	return nil
}

func main() {
	app := getApp()
	if err := app.Run(os.Args); err != nil {
		isJson := false
		for _, arg := range os.Args {
			if arg == "--json" {
				isJson = true
				break
			}
		}
		if isJson {
			// Extract just the underlying error message if it's wrapped by cli.Exit
			msg := err.Error()
			out, _ := json.Marshal(map[string]string{"error": msg})
			fmt.Println(string(out))
			os.Exit(1)
		} else {
			log.Fatal(err)
		}
	}
}
