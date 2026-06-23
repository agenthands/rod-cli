package types

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-rod/rod"
	"github.com/agenthands/godoll/browser"
	"github.com/agenthands/godoll/network"
	"github.com/agenthands/godoll/stealth"
	rodfingerprint "github.com/agenthands/godoll/fingerprint"
	"github.com/agenthands/rod-cli/internal/plugin"
	"github.com/agenthands/rod-cli/types/js"
	"github.com/agenthands/rod-cli/utils"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
)

// seams for testing: indirections through which production calls are routed so
// that otherwise-unreachable error branches can be exercised. Defaults are the
// real functions; tests swap them and restore via defer. Zero behavior change.
var (
	launcherLookPath = launcher.LookPath
	osRemoveAll      = os.RemoveAll
)

func launchBrowser(ctx context.Context, cfg Config) (*rod.Browser, error) {

	if cfg.CDPEndpoint != "" {
		return controlBrowser(ctx, cfg.CDPEndpoint)
	}

	if cfg.BrowserTempDir == "" {
		cfg.BrowserTempDir = DefaultBrowserTempDir
	}

	// browser must own a unique temp dir
	cfg.BrowserTempDir = fmt.Sprintf("%s/%s", cfg.BrowserTempDir, utils.RandomString(10))
	// Create basic launcher manually so we can set Headless, Proxy, UserDataDir
	browserLauncher := launcher.New().
		Context(ctx).
		Headless(cfg.Headless).
		NoSandbox(cfg.NoSandbox).
		Set("no-gpu").
		Set("--no-first-run").
		Set("ignore-certificate-errors", "true").
		Set("disable-xss-auditor", "true").
		Set("disable-popup-blocking").
		Set("mute-audio", "true").
		Set("use-mock-keychain").
		Set("--remote-allow-origins", "*").
		Set("--disable-dev-shm-usage").
		Set("--disable-features", "HttpsUpgrades").
		UserDataDir(cfg.BrowserTempDir)

	if cfg.BrowserBinPath != "" {
		browserLauncher.Bin(cfg.BrowserBinPath)
	} else {
		if browserPath, has := launcherLookPath(); has {
			browserLauncher.Bin(browserPath)
		} else {
			return nil, errors.New("the machine does not have Chrome installed. Please run 'rod-cli install' to fetch it, or set the browser executable path.")
		}
	}

	if cfg.Proxy != "" {
		browserLauncher.Proxy(cfg.Proxy)
	}

	// Create godoll options and apply stealth preset
	opts := browser.NewBrowserOptions().
		WithLauncher(browserLauncher).
		SetBrowserPreferences(browser.NewBrowserOptions().StealthPreset())

	browserInstance, err := browser.NewBrowserE(ctx, opts)
	if err != nil {
		return nil, errors.Wrap(err, "launch local browser failed via godoll")
	}

	return browserInstance, nil
}

func controlBrowser(ctx context.Context, controlURL string) (*rod.Browser, error) {
	browserInstance, err := browser.ConnectToRemoteBrowserWithContext(ctx, controlURL)
	if err != nil {
		return nil, errors.Wrap(err, "Error connecting to remote browser via godoll")
	}
	return browserInstance, nil
}

// Mode is the model type, indicates the model type of the tool
type Mode string

const (
	// Vision mode indicates the vision ll model,will load the vision tools
	Vision Mode = "vision"

	// Text mode indicates the no vision ll model,will load the text tools
	Text Mode = "text"
)

type Context struct {
	stdContext context.Context
	config     Config
	browser    *rod.Browser
	page       *rod.Page
	stateLock  sync.Mutex
	isInitial  atomic.Bool
	snapshot    *Snapshot
	mode        Mode
	consoleLogs []string
	requests    []string
	interceptor *network.Interceptor
	routes      map[string]string
	fingerprint *rodfingerprint.Fingerprint
	pluginEngine *plugin.PluginEngine
	loadedPlugins []string
}

func NewContext(ctx context.Context, cfg Config) *Context {
	return &Context{
		stdContext: ctx,
		config:     cfg,
		mode:       cfg.Mode,
		routes:     make(map[string]string),
	}
}

func (ctx *Context) EnsurePage() (*rod.Page, error) {
	if err := ctx.initial(); err != nil {
		return nil, err
	}
	return ctx.page, nil
}

func (ctx *Context) ControlledPage() (*rod.Page, error) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	if ctx.page == nil {
		return nil, errors.New("No tab to used, call rod_navigate first")
	}
	return ctx.page, nil
}

func (ctx *Context) initial() error {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()

	var err error
	if ctx.browser == nil {
		ctx.browser, err = launchBrowser(ctx.stdContext, ctx.config)
		if err != nil {
			return err
		}
		ctx.page, err = ctx.createPage()
		if err != nil {
			return err
		}
		return nil
	}
	if ctx.page == nil {
		ctx.page, err = ctx.createPage()
		if err != nil {
			return err
		}
	}

	return err

}

func (ctx *Context) CurrentMode() Mode {
	return ctx.mode
}

func (ctx *Context) ClosePage() error {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	return ctx.closePage()
}

func (ctx *Context) GetBrowser() *rod.Browser {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	return ctx.browser
}

func (ctx *Context) SetPage(p *rod.Page) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	ctx.page = p
}

func (ctx *Context) GetPluginEngine() *plugin.PluginEngine {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	if ctx.pluginEngine == nil {
		ctx.pluginEngine = plugin.NewPluginEngine()
		ctx.pluginEngine.Init()
	}
	return ctx.pluginEngine
}

func (ctx *Context) GetLoadedPlugins() []string {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	return ctx.loadedPlugins
}

func (ctx *Context) AddLoadedPlugin(path string) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	ctx.loadedPlugins = append(ctx.loadedPlugins, path)
}

func (ctx *Context) BuildSnapshot() (string, error) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	if ctx.page == nil {
		return "", errors.New("No tab to capture snapshot, call rod_navigate first")
	}
	snapshot, err := BuildSnapshot(ctx.page)
	if err != nil {
		return "", err
	}
	ctx.snapshot = snapshot
	return snapshot.String(), nil
}

func (ctx *Context) LatestSnapshot() (*Snapshot, error) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	if ctx.snapshot == nil {
		return nil, errors.New("No snapshot to used, call rod_snapshot first")
	}
	return ctx.snapshot, nil

}

func (ctx *Context) CloseBrowser() error {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	return ctx.closeBrowser()

}

func (ctx *Context) closePage() error {
	if ctx.page == nil {
		return nil
	}
	err := ctx.page.Close()
	if err != nil {
		return errors.Wrap(err, "close page failed")
	}
	ctx.page = nil
	return err
}
func (ctx *Context) closeBrowser() error {
	err := ctx.closePage()
	if err != nil {
		return err
	}

	if ctx.browser == nil {
		return nil
	}

	err = ctx.browser.Close()
	if err != nil {
		return errors.Wrap(err, "close browser failed")
	}
	ctx.browser = nil
	return nil
}

func (ctx *Context) createPage(urls ...string) (*rod.Page, error) {
	page, err := ctx.browser.Page(proto.TargetCreateTarget{URL: strings.Join(urls, "/")})
	if page != nil {
		// Apply stealth evasion
		em := stealth.NewEvasionManager(page)
		fg := rodfingerprint.NewFingerprintGenerator(rodfingerprint.FPWithBrowserNames("chrome"))
		fp, err := fg.Generate()
		if err == nil && fp != nil {
			ctx.fingerprint = fp
			em.SetFingerprint(fp)
		}
		_ = em.Apply()

		page.EvalOnNewDocument(js.InjectedSnapShot)
		
		// Setup logging
		go page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
			ctx.stateLock.Lock()
			defer ctx.stateLock.Unlock()
			var args []string
			for _, arg := range e.Args {
				if !arg.Value.Nil() {
					args = append(args, arg.Value.String())
				}
			}
			ctx.consoleLogs = append(ctx.consoleLogs, fmt.Sprintf("[%s] %s", e.Type, strings.Join(args, " ")))
		})()

		go page.EachEvent(func(e *proto.NetworkRequestWillBeSent) {
			ctx.stateLock.Lock()
			defer ctx.stateLock.Unlock()
			ctx.requests = append(ctx.requests, fmt.Sprintf("%s %s", e.Request.Method, e.Request.URL))
		})()
		
		// Setup godoll/network interceptor
		ctx.interceptor = network.NewInterceptor(page)
		ctx.updateInterceptorRules()
		
		if err := ctx.interceptor.Enable(); err != nil {
			return nil, errors.Wrap(err, "failed to enable network interceptor")
		}
	}
	if err != nil {
		return nil, errors.Wrap(err, "create page failed")
	}
	return page, nil
}

// Accessors
func (ctx *Context) GetConsoleLogs() []string {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	return ctx.consoleLogs
}

func (ctx *Context) GetRequests() []string {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	return ctx.requests
}

func (ctx *Context) updateInterceptorRules() {
	if ctx.interceptor == nil {
		return
	}
	
	ctx.interceptor.ClearRules()
	
	// Add mock routes first
	for pattern, body := range ctx.routes {
		ctx.interceptor.AddRule(network.InterceptRule{
			URLPattern: pattern,
			Action:     network.Mock,
			MockResponse: &network.MockResponse{
				StatusCode: 200,
				Body:       body,
				Headers: map[string]string{
					"Content-Type": "text/plain",
				},
			},
		})
	}
	
	// Add evasion headers catch-all rule
	var prof *stealth.Profile
	fp := ctx.fingerprint
	if fp != nil {
		p := stealth.FromFingerprint(fp)
		prof = &p
	} else {
		def := stealth.DefaultProfile()
		prof = &def
	}
	
	headers := map[string]string{
		"User-Agent": prof.UserAgent,
		"Accept-Language": prof.AcceptLanguage,
		"X-Requested-With": "", // This acts as a deletion since it overrides with empty
	}
	
	if prof.SpoofClientHints {
		var chPlatform string
		switch prof.Platform {
		case "Win32":
			chPlatform = "\"Windows\""
		case "MacIntel":
			chPlatform = "\"macOS\""
		default:
			chPlatform = fmt.Sprintf("\"%s\"", prof.Platform)
		}
		headers["Sec-Ch-Ua"] = "\"Not A(Brand\";v=\"99\", \"Google Chrome\";v=\"121\", \"Chromium\";v=\"121\""
		headers["Sec-Ch-Ua-Mobile"] = "?0"
		headers["Sec-Ch-Ua-Platform"] = chPlatform
	}
	
	ctx.interceptor.AddRule(network.InterceptRule{
		URLPattern: "*",
		Action:     network.Continue,
		ModifiedHeaders: headers,
	})
}

func (ctx *Context) AddRoute(pattern, body string) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	ctx.routes[pattern] = body
	ctx.updateInterceptorRules()
}

func (ctx *Context) RemoveRoute(pattern string) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	delete(ctx.routes, pattern)
	ctx.updateInterceptorRules()
}

func (ctx *Context) GetRoutes() map[string]string {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	res := make(map[string]string)
	for k, v := range ctx.routes {
		res[k] = v
	}
	return res
}

// Close the browser
// PS: This method only used because of server exit
func (ctx *Context) Close() error {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	ctx.closeBrowser()

	// remove browser temp dir
	if ctx.config.BrowserTempDir != "" && ctx.config.CDPEndpoint == "" {
		err := osRemoveAll(ctx.config.BrowserTempDir)
		if err != nil {
			return errors.Wrap(err, "remove browser temp dir failed")
		}
	}
	return nil
}
