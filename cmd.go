package main

import (

	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/agenthands/rod-cli/banner"
	"github.com/agenthands/rod-cli/daemon"
	"github.com/urfave/cli/v2"
)

func runClientCommand(c *cli.Context, req daemon.Request) error {
	session := c.String("session")
	
	// Format generic daemon-spawn flags from global cli args
	flags := []string{}
	if c.String("config") != "" { flags = append(flags, "--config", c.String("config")) }
	if c.String("cdp-endpoint") != "" { flags = append(flags, "--cdp-endpoint", c.String("cdp-endpoint")) }
	if c.Bool("headless") { flags = append(flags, "--headless") }
	if c.Bool("vision") { flags = append(flags, "--vision") }
	
	err := daemon.EnsureDaemon(session, os.Args[0], flags)
	if err != nil {
		return fmt.Errorf("failed to ensure daemon: %v", err)
	}

	msg, err := daemon.ClientExecute(session, req)
	if err != nil {
		if c.Bool("json") {
			out, _ := json.Marshal(map[string]string{"error": err.Error()})
			fmt.Println(string(out))
		} else {
			fmt.Println("Error:", err)
		}
		return err
	}

	if c.Bool("json") {
		out, _ := json.Marshal(map[string]string{"result": msg})
		fmt.Println(string(out))
	} else if c.Bool("raw") {
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
			&cli.StringFlag{Name: "session", Aliases: []string{"s"}, Usage: "named session", Value: "default"},
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
				Name:   "daemon",
				Hidden: true,
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "ppid"},
				},
				Action: func(c *cli.Context) error {
					return runDaemonServer(c)
				},
			},
			{
				Name:  "close",
				Usage: "Close the daemon session and the browser",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "close"})
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
					return runClientCommand(c, daemon.Request{Command: "open", Args: map[string]string{"url": url}})
				},
			},
			{
				Name:  "go-back",
				Usage: "Go back in the browser history",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "go-back"})
				},
			},
			{
				Name:  "go-forward",
				Usage: "Go forward in the browser history",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "go-forward"})
				},
			},
			{
				Name:  "reload",
				Usage: "Reload the current page",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "reload"})
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
					return runClientCommand(c, daemon.Request{Command: "click", Args: map[string]string{"ref": ref}})
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
					return runClientCommand(c, daemon.Request{Command: "dblclick", Args: map[string]string{"ref": ref}})
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
					return runClientCommand(c, daemon.Request{Command: "type", Args: map[string]string{"ref": ref, "text": text}})
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
					return runClientCommand(c, daemon.Request{Command: "fill", Args: map[string]string{"ref": ref, "text": text, "submit": fmt.Sprint(c.Bool("submit"))}})
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
					return runClientCommand(c, daemon.Request{Command: "hover", Args: map[string]string{"ref": ref}})
				},
			},
			{
				Name:  "select",
				Usage: "Select options in an element",
				Action: func(c *cli.Context) error {
					ref := c.Args().First()
					values := strings.Join(c.Args().Slice()[1:], ",")
					if ref == "" || values == "" {
						return fmt.Errorf("element ref and values are required")
					}
					return runClientCommand(c, daemon.Request{Command: "select", Args: map[string]string{"ref": ref, "values": values}})
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
					return runClientCommand(c, daemon.Request{Command: "eval", Args: map[string]string{"script": script}})
				},
			},
			{
				Name:  "snapshot",
				Usage: "Take a snapshot of the current page",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "snapshot"})
				},
			},
			{
				Name:  "screenshot",
				Usage: "Take a screenshot of the current page",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "Name of the screenshot"},
					&cli.StringFlag{Name: "selector", Usage: "CSS selector"},
				},
				Action: func(c *cli.Context) error {
					name := c.String("name")
					if name == "" {
						name = "screenshot"
					}
					return runClientCommand(c, daemon.Request{Command: "screenshot", Args: map[string]string{"name": name, "selector": c.String("selector")}})
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
					return runClientCommand(c, daemon.Request{Command: "pdf", Args: map[string]string{"name": name}})
				},
			},
		},
		Action: func(c *cli.Context) error {
			return runMCPServer(c)
		},
	}
}
