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
	// Phase-33 fingerprint-dimension toggles: capture as *bool only when explicitly
	// set so an unset flag leaves the field nil = keep the default-true baseline.
	if c.IsSet("font-spoof") {
		v := c.Bool("font-spoof")
		stealthFlags.FontSpoof = &v
	}
	if c.IsSet("media-devices-spoof") {
		v := c.Bool("media-devices-spoof")
		stealthFlags.MediaDevicesSpoof = &v
	}
	if c.IsSet("battery-spoof") {
		v := c.Bool("battery-spoof")
		stealthFlags.BatterySpoof = &v
	}
	if c.IsSet("codec-spoof") {
		v := c.Bool("codec-spoof")
		stealthFlags.CodecSpoof = &v
	}
	// Phase-30 CDP-footprint capture toggles: capture as *bool only when explicitly
	// set so an unset flag leaves the field nil = keep the default-OFF baseline.
	if c.IsSet("console-capture") {
		v := c.Bool("console-capture")
		stealthFlags.ConsoleCapture = &v
	}
	if c.IsSet("request-capture") {
		v := c.Bool("request-capture")
		stealthFlags.RequestCapture = &v
	}
	// CDP-DEEP-01 proxy toggles: capture only when explicitly set on the daemon
	// argv, so an unset flag stays nil (= keep the default-OFF baseline). Without
	// this the forwarded --cdp-proxy never reaches StealthFlags and the CDP proxy
	// silently never enables (CONCERNS landmine #1: forward → CAPTURE → resolve).
	if c.IsSet("cdp-proxy") {
		v := c.Bool("cdp-proxy")
		stealthFlags.CDPProxy = &v
	}
	if c.IsSet("cdp-jitter-ms") {
		v := c.Int("cdp-jitter-ms")
		stealthFlags.CDPJitterMs = &v
	}
	if c.IsSet("no-cdp-proxy") {
		v := c.Bool("no-cdp-proxy")
		stealthFlags.NoCDPProxy = &v
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

	log.Infof("Starting daemon for session %s", session)
	if err := daemon.StartServer(session, rodCtx); err != nil {
		log.Errorf("Daemon error: %s", err)
		return err
	}
	return nil
}

func main() {
	app := getApp()
	args, err := reorderGlobalFlags(app, os.Args)
	if err == nil {
		err = app.Run(args)
	}
	if err != nil {
		// Before records the parsed --json; an error before parsing stays plain.
		if isJSON, _ := app.Metadata["json"].(bool); isJSON {
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
