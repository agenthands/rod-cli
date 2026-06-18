package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/agenthands/rod-cli/actions"
	"github.com/agenthands/rod-cli/banner"
	"github.com/agenthands/rod-cli/types"
	"github.com/urfave/cli/v2"
)

func runWithContext(c *cli.Context, actionFunc func(rodCtx *types.Context) (string, error)) error {
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

	rodCtx := types.NewContext(context.Background(), *cfg)
	defer rodCtx.Close()

	msg, err := actionFunc(rodCtx)
	if err != nil {
		if cfg.Json {
			out, _ := json.Marshal(map[string]string{"error": err.Error()})
			fmt.Println(string(out))
		} else {
			fmt.Println("Error:", err)
		}
		return err
	}

	if cfg.Json {
		out, _ := json.Marshal(map[string]string{"result": msg})
		fmt.Println(string(out))
	} else if cfg.Raw {
		fmt.Println(msg)
	} else {
		fmt.Println(msg)
	}

	return nil
}

func getApp() *cli.App {
	return &cli.App{
		Name:        "rod-cli",
		Description: "Native web browsing, scraping, and interaction CLI for AI assistants",
		Usage:       "rod-cli [command] [options]",
		Version:     banner.Version,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "config file path"},
			&cli.StringFlag{Name: "cdp-endpoint", Aliases: []string{"cdp"}, Usage: "control browser by cdp"},
			&cli.BoolFlag{Name: "headless", Aliases: []string{"hl"}, Usage: "enable headless"},
			&cli.BoolFlag{Name: "no-banner", Aliases: []string{"nb"}, Usage: "disable show banner"},
			&cli.BoolFlag{Name: "vision", Aliases: []string{"vs"}, Usage: "support vision LLM"},
			&cli.BoolFlag{Name: "raw", Usage: "output raw results"},
			&cli.BoolFlag{Name: "json", Usage: "output structured json"},
		},
		Before: func(c *cli.Context) error {
			if !c.Bool("no-banner") {
				fmt.Println(banner.ShowBanner())
			}
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "serve",
				Usage: "Run the MCP server (default behavior when no command is provided)",
				Action: func(c *cli.Context) error {
					return runMCPServer(c)
				},
			},
			{
				Name:    "open",
				Aliases: []string{"goto"},
				Usage:   "Navigate to a URL",
				Action: func(c *cli.Context) error {
					url := c.Args().First()
					if url == "" {
						return fmt.Errorf("URL is required")
					}
					return runWithContext(c, func(rodCtx *types.Context) (string, error) {
						return actions.Navigate(rodCtx, url)
					})
				},
			},
			{
				Name:  "go-back",
				Usage: "Go back in the browser history",
				Action: func(c *cli.Context) error {
					return runWithContext(c, func(rodCtx *types.Context) (string, error) {
						return actions.GoBack(rodCtx)
					})
				},
			},
			{
				Name:  "go-forward",
				Usage: "Go forward in the browser history",
				Action: func(c *cli.Context) error {
					return runWithContext(c, func(rodCtx *types.Context) (string, error) {
						return actions.GoForward(rodCtx)
					})
				},
			},
			{
				Name:  "reload",
				Usage: "Reload the current page",
				Action: func(c *cli.Context) error {
					return runWithContext(c, func(rodCtx *types.Context) (string, error) {
						return actions.Reload(rodCtx)
					})
				},
			},
			{
				Name:  "click",
				Usage: "Click an element by its ref",
				Action: func(c *cli.Context) error {
					ref := c.Args().First()
					if ref == "" {
						return fmt.Errorf("element ref is required")
					}
					return runWithContext(c, func(rodCtx *types.Context) (string, error) {
						return actions.Click(rodCtx, ref)
					})
				},
			},
			{
				Name:  "dblclick",
				Usage: "Double click an element by its ref",
				Action: func(c *cli.Context) error {
					ref := c.Args().First()
					if ref == "" {
						return fmt.Errorf("element ref is required")
					}
					return runWithContext(c, func(rodCtx *types.Context) (string, error) {
						return actions.DblClick(rodCtx, ref)
					})
				},
			},
			{
				Name:  "type",
				Usage: "Type text into an element",
				Action: func(c *cli.Context) error {
					ref := c.Args().Get(0)
					text := c.Args().Get(1)
					if ref == "" || text == "" {
						return fmt.Errorf("element ref and text are required")
					}
					return runWithContext(c, func(rodCtx *types.Context) (string, error) {
						return actions.Type(rodCtx, ref, text)
					})
				},
			},
			{
				Name:  "fill",
				Usage: "Fill text into an element",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "submit", Usage: "Press enter after filling"},
				},
				Action: func(c *cli.Context) error {
					ref := c.Args().Get(0)
					text := c.Args().Get(1)
					if ref == "" || text == "" {
						return fmt.Errorf("element ref and text are required")
					}
					return runWithContext(c, func(rodCtx *types.Context) (string, error) {
						return actions.Fill(rodCtx, ref, text, c.Bool("submit"))
					})
				},
			},
			{
				Name:  "hover",
				Usage: "Hover an element by its ref",
				Action: func(c *cli.Context) error {
					ref := c.Args().First()
					if ref == "" {
						return fmt.Errorf("element ref is required")
					}
					return runWithContext(c, func(rodCtx *types.Context) (string, error) {
						return actions.Hover(rodCtx, ref)
					})
				},
			},
			{
				Name:  "select",
				Usage: "Select options in an element",
				Action: func(c *cli.Context) error {
					ref := c.Args().First()
					values := c.Args().Slice()[1:]
					if ref == "" || len(values) == 0 {
						return fmt.Errorf("element ref and values are required")
					}
					return runWithContext(c, func(rodCtx *types.Context) (string, error) {
						return actions.Select(rodCtx, ref, values)
					})
				},
			},
			{
				Name:  "eval",
				Usage: "Evaluate a JS script",
				Action: func(c *cli.Context) error {
					script := c.Args().First()
					if script == "" {
						return fmt.Errorf("script is required")
					}
					return runWithContext(c, func(rodCtx *types.Context) (string, error) {
						return actions.Evaluate(rodCtx, script)
					})
				},
			},
			{
				Name:  "snapshot",
				Usage: "Take a snapshot of the current page",
				Action: func(c *cli.Context) error {
					return runWithContext(c, func(rodCtx *types.Context) (string, error) {
						return actions.Snapshot(rodCtx)
					})
				},
			},
			{
				Name:  "screenshot",
				Usage: "Take a screenshot of the current page",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "Name of the screenshot"},
					&cli.StringFlag{Name: "selector", Usage: "CSS selector"},
					&cli.Float64Flag{Name: "width", Usage: "Width"},
					&cli.Float64Flag{Name: "height", Usage: "Height"},
				},
				Action: func(c *cli.Context) error {
					name := c.String("name")
					if name == "" {
						name = "screenshot"
					}
					return runWithContext(c, func(rodCtx *types.Context) (string, error) {
						return actions.Screenshot(rodCtx, name, c.String("selector"), c.Float64("width"), c.Float64("height"))
					})
				},
			},
			{
				Name:  "pdf",
				Usage: "Export page to PDF",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "Name of the pdf"},
				},
				Action: func(c *cli.Context) error {
					name := c.String("name")
					if name == "" {
						name = "page"
					}
					return runWithContext(c, func(rodCtx *types.Context) (string, error) {
						return actions.Pdf(rodCtx, name)
					})
				},
			},
		},
		Action: func(c *cli.Context) error {
			return runMCPServer(c)
		},
	}
}
