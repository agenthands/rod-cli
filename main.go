package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"
	"github.com/agenthands/rod-cli/types"
	"github.com/urfave/cli/v2"
)

func runMCPServer(c *cli.Context) error {
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
	cfg.Raw = c.Bool("raw")
	cfg.Json = c.Bool("json")

	types.InitLogger(cfg.LoggerConfig)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := NewRunner(ctx, *cfg)
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
		defer signal.Stop(ch)
		<-ch
		log.Info("Received signal, exiting...")
		cancel()
	}()
	runner.Run()

	err = runner.Close()
	if err != nil {
		log.Errorf("Server close error: %s", err)
	}
	return nil
}

func main() {
	app := getApp()
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
