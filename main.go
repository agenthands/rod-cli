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
