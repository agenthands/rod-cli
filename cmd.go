package main

import (

	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/agenthands/rod-cli/banner"
	"github.com/agenthands/rod-cli/daemon"
	"github.com/agenthands/rod-cli/types"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/urfave/cli/v2"
)

// profileListValue is the reserved --profile value that triggers built-in profile
// discovery instead of selecting a profile. `--profile=list` never loads a profile
// named "list" and never launches a daemon.
const profileListValue = "list"

// maybeHandleProfileList intercepts `--profile=list` and prints the built-in
// profile library WITHOUT launching a daemon or loading any profile. It returns
// true when it handled the request (the caller should then return nil). It honors
// --raw (names only, one per line) and --json (a structured list), defaulting to a
// concise human table — consistent with the project's quiet-output ethos.
func maybeHandleProfileList(c *cli.Context) bool {
	if c.String("profile") != profileListValue {
		return false
	}
	names := types.BuiltinProfileNames()

	if c.Bool("json") {
		type profileInfo struct {
			Name                string `json:"name"`
			Platform            string `json:"platform"`
			UserAgent           string `json:"userAgent"`
			Screen              string `json:"screen"`
			HardwareConcurrency int    `json:"hardwareConcurrency"`
			DeviceMemory        int    `json:"deviceMemory"`
		}
		infos := make([]profileInfo, 0, len(names))
		for _, n := range names {
			p, ok, err := types.LoadBuiltinProfile(n)
			if err != nil || !ok {
				// A corrupt embedded build is the only way this fails (the 32-02 gate
				// loads every built-in). Still surface the name so --json discovery
				// agrees with --raw rather than silently hiding it.
				infos = append(infos, profileInfo{Name: n, Screen: "unreadable"})
				continue
			}
			infos = append(infos, profileInfo{
				Name:                n,
				Platform:            p.Platform,
				UserAgent:           p.UserAgent,
				Screen:              fmt.Sprintf("%dx%d", p.Screen.Width, p.Screen.Height),
				HardwareConcurrency: p.HardwareConcurrency,
				DeviceMemory:        p.DeviceMemory,
			})
		}
		out, _ := json.Marshal(map[string]interface{}{"profiles": infos})
		fmt.Println(string(out))
		return true
	}

	if c.Bool("raw") {
		for _, n := range names {
			fmt.Println(n)
		}
		return true
	}

	// Human form: a concise aligned table (name + OS / screen / hardware).
	width := 0
	for _, n := range names {
		if len(n) > width {
			width = len(n)
		}
	}
	for _, n := range names {
		p, ok, err := types.LoadBuiltinProfile(n)
		if err != nil || !ok {
			fmt.Println(n)
			continue
		}
		fmt.Printf("%-*s  %s, %dx%d, %d cores / %dGB\n",
			width, n, p.Platform, p.Screen.Width, p.Screen.Height,
			p.HardwareConcurrency, p.DeviceMemory)
	}
	return true
}

// daemonRunning reports whether the per-session daemon is already up by pinging
// it. A nil error from the ping means the daemon answered.
func daemonRunning(session string) bool {
	_, err := daemon.ClientExecute(session, daemon.Request{Command: "ping"})
	return err == nil
}

// isJSONValue reports whether msg is an already-structured JSON object or array.
// Used by the --json output path to pass a daemon-produced structured result
// (e.g. stealth-check) through verbatim instead of re-wrapping it as a string.
// Plain human-message results never start with { or [, so they stay wrapped.
func isJSONValue(msg string) bool {
	t := strings.TrimSpace(msg)
	if len(t) == 0 || (t[0] != '{' && t[0] != '[') {
		return false
	}
	return json.Valid([]byte(t))
}

func runClientCommand(c *cli.Context, req daemon.Request) error {
	// `--profile=list` is a discovery request, not a session command: print the
	// built-in library and exit WITHOUT spawning a daemon or loading a profile.
	if maybeHandleProfileList(c) {
		return nil
	}

	session := c.String("session")

	// Format generic daemon-spawn flags from global cli args
	flags := []string{}
	if c.String("config") != "" { flags = append(flags, "--config", c.String("config")) }
	if c.String("cdp-endpoint") != "" { flags = append(flags, "--cdp-endpoint", c.String("cdp-endpoint")) }
	if c.Bool("headless") { flags = append(flags, "--headless") }
	if c.Bool("vision") { flags = append(flags, "--vision") }

	// Non-secret stealth flags are forwarded verbatim into the daemon spawn args.
	// This is the persistence linchpin: a stealth flag only "sticks" for the
	// session if it is present at spawn time (EnsureDaemon appends flags into the
	// daemon argv). The proxy URL and profile path are not secrets.
	if c.String("proxy") != "" { flags = append(flags, "--proxy", c.String("proxy")) }
	if c.String("profile") != "" { flags = append(flags, "--profile", c.String("profile")) }
	// The 4 curated fingerprint pins are non-secret (UA is PII-ish but not a
	// credential), so verbatim argv forwarding is correct — they must be present
	// at spawn to "stick" for the session.
	if c.String("user-agent") != "" { flags = append(flags, "--user-agent", c.String("user-agent")) }
	if c.String("locale") != "" { flags = append(flags, "--locale", c.String("locale")) }
	if c.String("timezone") != "" { flags = append(flags, "--timezone", c.String("timezone")) }
	if c.String("platform") != "" { flags = append(flags, "--platform", c.String("platform")) }
	// Phase-27 hardening toggles default ON; forward ONLY when the user explicitly
	// set them so the daemon's *bool stays nil = keep-default-true otherwise.
	if c.IsSet("webrtc-protection") { flags = append(flags, fmt.Sprintf("--webrtc-protection=%t", c.Bool("webrtc-protection"))) }
	if c.IsSet("canvas-noise") { flags = append(flags, fmt.Sprintf("--canvas-noise=%t", c.Bool("canvas-noise"))) }
	// Phase-33 fingerprint-dimension toggles default ON; forward ONLY when explicitly
	// set so the daemon's *bool stays nil = keep-default-true otherwise.
	if c.IsSet("font-spoof") { flags = append(flags, fmt.Sprintf("--font-spoof=%t", c.Bool("font-spoof"))) }
	if c.IsSet("media-devices-spoof") { flags = append(flags, fmt.Sprintf("--media-devices-spoof=%t", c.Bool("media-devices-spoof"))) }
	if c.IsSet("battery-spoof") { flags = append(flags, fmt.Sprintf("--battery-spoof=%t", c.Bool("battery-spoof"))) }
	if c.IsSet("codec-spoof") { flags = append(flags, fmt.Sprintf("--codec-spoof=%t", c.Bool("codec-spoof"))) }
	// Phase-30 CDP-footprint capture toggles default OFF; forward ONLY when the user
	// explicitly set them so the daemon's *bool stays nil = keep-default-off otherwise.
	if c.IsSet("console-capture") { flags = append(flags, fmt.Sprintf("--console-capture=%t", c.Bool("console-capture"))) }
	if c.IsSet("request-capture") { flags = append(flags, fmt.Sprintf("--request-capture=%t", c.Bool("request-capture"))) }
	// Phase-28 humanize tuning flags forward ONLY when explicitly set so the
	// daemon-side pointer stays nil (= keep godoll default) otherwise. Bool flags
	// use the =%t form (so --mouse-tremor=false survives); int/float pass a value.
	if c.IsSet("typing-speed-min") { flags = append(flags, "--typing-speed-min", fmt.Sprint(c.Int("typing-speed-min"))) }
	if c.IsSet("typing-speed-max") { flags = append(flags, "--typing-speed-max", fmt.Sprint(c.Int("typing-speed-max"))) }
	if c.IsSet("typo-rate") { flags = append(flags, "--typo-rate", fmt.Sprint(c.Float64("typo-rate"))) }
	if c.IsSet("mouse-tremor") { flags = append(flags, fmt.Sprintf("--mouse-tremor=%t", c.Bool("mouse-tremor"))) }
	if c.IsSet("mouse-steps") { flags = append(flags, "--mouse-steps", fmt.Sprint(c.Int("mouse-steps"))) }
	if c.IsSet("mouse-speed-min") { flags = append(flags, "--mouse-speed-min", fmt.Sprint(c.Int("mouse-speed-min"))) }
	if c.IsSet("mouse-speed-max") { flags = append(flags, "--mouse-speed-max", fmt.Sprint(c.Int("mouse-speed-max"))) }
	if c.IsSet("mouse-deviation") { flags = append(flags, "--mouse-deviation", fmt.Sprint(c.Float64("mouse-deviation"))) }
	if c.IsSet("scroll-duration") { flags = append(flags, "--scroll-duration", fmt.Sprint(c.Int("scroll-duration"))) }
	if c.IsSet("scroll-physics") { flags = append(flags, fmt.Sprintf("--scroll-physics=%t", c.Bool("scroll-physics"))) }
	// CDP-DEEP-01 proxy flags forward ONLY when explicitly set so the
	// daemon-side pointer stays nil (= keep-default-off).
	if c.IsSet("cdp-proxy") { flags = append(flags, fmt.Sprintf("--cdp-proxy=%t", c.Bool("cdp-proxy"))) }
	if c.IsSet("cdp-jitter-ms") { flags = append(flags, "--cdp-jitter-ms", fmt.Sprint(c.Int("cdp-jitter-ms"))) }
	if c.IsSet("no-cdp-proxy") { flags = append(flags, fmt.Sprintf("--no-cdp-proxy=%t", c.Bool("no-cdp-proxy"))) }

	// proxy-auth is a CREDENTIAL — it must NEVER enter the daemon argv (argv is
	// world-readable via /proc/<pid>/cmdline and `ps`). Pass it out-of-band through
	// the daemon's environment instead; runDaemonServer reads ROD_CLI_PROXY_AUTH.
	var extraEnv []string
	if c.String("proxy-auth") != "" {
		extraEnv = append(extraEnv, "ROD_CLI_PROXY_AUTH="+c.String("proxy-auth"))
	}

	// Stealth config is resolved once at daemon spawn. If the daemon is already
	// running, a stealth flag cannot retroactively apply — warn to STDERR (never
	// stdout, so --raw/piped callers are unaffected) and proceed with the existing
	// config. No silent ignore, no surprise auto-restart. The proxy-auth value is
	// never echoed.
	stealthRequested := c.String("proxy") != "" || c.String("proxy-auth") != "" || c.String("profile") != "" ||
		c.String("user-agent") != "" || c.String("locale") != "" || c.String("timezone") != "" || c.String("platform") != "" ||
		c.IsSet("webrtc-protection") || c.IsSet("canvas-noise") ||
		c.IsSet("font-spoof") || c.IsSet("media-devices-spoof") ||
		c.IsSet("battery-spoof") || c.IsSet("codec-spoof") ||
		c.IsSet("typing-speed-min") || c.IsSet("typing-speed-max") || c.IsSet("typo-rate") ||
		c.IsSet("mouse-tremor") || c.IsSet("mouse-steps") || c.IsSet("mouse-speed-min") ||
		c.IsSet("mouse-speed-max") || c.IsSet("mouse-deviation") ||
		c.IsSet("scroll-duration") || c.IsSet("scroll-physics") ||
		c.IsSet("console-capture") || c.IsSet("request-capture") ||
		c.IsSet("cdp-proxy") || c.IsSet("cdp-jitter-ms") || c.IsSet("no-cdp-proxy")
	if stealthRequested && daemonRunning(session) {
		fmt.Fprintf(os.Stderr, "warning: session %q is already running; stealth flags apply at session spawn — run `close` first to re-apply\n", session)
	}

	err := daemon.EnsureDaemon(session, os.Args[0], flags, extraEnv)
	if err != nil {
		return fmt.Errorf("failed to ensure daemon: %v", err)
	}

	msg, err := daemon.ClientExecute(session, req)
	if err != nil {
		return err
	}

	if c.Bool("json") {
		// If the daemon already produced a structured JSON object/array (e.g.
		// stealth-check's per-signal verdicts), pass it through verbatim so it is
		// not double-wrapped into {"result":"<escaped json string>"}. Plain string
		// results (every other command) are wrapped as before.
		if isJSONValue(msg) {
			fmt.Println(msg)
		} else {
			out, _ := json.Marshal(map[string]string{"result": msg})
			fmt.Println(string(out))
		}
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
			&cli.StringFlag{Name: "proxy", Usage: "route this session through an HTTP or SOCKS5 proxy (scheme in URL, e.g. http://host:port or socks5://host:port)"},
			&cli.StringFlag{Name: "proxy-auth", Usage: "proxy credentials as user:pass (handled via CDP, never URL-embedded)"},
			&cli.StringFlag{Name: "profile", Usage: "stealth profile: a built-in name (see --profile=list), a custom name resolved under ~/.rod-cli/profiles/, or a path to a JSON profile file"},
			&cli.StringFlag{Name: "user-agent", Usage: "pin navigator.userAgent / HTTP UA for this session (the fingerprint derivation anchor)"},
			&cli.StringFlag{Name: "locale", Usage: "pin the BCP-47 locale (e.g. en-US); derived from languages when unset"},
			&cli.StringFlag{Name: "timezone", Usage: "pin the IANA timezone (e.g. America/New_York)"},
			&cli.StringFlag{Name: "platform", Usage: "pin navigator.platform (e.g. Win32, MacIntel, Linux); auto-derived from the UA OS token when unset"},
			&cli.BoolFlag{Name: "webrtc-protection", Usage: "Prevent WebRTC local-IP leaks (default on; --webrtc-protection=false to disable)", Value: true},
			&cli.BoolFlag{Name: "canvas-noise", Usage: "Apply stable-per-session canvas/WebGL/audio noise (default on; --canvas-noise=false to disable)", Value: true},
			// Phase-33 advanced fingerprint-dimension hardening (EVAD-02/03). Each
			// default ON; the injected values are coherent with the profile OS
			// (Windows fonts on a Windows profile, etc.) and stable per session.
			&cli.BoolFlag{Name: "font-spoof", Usage: "Spoof OS-coherent font availability (default on; --font-spoof=false to disable)", Value: true},
			&cli.BoolFlag{Name: "media-devices-spoof", Usage: "Spoof navigator.mediaDevices.enumerateDevices() (default on; --media-devices-spoof=false to disable)", Value: true},
			&cli.BoolFlag{Name: "battery-spoof", Usage: "Spoof navigator.getBattery() (default on; --battery-spoof=false to disable)", Value: true},
			&cli.BoolFlag{Name: "codec-spoof", Usage: "Spoof media canPlayType / codec support (default on; --codec-spoof=false to disable)", Value: true},
			&cli.BoolFlag{Name: "cdp-proxy", Usage: "Enable the in-process CDP WebSocket proxy for traffic logging and normalization (default off)", Value: false},
		&cli.IntFlag{Name: "cdp-jitter-ms", Usage: "Max random CDP command delay in ms (0=off, requires --cdp-proxy)", Value: 0},
		&cli.BoolFlag{Name: "no-cdp-proxy", Usage: "Bypass the CDP proxy even if --cdp-proxy is set (escape hatch)", Value: false},
			// Phase-30 CDP-footprint capture toggles (CDP-01). Default OFF: a plain
			// session enables neither Runtime nor Network. Enable at spawn to let the
			// console / requests commands collect their logs. Resolved once per session.
			&cli.BoolFlag{Name: "console-capture", Usage: "Capture browser console messages for the `console` command (default off; enables Runtime CDP domain)"},
			&cli.BoolFlag{Name: "request-capture", Usage: "Capture network requests for the `requests`/`request` commands (default off; enables Network CDP domain)"},
			// Phase-28 human-behavior tuning (HUMANIZE-01). Each is unset by default
			// (no Value:) so c.IsSet gates forwarding; an unset knob keeps godoll's
			// own default (zero regression). Resolved once at session spawn.
			&cli.IntFlag{Name: "typing-speed-min", Usage: "Min per-keystroke delay in ms; set with --typing-speed-max to tune typing speed (the spread IS the delay jitter)"},
			&cli.IntFlag{Name: "typing-speed-max", Usage: "Max per-keystroke delay in ms; set with --typing-speed-min (wider min/max spread = more delay jitter)"},
			&cli.Float64Flag{Name: "typo-rate", Usage: "Probability 0.0-1.0 of an injected typo per keystroke"},
			&cli.BoolFlag{Name: "mouse-tremor", Usage: "Add microscopic tremor to the mouse path (--mouse-tremor=false to disable)"},
			&cli.IntFlag{Name: "mouse-steps", Usage: "Number of interpolation steps along the mouse path (more = smoother)"},
			&cli.IntFlag{Name: "mouse-speed-min", Usage: "Min mouse speed in px/s; set with --mouse-speed-max"},
			&cli.IntFlag{Name: "mouse-speed-max", Usage: "Max mouse speed in px/s; set with --mouse-speed-min"},
			&cli.Float64Flag{Name: "mouse-deviation", Usage: "Mouse-path randomness factor 0.0-1.0"},
			&cli.IntFlag{Name: "scroll-duration", Usage: "Base scroll animation duration in ms"},
			&cli.BoolFlag{Name: "scroll-physics", Usage: "Use physics-based (eased) scrolling (godoll default; cannot be disabled via flag in v1.6)"},
		},
		Commands: []*cli.Command{
			{
				Name:  "install",
				Usage: "Install the Chromium browser required by rod-cli",
				Action: func(c *cli.Context) error {
					if !c.Bool("raw") && !c.Bool("json") {
						fmt.Println("Downloading Chromium (this may take a minute)...")
					}
					browserPath := launcher.NewBrowser().MustGet()
					if c.Bool("json") {
						out, _ := json.Marshal(map[string]string{"path": browserPath, "status": "installed"})
						fmt.Println(string(out))
					} else {
						fmt.Printf("Chromium installed successfully at: %s\n", browserPath)
					}
					return nil
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
				Name:  "sessions",
				Usage: "List all active background sessions",
				Action: func(c *cli.Context) error {
					sessions, err := daemon.ListSessions()
					if err != nil {
						return err
					}
					if c.Bool("json") {
						out, _ := json.Marshal(map[string]interface{}{"sessions": sessions})
						fmt.Println(string(out))
					} else {
						if len(sessions) == 0 {
							fmt.Println("No active sessions")
						} else {
							fmt.Println("Active sessions:")
							for _, s := range sessions {
								fmt.Println("- " + s)
							}
						}
					}
					return nil
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
				Name:  "check",
				Usage: "Check a checkbox or radio button by its ref",
				Action: func(c *cli.Context) error {
					ref := c.Args().First()
					if ref == "" {
						return fmt.Errorf("element ref is required")
					}
					return runClientCommand(c, daemon.Request{Command: "check", Args: map[string]string{"ref": ref}})
				},
			},
			{
				Name:  "uncheck",
				Usage: "Uncheck a checkbox or radio button by its ref",
				Action: func(c *cli.Context) error {
					ref := c.Args().First()
					if ref == "" {
						return fmt.Errorf("element ref is required")
					}
					return runClientCommand(c, daemon.Request{Command: "uncheck", Args: map[string]string{"ref": ref}})
				},
			},
			{
				Name:  "upload",
				Usage: "Upload files to an element",
				Action: func(c *cli.Context) error {
					ref := c.Args().First()
					files := strings.Join(c.Args().Slice()[1:], ",")
					if ref == "" || files == "" {
						return fmt.Errorf("element ref and at least one file are required")
					}
					return runClientCommand(c, daemon.Request{Command: "upload", Args: map[string]string{"ref": ref, "files": files}})
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
					ref := c.Args().Get(1)
					if script == "" {
						return fmt.Errorf("script is required")
					}
					return runClientCommand(c, daemon.Request{Command: "eval", Args: map[string]string{"script": script, "ref": ref}})
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
				Name:  "stealth-check",
				Usage: "Run per-signal stealth verdicts against the current page (or navigate to URL first)",
				Action: func(c *cli.Context) error {
					url := c.Args().First()
					return runClientCommand(c, daemon.Request{Command: "stealth-check", Args: map[string]string{
						"url":  url,
						"raw":  fmt.Sprint(c.Bool("raw")),
						"json": fmt.Sprint(c.Bool("json")),
					}})
				},
			},
			{
				Name:  "cdp-traffic",
				Usage: "Print logged CDP protocol traffic from the proxy (requires --cdp-proxy). WARNING: output may contain sensitive CDP payload data (URLs, cookies, page content).",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "cdp-traffic", Args: map[string]string{
						"json": fmt.Sprint(c.Bool("json")),
					}})
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
			{
				Name:  "press",
				Usage: "Press a keyboard key",
				Action: func(c *cli.Context) error {
					key := c.Args().First()
					if key == "" {
						return fmt.Errorf("key is required")
					}
					return runClientCommand(c, daemon.Request{Command: "press", Args: map[string]string{"key": key}})
				},
			},
			{
				Name:  "mousemove",
				Usage: "Move mouse to x y coordinates",
				Action: func(c *cli.Context) error {
					x := c.Args().Get(0)
					y := c.Args().Get(1)
					if x == "" || y == "" {
						return fmt.Errorf("x and y are required")
					}
					return runClientCommand(c, daemon.Request{Command: "mousemove", Args: map[string]string{"x": x, "y": y}})
				},
			},
			{
				Name:  "mousedown",
				Usage: "Trigger mouse down",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "mousedown", Args: map[string]string{"button": c.Args().First()}})
				},
			},
			{
				Name:  "mouseup",
				Usage: "Trigger mouse up",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "mouseup", Args: map[string]string{"button": c.Args().First()}})
				},
			},
			{
				Name:  "mousewheel",
				Usage: "Scroll mouse wheel",
				Action: func(c *cli.Context) error {
					dx := c.Args().Get(0)
					dy := c.Args().Get(1)
					if dx == "" || dy == "" {
						return fmt.Errorf("dx and dy are required")
					}
					return runClientCommand(c, daemon.Request{Command: "mousewheel", Args: map[string]string{"dx": dx, "dy": dy}})
				},
			},
			{
				Name:  "resize",
				Usage: "Resize the browser window",
				Action: func(c *cli.Context) error {
					width := c.Args().Get(0)
					height := c.Args().Get(1)
					if width == "" || height == "" {
						return fmt.Errorf("width and height are required")
					}
					return runClientCommand(c, daemon.Request{Command: "resize", Args: map[string]string{"width": width, "height": height}})
				},
			},
			{
				Name:  "tab-list",
				Usage: "List all tabs",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "tab-list"})
				},
			},
			{
				Name:  "tab-new",
				Usage: "Create a new tab",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "tab-new", Args: map[string]string{"url": c.Args().First()}})
				},
			},
			{
				Name:  "plugin",
				Usage: "Manage external plugins",
				Subcommands: []*cli.Command{
					{
						Name:  "load",
						Usage: "Load a plugin script",
						Action: func(c *cli.Context) error {
							path := c.Args().First()
							if path == "" {
								return fmt.Errorf("plugin path is required")
							}
							return runClientCommand(c, daemon.Request{Command: "plugin-load", Args: map[string]string{"path": path}})
						},
					},
					{
						Name:  "list",
						Usage: "List loaded plugins",
						Action: func(c *cli.Context) error {
							return runClientCommand(c, daemon.Request{Command: "plugin-list"})
						},
					},
					{
						Name:  "run",
						Usage: "Trigger a loaded plugin",
						Action: func(c *cli.Context) error {
							name := c.Args().First()
							return runClientCommand(c, daemon.Request{Command: "plugin-run", Args: map[string]string{"name": name}})
						},
					},
				},
			},
			{
				Name:  "tab-close",
				Usage: "Close a browser tab",
				Action: func(c *cli.Context) error {
					index := c.Args().First()
					if index == "" {
						return fmt.Errorf("tab index is required")
					}
					return runClientCommand(c, daemon.Request{Command: "tab-close", Args: map[string]string{"index": index}})
				},
			},
			{
				Name:  "tab-select",
				Usage: "Select a browser tab",
				Action: func(c *cli.Context) error {
					index := c.Args().First()
					if index == "" {
						return fmt.Errorf("tab index is required")
					}
					return runClientCommand(c, daemon.Request{Command: "tab-select", Args: map[string]string{"index": index}})
				},
			},
			{
				Name:  "dialog-accept",
				Usage: "Accept next dialog",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "prompt", Usage: "Text to enter into prompt"},
				},
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "dialog-accept", Args: map[string]string{"promptText": c.String("prompt")}})
				},
			},
			{
				Name:  "dialog-dismiss",
				Usage: "Dismiss next dialog",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "dialog-dismiss"})
				},
			},
			{
				Name:  "cookie-get",
				Usage: "Get cookies",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "cookie-get"})
				},
			},
			{
				Name:  "cookie-set",
				Usage: "Set a cookie",
				Action: func(c *cli.Context) error {
					name := c.Args().Get(0)
					value := c.Args().Get(1)
					if name == "" || value == "" {
						return fmt.Errorf("cookie name and value are required")
					}
					return runClientCommand(c, daemon.Request{Command: "cookie-set", Args: map[string]string{"name": name, "value": value}})
				},
			},
			{
				Name:  "cookie-delete",
				Usage: "Delete a cookie",
				Action: func(c *cli.Context) error {
					name := c.Args().First()
					if name == "" {
						return fmt.Errorf("cookie name is required")
					}
					return runClientCommand(c, daemon.Request{Command: "cookie-delete", Args: map[string]string{"name": name}})
				},
			},
			{
				Name:  "cookie-clear",
				Usage: "Clear cookies",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "cookie-clear"})
				},
			},
			{
				Name:  "localstorage-get",
				Usage: "Get localStorage item or all",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "localstorage-get", Args: map[string]string{"key": c.Args().First()}})
				},
			},
			{
				Name:  "localstorage-set",
				Usage: "Set localStorage item",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "localstorage-set", Args: map[string]string{"key": c.Args().Get(0), "value": c.Args().Get(1)}})
				},
			},
			{
				Name:  "localstorage-delete",
				Usage: "Delete localStorage entry",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "localstorage-delete", Args: map[string]string{"key": c.Args().First()}})
				},
			},
			{
				Name:  "localstorage-clear",
				Usage: "Clear localStorage",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "localstorage-clear"})
				},
			},
			{
				Name:  "sessionstorage-get",
				Usage: "Get sessionStorage item or all",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "sessionstorage-get", Args: map[string]string{"key": c.Args().First()}})
				},
			},
			{
				Name:  "sessionstorage-set",
				Usage: "Set sessionStorage item",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "sessionstorage-set", Args: map[string]string{"key": c.Args().Get(0), "value": c.Args().Get(1)}})
				},
			},
			{
				Name:  "sessionstorage-delete",
				Usage: "Delete sessionStorage entry",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "sessionstorage-delete", Args: map[string]string{"key": c.Args().First()}})
				},
			},
			{
				Name:  "sessionstorage-clear",
				Usage: "Clear sessionStorage",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "sessionstorage-clear"})
				},
			},
			{
				Name:  "route",
				Usage: "Mock network requests",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "body", Usage: "Mock response body"},
				},
				Action: func(c *cli.Context) error {
					pattern := c.Args().First()
					if pattern == "" {
						return fmt.Errorf("route pattern is required")
					}
					return runClientCommand(c, daemon.Request{Command: "route", Args: map[string]string{"pattern": pattern, "body": c.String("body")}})
				},
			},
			{
				Name:  "route-list",
				Usage: "List active routes",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "route-list"})
				},
			},
			{
				Name:  "unroute",
				Usage: "Remove a route",
				Action: func(c *cli.Context) error {
					pattern := c.Args().First()
					return runClientCommand(c, daemon.Request{Command: "unroute", Args: map[string]string{"pattern": pattern}})
				},
			},
			{
				Name:  "console",
				Usage: "List console messages (requires the session spawned with --console-capture; off by default)",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "console"})
				},
			},
			{
				Name:  "requests",
				Usage: "List network requests (requires the session spawned with --request-capture; off by default)",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "requests"})
				},
			},
			{
				Name:  "request",
				Usage: "Show details for a specific request (requires the session spawned with --request-capture; off by default)",
				Action: func(c *cli.Context) error {
					index := c.Args().First()
					if index == "" {
						return fmt.Errorf("request index is required")
					}
					return runClientCommand(c, daemon.Request{Command: "request", Args: map[string]string{"index": index}})
				},
			},
			{
				Name:  "drag",
				Usage: "Drag an element to another element",
				Action: func(c *cli.Context) error {
					start := c.Args().Get(0)
					end := c.Args().Get(1)
					if start == "" || end == "" {
						return fmt.Errorf("start and end refs are required")
					}
					return runClientCommand(c, daemon.Request{Command: "drag", Args: map[string]string{"start": start, "end": end}})
				},
			},
			{
				Name:  "drop",
				Usage: "Drop a file onto an element",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "path", Usage: "File path to drop"},
				},
				Action: func(c *cli.Context) error {
					ref := c.Args().First()
					if ref == "" {
						return fmt.Errorf("element ref is required")
					}
					path := c.String("path")
					if path == "" {
						return fmt.Errorf("--path is required")
					}
					return runClientCommand(c, daemon.Request{Command: "drop", Args: map[string]string{"ref": ref, "path": path}})
				},
			},
			{
				Name:  "state-save",
				Usage: "Save browser state to a file",
				Action: func(c *cli.Context) error {
					path := c.Args().First()
					if path == "" {
						return fmt.Errorf("file path is required")
					}
					return runClientCommand(c, daemon.Request{Command: "state-save", Args: map[string]string{"path": path}})
				},
			},
			{
				Name:  "state-load",
				Usage: "Load browser state from a file",
				Action: func(c *cli.Context) error {
					path := c.Args().First()
					if path == "" {
						return fmt.Errorf("file path is required")
					}
					return runClientCommand(c, daemon.Request{Command: "state-load", Args: map[string]string{"path": path}})
				},
			},
			{
				Name:  "highlight",
				Usage: "Highlight an element by its ref",
				Action: func(c *cli.Context) error {
					ref := c.Args().First()
					if ref == "" {
						return fmt.Errorf("element ref is required")
					}
					return runClientCommand(c, daemon.Request{Command: "highlight", Args: map[string]string{"ref": ref}})
				},
			},
			{
				Name:  "highlight-clear",
				Usage: "Clear all highlights",
				Action: func(c *cli.Context) error {
					return runClientCommand(c, daemon.Request{Command: "highlight-clear"})
				},
			},

			{
				Name:  "show",
				Usage: "Show the browser or launch interactive annotation",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "annotate", Usage: "Launch interactive annotation UI"},
				},
				Action: func(c *cli.Context) error {
					annotate := fmt.Sprint(c.Bool("annotate"))
					return runClientCommand(c, daemon.Request{Command: "show", Args: map[string]string{"annotate": annotate}})
				},
			},
		},
		Action: func(c *cli.Context) error {
			// `rod-cli --profile=list` (no subcommand) lists the built-in profiles.
			if maybeHandleProfileList(c) {
				return nil
			}
			if !c.Bool("no-banner") {
				fmt.Println(banner.ShowBanner())
			}
			return cli.ShowAppHelp(c)
		},
	}
}
