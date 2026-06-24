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
