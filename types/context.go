package types

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
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

// parseProxyConfig maps a proxy URL (with scheme) and an optional "user:pass"
// auth string onto a godoll browser.ProxyConfig.
//
//   - An empty proxyURL means "no proxy": it returns (nil, nil) and the caller
//     skips all proxy wiring (proxyAuth alone never synthesizes a proxy).
//   - The URL scheme becomes Protocol: "http"/"https" → "http", "socks5",
//     "socks4". host:port becomes Address.
//   - URL-embedded credentials (http://user:pass@host) are STRIPPED — they must
//     never reach Chrome's --proxy-server (Chrome removed support and they leak;
//     SOCKS5 auth is unsupported there). Auth flows exclusively through CDP via
//     ProxyConfig.SetupBrowserAuth. The stripped Address stays credential-free.
//   - proxyAuth is split on the FIRST colon only (the password may itself contain
//     colons) into Username/Password.
//
// It is pure parsing: it performs NO logging, so neither the proxy URL nor the
// credentials are ever emitted (T-25-05/T-25-08).
func parseProxyConfig(proxyURL, proxyAuth string) (*browser.ProxyConfig, error) {
	if proxyURL == "" {
		return nil, nil
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, errors.Wrapf(err, "parse proxy url")
	}
	if u.Scheme == "" {
		return nil, errors.Errorf("proxy url %q is missing a scheme (expected http://, socks5://, or socks4://)", proxyURL)
	}
	if u.Host == "" {
		return nil, errors.Errorf("proxy url %q is missing a host:port", proxyURL)
	}
	// Reject URL-embedded credentials loudly rather than silently dropping them
	// (which would surface as a confusing unauthenticated 407). Credentials must
	// be supplied via --proxy-auth so they are handled via CDP and never reach
	// Chrome's --proxy-server.
	if u.User != nil {
		return nil, errors.Errorf("embedded proxy credentials in the URL are not supported; pass them via --proxy-auth")
	}

	var protocol string
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		protocol = "http"
	case "socks5":
		protocol = "socks5"
	case "socks4":
		protocol = "socks4"
	default:
		return nil, errors.Errorf("unsupported proxy scheme %q (expected http, https, socks5, or socks4)", u.Scheme)
	}

	// u.Host is host:port WITHOUT userinfo — any embedded user:pass@ in the URL
	// is parked in u.User and deliberately dropped here so it can never reach
	// LauncherURL()/--proxy-server.
	cfg := &browser.ProxyConfig{
		Protocol: protocol,
		Address:  u.Host,
	}

	if proxyAuth != "" {
		user, pass, found := strings.Cut(proxyAuth, ":")
		if !found {
			return nil, errors.Errorf("proxy auth must be in user:pass form")
		}
		cfg.Username = user
		cfg.Password = pass
	}

	return cfg, nil
}

// launchBrowser returns the browser plus a proxyCleanup func. proxyCleanup is
// non-nil only when an authenticated proxy spun up a local godoll relay; the
// caller MUST invoke it on session close to stop that relay (T-25-07). It is nil
// for the no-proxy and no-auth-proxy paths.
func launchBrowser(ctx context.Context, cfg Config) (*rod.Browser, func(), error) {

	if cfg.CDPEndpoint != "" {
		b, err := controlBrowser(ctx, cfg.CDPEndpoint)
		return b, nil, err
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
			return nil, nil, errors.New("the machine does not have Chrome installed. Please run 'rod-cli install' to fetch it, or set the browser executable path.")
		}
	}

	// Per-session proxy via godoll's proxy API (replaces the bare
	// launcher.Proxy). cfg.Stealth.Proxy is the authoritative source. For the
	// no-auth case ApplyToLauncher just sets --proxy-server; for the auth case it
	// starts a local CONNECT relay (handling the SOCKS5/HTTP-auth gap) and returns
	// a cleanup that stops it. Credentials never reach --proxy-server.
	proxyCfg, err := parseProxyConfig(cfg.Stealth.Proxy, cfg.Stealth.ProxyAuth)
	if err != nil {
		return nil, nil, errors.Wrap(err, "invalid proxy configuration")
	}
	var proxyCleanup func()
	if proxyCfg != nil {
		proxyCleanup, err = proxyCfg.ApplyToLauncher(browserLauncher)
		if err != nil {
			return nil, nil, errors.Wrap(err, "apply proxy to launcher")
		}
	}

	// Create godoll options and apply stealth preset
	opts := browser.NewBrowserOptions().
		WithLauncher(browserLauncher).
		SetBrowserPreferences(browser.NewBrowserOptions().StealthPreset())

	browserInstance, err := browser.NewBrowserE(ctx, opts)
	if err != nil {
		if proxyCleanup != nil {
			proxyCleanup() // don't leak the relay if the browser fails to start
		}
		return nil, nil, errors.Wrap(err, "launch local browser failed via godoll")
	}

	// Register the persistent CDP auth handler BEFORE any Page()/Navigate() so an
	// authenticated proxy is answered programmatically — no 407 / hanging native
	// auth dialog (T-25-06). SetupBrowserAuth is a no-op unless HasAuth().
	if proxyCfg != nil && proxyCfg.HasAuth() {
		proxyCfg.SetupBrowserAuth(browserInstance)
	}

	return browserInstance, proxyCleanup, nil
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
	// profile is the active stealth.Profile that drives BOTH godoll's evasion JS
	// injection and the rod-cli network interceptor. It is built in createPage by
	// overlaying the config-pinned cfg.Stealth identity onto stealth.DefaultProfile,
	// making the resolved cfg.Stealth the SINGLE source of truth for the live page
	// (FINGERPRINT-01 wiring). Nil until the first page is created.
	profile *stealth.Profile
	pluginEngine *plugin.PluginEngine
	loadedPlugins []string
	// proxyCleanup stops the per-session authenticated-proxy relay (godoll
	// StartProxyRelay). Non-nil only when an authenticated proxy is in use; it is
	// invoked exactly once on browser close and nil'd to guard against double-call
	// (T-25-07).
	proxyCleanup func()
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
		var proxyCleanup func()
		ctx.browser, proxyCleanup, err = launchBrowser(ctx.stdContext, ctx.config)
		if err != nil {
			return err
		}
		ctx.proxyCleanup = proxyCleanup
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
	// Stop the per-session proxy relay (if any) regardless of page/browser state,
	// so the local listener is never leaked. Guard against double-call by nil'ing.
	if ctx.proxyCleanup != nil {
		ctx.proxyCleanup()
		ctx.proxyCleanup = nil
	}

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

// defaultChromeMajor is the Sec-Ch-Ua brand version used when the active profile
// UA carries no parseable Chrome token. It mirrors the Chrome major of
// stealth.DefaultProfile()'s UA, preserving the prior hardcoded behavior for
// empty/garbage UAs (matches godoll's defaultChromeMajor in stealth/evasion.go).
const defaultChromeMajor = "121"

// profileFromStealth builds the active stealth.Profile by overlaying every
// non-zero identity field of the resolved cfg.Stealth onto stealth.DefaultProfile.
// The resolved cfg.Stealth (produced by ResolveStealth's UA-anchored derive +
// coherence validation) is the single source of truth for the live page: this
// profile drives godoll's evasion JS injection AND the rod-cli interceptor.
// With an empty cfg.Stealth identity the result equals DefaultProfile(), so the
// no-pin path is byte-for-byte the prior default behavior (no regression).
func profileFromStealth(s StealthConfig) stealth.Profile {
	p := stealth.DefaultProfile()
	// rod-cli emits coherent, UA-derived Client-Hints by default so Sec-Ch-Ua,
	// navigator.userAgentData, and the UA all tell one version story (FINGERPRINT-02/03).
	// DefaultProfile() ships SpoofClientHints=false; without this the default identity
	// would send no Sec-Ch-Ua and an empty userAgentData.brands — itself a detection tell,
	// and a regression vs the pre-v1.6 FromFingerprint path which had it on. The overlay
	// below still lets an explicit profile keep it on; there is intentionally no way to
	// ship an incoherent CH-off identity from the resolved active profile.
	p.SpoofClientHints = true
	if s.UserAgent != "" {
		p.UserAgent = s.UserAgent
	}
	if s.Platform != "" {
		p.Platform = s.Platform
	}
	if s.Locale != "" {
		p.Locale = s.Locale
	}
	if s.Timezone != "" {
		p.Timezone = s.Timezone
	}
	if s.AcceptLanguage != "" {
		p.AcceptLanguage = s.AcceptLanguage
	}
	if len(s.Languages) > 0 {
		p.Languages = s.Languages
	}
	if s.Screen.Width != 0 {
		p.Screen.Width = s.Screen.Width
	}
	if s.Screen.Height != 0 {
		p.Screen.Height = s.Screen.Height
	}
	if s.Screen.DeviceScaleFactor != 0 {
		p.Screen.DeviceScaleFactor = s.Screen.DeviceScaleFactor
	}
	if s.HardwareConcurrency != 0 {
		p.HardwareConcurrency = s.HardwareConcurrency
	}
	if s.DeviceMemory != 0 {
		p.DeviceMemory = s.DeviceMemory
	}
	if s.Vendor != "" {
		p.Vendor = s.Vendor
	}
	if s.SpoofClientHints {
		p.SpoofClientHints = true
	}
	return p
}

func (ctx *Context) createPage(urls ...string) (*rod.Page, error) {
	page, err := ctx.browser.Page(proto.TargetCreateTarget{URL: strings.Join(urls, "/")})
	if page != nil {
		// Apply stealth evasion
		em := stealth.NewEvasionManager(page)
		// Generate a random fingerprint ONLY for the dimensions godoll still needs
		// from a Fingerprint (WebGL VideoCard etc.) — it is NOT the identity source.
		fg := rodfingerprint.NewFingerprintGenerator(rodfingerprint.FPWithBrowserNames("chrome"))
		fp, fpErr := fg.Generate()
		if fpErr != nil {
			fmt.Fprintf(os.Stderr, "warning: fingerprint generation failed: %v\n", fpErr)
		} else if fp != nil {
			ctx.fingerprint = fp
		}
		// Build the active stealth.Profile from the resolved cfg.Stealth: the
		// config-pinned identity is the SINGLE source of truth that drives both
		// godoll's evasion JS and rod-cli's interceptor (FINGERPRINT-01 wiring).
		// With an empty cfg.Stealth identity the overlay equals DefaultProfile(),
		// so the no-pin path is unchanged from the prior default.
		prof := profileFromStealth(ctx.config.Stealth)
		ctx.profile = &prof
		em.SetProfile(prof)
		if err := em.Apply(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: evasion Apply failed: %v\n", err)
		}

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
	
	// Add evasion headers catch-all rule.
	// Prefer the pinned active profile (built from cfg.Stealth in createPage) so
	// the interceptor headers tell the SAME identity story as godoll's JS injection.
	// Fall back to the random-fingerprint-derived profile, then DefaultProfile.
	var prof *stealth.Profile
	if ctx.profile != nil {
		prof = ctx.profile
	} else if fp := ctx.fingerprint; fp != nil {
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
		// Derive the brand version from the active profile UA's `Chrome/<major>`
		// token (parsed by parseChromeMajor's `Chrome/(\d+)` regexp) so the
		// Sec-Ch-Ua header agrees with the UA and godoll's userAgentData brand
		// version (the FINGERPRINT-02 triple-agreement). Reuse parseChromeMajor
		// (same types package, from Plan 01) — ONE derivation path in rod-cli.
		// Fall back to the prior default major when the UA has no Chrome token so
		// behavior is preserved for empty/garbage UAs.
		chMajor := defaultChromeMajor
		if m, ok := parseChromeMajor(prof.UserAgent); ok {
			chMajor = strconv.Itoa(m)
		}
		headers["Sec-Ch-Ua"] = fmt.Sprintf("\"Not A(Brand\";v=\"99\", \"Google Chrome\";v=\"%s\", \"Chromium\";v=\"%s\"", chMajor, chMajor)
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
