package types

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-rod/rod"
	"github.com/agenthands/godoll/browser"
	"github.com/agenthands/godoll/network"
	"github.com/agenthands/godoll/stealth"
	rodfingerprint "github.com/agenthands/godoll/fingerprint"
	"github.com/agenthands/rod-cli/internal/plugin"
	"github.com/agenthands/rod-cli/internal/cdpproxy"
	"github.com/agenthands/rod-cli/types/js"
	"github.com/agenthands/rod-cli/utils"
	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
)

// seams for testing: indirections through which production calls are routed so
// that otherwise-unreachable error branches can be exercised. Defaults are the
// real functions; tests swap them and restore via defer. Zero behavior change.
var (
	launcherLookPath = launcher.LookPath
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

// launchBrowser returns the browser plus a session cleanup func. The caller
// MUST invoke it after closing the browser: it stops the local godoll relay when
// an authenticated proxy spun one up (T-25-07) and removes the per-session
// browser profile dir. It is nil only on the CDPEndpoint path, which owns neither.
func launchBrowser(cfg Config) (*rod.Browser, *cdpproxy.Proxy, func(), error) {

	if cfg.CDPEndpoint != "" {
		b, err := controlBrowser(context.Background(), cfg.CDPEndpoint)
		return b, nil, nil, err
	}

	// Create basic launcher manually so we can set Headless, Proxy, UserDataDir
	browserLauncher := launcher.New().
		Context(context.Background()).
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
		Set("--disable-features", "HttpsUpgrades")

	if cfg.BrowserBinPath != "" {
		browserLauncher.Bin(cfg.BrowserBinPath)
	} else {
		if browserPath, has := launcherLookPath(); has {
			browserLauncher.Bin(browserPath)
		} else {
			return nil, nil, nil, errors.New("the machine does not have Chrome installed. Please run 'rod-cli install' to fetch it, or set the browser executable path.")
		}
	}

	// Per-session proxy via godoll's proxy API (replaces the bare
	// launcher.Proxy). cfg.Stealth.Proxy is the authoritative source. For the
	// no-auth case ApplyToLauncher just sets --proxy-server; for the auth case it
	// starts a local CONNECT relay (handling the SOCKS5/HTTP-auth gap) and returns
	// a cleanup that stops it. Credentials never reach --proxy-server.
	proxyCfg, err := parseProxyConfig(cfg.Stealth.Proxy, cfg.Stealth.ProxyAuth)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "invalid proxy configuration")
	}
	var proxyCleanup func()
	if proxyCfg != nil {
		proxyCleanup, err = proxyCfg.ApplyToLauncher(browserLauncher)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "apply proxy to launcher")
		}
		// Chrome bypasses the proxy for loopback hosts (localhost/127.0.0.1/[::1])
		// by default, so traffic to a local target never traverses --proxy-server.
		// "<-loopback>" removes loopback from the implicit bypass list, forcing
		// local targets through the proxy. Required for an in-process MITM proxy to
		// observe traffic to a local app (e.g. a containerized scan target);
		// harmless for remote proxies/targets.
		browserLauncher.Set("proxy-bypass-list", "<-loopback>")
	}

	// Create godoll options and apply stealth preset
	opts := browser.NewBrowserOptions().
		WithLauncher(browserLauncher).
		SetBrowserPreferences(browser.NewBrowserOptions().StealthPreset())

	// CDP-DEEP-01: optionally wrap the CDP WebSocket in a pass-through proxy
	// for traffic logging, Runtime normalization, and timing jitter.
	// --no-cdp-proxy bypasses even if --cdp-proxy is set.
	var cdpProxyInstance *cdpproxy.Proxy
	if boolVal(cfg.Stealth.CDPProxy, false) && !boolVal(cfg.Stealth.NoCDPProxy, false) {
		jitterMs := 0
		if cfg.Stealth.CDPJitterMs != nil {
			jitterMs = *cfg.Stealth.CDPJitterMs
		}
		if jitterMs > 1000 {
			fmt.Fprintf(os.Stderr, "warning: --cdp-jitter-ms=%d is very high; navigation may be extremely slow\n", jitterMs)
		}
		opts = opts.WithCDPWrapper(func(inner cdp.WebSocketable) cdp.WebSocketable {
			cdpProxyInstance = cdpproxy.New(inner, 1024, jitterMs)
			return cdpProxyInstance
		})
	}

	// HARDEN-01 browser-pref leg: disable non-proxied UDP so the real host IP
	// cannot leak past a proxy via WebRTC. Gated on the toggle (default on).
	if boolVal(cfg.Stealth.WebRTCLeakProtection, true) {
		opts = opts.WithWebRTCLeakProtection(true)
	}

	// browser must own a unique profile dir. Created last, so no early return
	// above can leak it.
	browserTempDir, err := newSessionProfile(cfg.BrowserTempDir, browserLauncher)
	if err != nil {
		if proxyCleanup != nil {
			proxyCleanup()
		}
		if browserTempDir != "" {
			removeProfileDir(browserTempDir)
		}
		return nil, nil, nil, errors.Wrap(err, "create browser profile dir")
	}
	browserLauncher.UserDataDir(browserTempDir)

	// The launcher never reaps browserTempDir; without this the profile dirs
	// accumulate under the profile root forever.
	sessionCleanup := func() {
		if proxyCleanup != nil {
			proxyCleanup()
		}
		removeSessionProfile(browserTempDir, browserLauncher)
	}

	browserInstance, err := browser.NewBrowserE(context.Background(), opts)
	if err != nil {
		sessionCleanup()
		return nil, nil, nil, errors.Wrap(err, "launch local browser failed via godoll")
	}

	if proxyCfg != nil && proxyCfg.HasAuth() {
		proxyCfg.SetupBrowserAuth(browserInstance)
	}

	return browserInstance, cdpProxyInstance, sessionCleanup, nil
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
	snapGen     int
	mode        Mode
	consoleLogs []string
	requests    []string
	// consoleDropped / requestsDropped count entries evicted by captureCap since
	// the last clear, so the console/requests output can say the log is incomplete.
	consoleDropped int
	requestsDropped int
	interceptor *network.Interceptor
	routes      map[string]string
	fingerprint *rodfingerprint.Fingerprint
	// profile is the active stealth.Profile that drives BOTH godoll's evasion JS
	// injection and the rod-cli network interceptor. It is built in preparePage by
	// overlaying the config-pinned cfg.Stealth identity onto stealth.DefaultProfile,
	// making the resolved cfg.Stealth the SINGLE source of truth for the live page
	// (FINGERPRINT-01 wiring). Nil until the first page is created.
	profile *stealth.Profile
	// noiseSeed is the per-session canvas/audio noise seed, generated once in
	// NewContext so re-reads within a session are stable (HARDEN-02). One daemon =
	// one session = one seed; a fresh daemon gets a fresh seed.
	noiseSeed uint64
	pluginEngine *plugin.PluginEngine
	loadedPlugins []string
	// sessionCleanup stops the per-session authenticated-proxy relay (godoll
	// StartProxyRelay) and removes the session profile dir. Nil on the
	// CDPEndpoint path; it is invoked exactly once on browser close and nil'd to
	// guard against double-call (T-25-07).
	sessionCleanup func()
	// cdpDomains is the per-session CDP domain-enable ledger (Phase 30 CDP-01 / D-04
	// instrumentation): the set of footprint-adding CDP domains this session has
	// enabled BEYOND what go-rod structurally requires (Page/Target/etc). Keyed by
	// CDPDomain* name. Written under stateLock at each enable-point: Runtime/Network
	// are recorded when their event subscription is REQUESTED (synchronously, before
	// the async EachEvent goroutine that triggers the enable — conservative: it can
	// only over-report, never hide); Fetch is recorded only after a CONFIRMED
	// interceptor Enable() success. It is cumulative ("ever enabled this session") —
	// closePage disabling the interceptor does not clear Fetch. Read via
	// GetEnabledCDPDomains. The deterministic offline harness asserts a plain session
	// records NONE of these — the falsifiable CDP-01 baseline gate.
	cdpProxy    *cdpproxy.Proxy
	cdpDomains map[string]bool
	// prepared records the TargetIDs preparePage has run on, so switchTo prepares
	// each tab exactly once.
	prepared map[proto.TargetTargetID]bool
	// engineSessions records the sessions installEngine has run on.
	engineSessions map[proto.TargetSessionID]bool
}

// CDP domain ledger keys (Phase 30 D-04). String constants so the instrumentation
// set-points and the harness assertion can never silently drift on a typo.
const (
	CDPDomainRuntime = "Runtime"
	CDPDomainNetwork = "Network"
	CDPDomainFetch   = "Fetch"
	CDPDomainDOM     = "DOM"
)

func NewContext(ctx context.Context, cfg Config) *Context {
	c := &Context{
		stdContext: ctx,
		config:     cfg,
		mode:       cfg.Mode,
		routes:     make(map[string]string),
		cdpDomains: make(map[string]bool),
	}
	// Generate the per-session canvas/audio noise seed ONCE (crypto/rand for an
	// unpredictable value). Generating it here guarantees the same seed across
	// every preparePage call for the life of the daemon, so re-reads of a canvas
	// within the session are stable (HARDEN-02).
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failure is near-impossible, but a silent zero seed would be
		// deterministic across every such daemon (cross-session fingerprint
		// correlation — the opposite of "fresh session = fresh seed"). Fall back to
		// a non-constant source rather than ship the predictable 0.
		binary.LittleEndian.PutUint64(b[:], uint64(time.Now().UnixNano())^uint64(os.Getpid()))
	}
	c.noiseSeed = binary.LittleEndian.Uint64(b[:])
	return c
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
		ctx.browser, ctx.cdpProxy, ctx.sessionCleanup, err = launchBrowser(ctx.config)
		if err != nil {
			return err
		}
		if err := ctx.autoPrepare(ctx.browser); err != nil {
			return errors.Wrap(err, "auto-attach failed")
		}
	}
	if ctx.page == nil {
		page, err := ctx.browser.Page(proto.TargetCreateTarget{})
		if err != nil {
			return errors.Wrap(err, "create page failed")
		}
		ctx.switchTo(page)
	}
	return nil
}

// autoPrepare makes Chrome pause every new target before its first script
// (Target.setAutoAttach with waitForDebuggerOnStart) until rod-cli has prepared
// it, so popups (window.open, target=_blank) and tabs rod-cli opens all run
// their first document with evasions. Targets that existed before the attach
// are not paused and stay unprepared until switchTo adopts them.
func (ctx *Context) autoPrepare(b *rod.Browser) error {
	go b.EachEvent(func(e *proto.TargetAttachedToTarget) {
		if e.WaitingForDebugger {
			go ctx.prepareAttached(b, e)
		}
	})()
	return proto.TargetSetAutoAttach{AutoAttach: true, WaitForDebuggerOnStart: true, Flatten: true}.Call(b)
}

// prepareAttached prepares a paused page target through its auto-attached
// session, then resumes it. That session stays attached, so its new-document
// scripts and overrides last for the life of the tab. It is not the session
// rod-cli drives the tab with: a Page.navigate already queued there (TabNew)
// would block the preparation until the resume. Every paused target is
// resumed, page or not, so none is left waiting.
func (ctx *Context) prepareAttached(b *rod.Browser, e *proto.TargetAttachedToTarget) {
	s := b.PageFromSession(e.SessionID)
	if e.TargetInfo.Type == proto.TargetTargetInfoTypePage {
		s.TargetID = e.TargetInfo.TargetID
		ctx.stateLock.Lock()
		if !ctx.prepared[s.TargetID] {
			ctx.preparePage(s)
		}
		ctx.stateLock.Unlock()
	}
	if err := (proto.RuntimeRunIfWaitingForDebugger{}).Call(s); err != nil {
		fmt.Fprintf(os.Stderr, "warning: resume new target failed: %v\n", err)
	}
}

func (ctx *Context) CurrentMode() Mode {
	return ctx.mode
}

// HumanizeTuning returns the session's resolved StealthConfig so the actions
// package can read the Phase-28 human-behavior tuning knobs (pointer-typed; nil
// = unset = emit no godoll option). config is unexported and actions is a
// separate package, so this exported accessor is the seam. Returning the whole
// StealthConfig (rather than the built option slices) keeps the godoll/humanize
// import OUT of types — actions, which already imports humanize, builds the
// option slices itself. The returned struct is a SHALLOW copy: its value fields
// are copied, but its pointer knobs (*int, *bool, …) still alias the frozen
// session config, so they are read-only by convention (the actions builders only
// deref them, never assign through them).
func (ctx *Context) HumanizeTuning() StealthConfig {
	return ctx.config.Stealth
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

// SwitchTo makes p the current tab — the only way to change it. Prepares p
// (stealth, emulation, capture) if prepareAttached has not, installs the
// snapshot engine on p's session, clears the snapshot, moves mock routes to p. Selecting the current tab is a
// no-op. p == nil means no current tab (next EnsurePage opens one).
// adopted = p existed before rod-cli attached; its loaded document has no
// evasions until the next navigation.
func (ctx *Context) SwitchTo(p *rod.Page) (adopted bool) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	return ctx.switchTo(p)
}

// switchTo is SwitchTo with stateLock held.
func (ctx *Context) switchTo(p *rod.Page) (adopted bool) {
	if p == nil && ctx.page == nil || p != nil && ctx.page != nil && p.TargetID == ctx.page.TargetID {
		return false
	}
	if p != nil && !ctx.prepared[p.TargetID] {
		adopted = true
		ctx.preparePage(p)
	}
	if p != nil && !ctx.engineSessions[p.SessionID] {
		ctx.installEngine(p)
	}
	if ctx.interceptor != nil {
		ctx.interceptor.Disable()
		ctx.interceptor = nil
	}
	ctx.snapshot, ctx.page = nil, p
	if p != nil && len(ctx.routes) > 0 {
		ctx.ensureInterceptorEnabled(p)
	}
	return adopted
}

// installEngine puts the snapshot engine on p's session: on every new document
// and on the current one. Isolated worlds belong to a session, so it goes on
// the session rod-cli drives p with, not the one prepareAttached used.
func (ctx *Context) installEngine(p *rod.Page) {
	if ctx.engineSessions == nil {
		ctx.engineSessions = make(map[proto.TargetSessionID]bool)
	}
	ctx.engineSessions[p.SessionID] = true
	// The engine lives in an isolated world so page scripts cannot see it.
	if _, err := (proto.PageAddScriptToEvaluateOnNewDocument{Source: js.InjectedSnapShot, WorldName: utils.EngineWorld}).Call(p); err != nil {
		fmt.Fprintf(os.Stderr, "warning: snapshot engine new-document script failed: %v\n", err)
	}
	if _, err := utils.EngineEval(p, js.InjectEngine); err != nil {
		fmt.Fprintf(os.Stderr, "warning: snapshot engine inject failed: %v\n", err)
	}
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

func (ctx *Context) BuildSnapshot(o SnapshotOpts) (string, error) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	if ctx.page == nil {
		return "", errors.New("No tab to capture snapshot, call rod_navigate first")
	}
	ctx.snapGen++
	o.Gen = ctx.snapGen
	snapshot, err := BuildSnapshot(ctx.page, o)
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
	// switchTo(nil) tears down the interceptor bound to this page; ctx.routes is
	// kept so the next page re-establishes it.
	p := ctx.page
	ctx.switchTo(nil)
	if err := p.Close(); err != nil {
		return errors.Wrap(err, "close page failed")
	}
	return nil
}
func (ctx *Context) closeBrowser() error {
	// Stop the per-session proxy relay and remove the profile on every path, after
	// the browser close below, so neither is leaked. Guard against double-call by
	// nil'ing.
	defer func() {
		if ctx.sessionCleanup != nil {
			ctx.sessionCleanup()
			ctx.sessionCleanup = nil
		}
	}()

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
// osForPlatform maps a navigator.platform value to the godoll fingerprint OS key
// (the FPWithOS constraint) so the generated fonts/codecs/media-devices are
// COHERENT with the profile's OS — Windows fonts on a Windows profile, macOS on a
// Mac profile (coherent-not-random lens). Matching is substring-based so platform
// variants (e.g. "Linux x86_64", "Linux aarch64") still resolve to the right OS
// key rather than falling through to the windows default inside the generator.
// Mobile keys (android/ios) are tested BEFORE the generic linux/mac substrings so
// an explicit android/ios platform token wins. NOTE the inherent limit: a genuine
// Android navigator.platform is "Linux armv8l" and iPadOS reports "MacIntel" —
// neither carries an android/ios token, so they resolve to linux/macos. That is
// acceptable under the current desktop-Chrome-only scope (no mobile profile
// exists); a mobile profile would need an explicit OS field, not platform
// sniffing. An unknown/empty platform falls back to windows (the safe default).
func osForPlatform(platform string) string {
	p := strings.ToLower(platform)
	switch {
	case strings.Contains(p, "android"):
		return "android"
	case strings.Contains(p, "iphone"), strings.Contains(p, "ipad"), strings.Contains(p, "ios"):
		return "ios"
	case strings.HasPrefix(p, "win"):
		return "windows"
	case strings.Contains(p, "mac"):
		return "macos"
	case strings.Contains(p, "linux"):
		return "linux"
	default:
		return "windows"
	}
}

// The resolved cfg.Stealth (produced by ResolveStealth's UA-anchored derive +
// coherence validation) is the single source of truth for the live page: this
// profile drives godoll's evasion JS injection AND the rod-cli interceptor.
// With an empty cfg.Stealth identity the result equals DefaultProfile() EXCEPT
// that Client-Hints spoofing is forced on (p.SpoofClientHints = true below) for
// coherence — DefaultProfile() ships it off. The no-pin path therefore emits
// Sec-Ch-Ua headers + navigator.userAgentData where the bare default would emit
// neither; see the lines below for the rationale (this is intentional, not a
// regression).
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
	// HARDEN-02: one toggle gates BOTH canvas and audio noise. With CanvasNoise
	// defaulting true (nil resolves to true), the no-pin path noises canvas+audio.
	canvasNoise := boolVal(s.CanvasNoise, true)
	p.SpoofCanvas = canvasNoise
	p.SpoofAudioContext = canvasNoise
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

// chPlatformFor maps a navigator.platform value to the Sec-Ch-Ua-Platform /
// UserAgentMetadata.Platform token (WITHOUT surrounding quotes — the Emulation
// metadata field is the bare value; Chrome quotes it on the wire). Mirrors the
// mapping that previously lived inline in updateInterceptorRules so the Emulation
// identity tells the same OS story the interceptor used to inject.
func chPlatformFor(platform string) string {
	switch platform {
	case "Win32":
		return "Windows"
	case "MacIntel":
		return "macOS"
	default:
		return platform
	}
}

// brandsForUA builds the Sec-Ch-Ua brand list from the active profile UA's
// `Chrome/<major>` token (parseChromeMajor; defaultChromeMajor fallback for
// empty/garbage UAs). It mirrors the EXACT tuple the interceptor catch-all used
// to inject at context.go ~671 so the Emulation-emitted Sec-Ch-Ua agrees with the
// UA and godoll's navigator.userAgentData brand version (FINGERPRINT-02). ONE
// derivation path: same parseChromeMajor as the rest of rod-cli.
func brandsForUA(ua string) []*proto.EmulationUserAgentBrandVersion {
	major := defaultChromeMajor
	if m, ok := parseChromeMajor(ua); ok {
		major = strconv.Itoa(m)
	}
	return []*proto.EmulationUserAgentBrandVersion{
		{Brand: "Not A(Brand", Version: "99"},
		{Brand: "Google Chrome", Version: major},
		{Brand: "Chromium", Version: major},
	}
}

// applyEmulationIdentity carries the active profile's HTTP identity onto the page
// via Chrome's Emulation domain (no `enable` command → zero CDP footprint). With
// UserAgentMetadata set, Chrome natively emits coherent Sec-Ch-Ua* headers and
// navigator.userAgentData WITHOUT Network.enable — the CDP-01 design-C mechanism.
// UA-metadata is attached only when the active profile spoofs Client-Hints (mirrors
// the prior interceptor gating). Log-and-continue on error (VALIDATE-03 discipline,
// matching em.Apply) so a failed override never aborts the daemon.
func (ctx *Context) applyEmulationIdentity(page *rod.Page, prof *stealth.Profile) {
	override := proto.EmulationSetUserAgentOverride{
		UserAgent:      prof.UserAgent,
		AcceptLanguage: prof.AcceptLanguage,
		Platform:       prof.Platform,
	}
	if prof.SpoofClientHints {
		override.UserAgentMetadata = &proto.EmulationUserAgentMetadata{
			Brands:   brandsForUA(prof.UserAgent),
			Platform: chPlatformFor(prof.Platform),
			Mobile:   false,
		}
	}
	if err := override.Call(page); err != nil {
		fmt.Fprintf(os.Stderr, "warning: emulation user-agent override failed: %v\n", err)
	}
}

// preparePage applies the per-page session setup (stealth evasions, emulation
// identity, opt-in capture). Caller MUST hold
// stateLock. Errors are logged and setup continues. Every page goes through it
// exactly once, from prepareAttached (new targets) or switchTo (adopted ones).
func (ctx *Context) preparePage(page *rod.Page) {
	if ctx.prepared == nil {
		ctx.prepared = make(map[proto.TargetTargetID]bool)
	}
	ctx.prepared[page.TargetID] = true
	// Apply stealth evasion
	em := stealth.NewEvasionManager(page)
	// Build the active stealth.Profile from the resolved cfg.Stealth: the
	// config-pinned identity is the SINGLE source of truth that drives both
	// godoll's evasion JS and rod-cli's interceptor (FINGERPRINT-01 wiring).
	// With an empty cfg.Stealth identity the overlay equals DefaultProfile(),
	// so the no-pin path is unchanged from the prior default.
	prof := profileFromStealth(ctx.config.Stealth)
	ctx.profile = &prof
	// EVAD-01/02 (Phase 33): generate the godoll fingerprint CONSTRAINED to the
	// profile's OS + locale so the dormant dimension injectors (fonts/media
	// devices/codecs/battery/plugins) tell the SAME OS story as the profile —
	// Windows fonts on a Windows profile, macOS fonts on a Mac profile. NEVER an
	// unconstrained random draw on a pinned identity (the coherent-not-random
	// lens: an incoherent dimension is a NEW tell, worse than not hardening).
	// The generator is SEEDED with the per-session noiseSeed so a page recreated
	// within this session reproduces identical dimensions (stability, EVAD
	// criterion 4).
	fpOpts := []rodfingerprint.Option{
		rodfingerprint.FPWithBrowserNames("chrome"),
		rodfingerprint.FPWithOS(osForPlatform(prof.Platform)),
	}
	if prof.Locale != "" {
		fpOpts = append(fpOpts, rodfingerprint.FPWithLocales(prof.Locale))
	}
	fg := rodfingerprint.NewFingerprintGeneratorSeeded(int64(ctx.noiseSeed), fpOpts...)
	fp, fpErr := fg.Generate()
	if fpErr != nil {
		fmt.Fprintf(os.Stderr, "warning: fingerprint generation failed: %v\n", fpErr)
	} else if fp != nil {
		ctx.fingerprint = fp
		// Order matters: em.SetFingerprint derives a profile from the fp
		// (FromFingerprint) and overwrites em.profile, so SetProfile MUST run
		// AFTER it to restore the config-pinned identity as the source of truth.
		// em.fingerprint stays set, which is what activates the dormant
		// applyFingerprintDimensions path (skipped today because it was nil).
		em.SetFingerprint(fp)
		// EVAD-02 per-vector gating: a toggle OFF skips that dimension's injection
		// entirely so the vector reverts to the un-hardened browser default
		// (provably effective, not cosmetic). Each defaults ON. `plugins` rides
		// the same godoll path (no separate toggle — see CONTEXT D-01).
		em.SetDimensionOptions(stealth.DimensionOptions{
			Fonts:        boolVal(ctx.config.Stealth.FontSpoof, true),
			MediaDevices: boolVal(ctx.config.Stealth.MediaDevicesSpoof, true),
			Battery:      boolVal(ctx.config.Stealth.BatterySpoof, true),
			Codecs:       boolVal(ctx.config.Stealth.CodecSpoof, true),
			Plugins:      true,
		})
	}
	em.SetProfile(prof)
	// HARDEN-02: thread the stable per-session noise seed before Apply() so the
	// seeded canvas/audio scripts produce identical re-reads within the session.
	em.SetNoiseSeed(ctx.noiseSeed)
	if err := em.Apply(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: evasion Apply failed: %v\n", err)
	}

	// HARDEN-01 JS leg: wrap RTCPeerConnection so local-IP enumeration cannot
	// leak past a proxy. Gated on the toggle; log-and-continue (VALIDATE-03) so a
	// failure never aborts the daemon — matches the em.Apply() discipline above.
	if boolVal(ctx.config.Stealth.WebRTCLeakProtection, true) {
		if err := em.EvadeWebRTC(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: WebRTC evasion failed: %v\n", err)
		}
	}

	// CDP-01 (design C+D, CONTEXT D-09): carry the HTTP identity natively via
	// Chrome's Emulation domain instead of the per-request interceptor catch-all.
	// Emulation.setUserAgentOverride has NO `enable` command, so this costs ZERO
	// CDP footprint while Chrome itself emits coherent UA / Sec-Ch-Ua* /
	// navigator.userAgentData (the FINGERPRINT-01/02 triple-agreement), letting a
	// plain session run with neither Runtime, Network, nor Fetch enabled. The old
	// always-on interceptor identity rule is removed (see updateInterceptorRules).
	ctx.applyEmulationIdentity(page, &prof)

	// Console capture (CDP-01): the RuntimeConsoleAPICalled subscription forces
	// Runtime.enable, so it is OPT-IN (default OFF). When off the subscription is
	// never registered, so a plain session never enables Runtime. The `console`
	// command requires the daemon spawned with --console-capture.
	if boolVal(ctx.config.Stealth.ConsoleCapture, false) {
		ctx.RecordCDPDomain(CDPDomainRuntime)
		go page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
			ctx.stateLock.Lock()
			defer ctx.stateLock.Unlock()
			var args []string
			for _, arg := range e.Args {
				if !arg.Value.Nil() {
					args = append(args, arg.Value.String())
				}
			}
			var n int
			ctx.consoleLogs, n = appendCapped(ctx.consoleLogs, fmt.Sprintf("[%s] %s", e.Type, strings.Join(args, " ")))
			ctx.consoleDropped += n
		}, func(e *proto.RuntimeExceptionThrown) {
			ctx.stateLock.Lock()
			defer ctx.stateLock.Unlock()
			msg := e.ExceptionDetails.Text
			if ex := e.ExceptionDetails.Exception; ex != nil && ex.Description != "" {
				msg = ex.Description
			}
			var n int
			ctx.consoleLogs, n = appendCapped(ctx.consoleLogs, "[exception] "+msg)
			ctx.consoleDropped += n
		})()
	}

	// Request capture (CDP-01): the NetworkRequestWillBeSent subscription forces
	// Network.enable, so it is OPT-IN (default OFF). The `requests`/`request`
	// commands require the daemon spawned with --request-capture.
	if boolVal(ctx.config.Stealth.RequestCapture, false) {
		ctx.RecordCDPDomain(CDPDomainNetwork)
		go page.EachEvent(func(e *proto.NetworkRequestWillBeSent) {
			ctx.stateLock.Lock()
			defer ctx.stateLock.Unlock()
			var n int
			ctx.requests, n = appendCapped(ctx.requests, fmt.Sprintf("%s %s", e.Request.Method, e.Request.URL))
			ctx.requestsDropped += n
		})()
	}
}

// captureCap bounds the console and request buffers so a chatty page cannot
// grow daemon memory without limit; the oldest entries are dropped and counted.
const captureCap = 2000

// appendCapped appends entry, trims the oldest entries beyond captureCap, and
// returns the buffer and how many entries it dropped.
func appendCapped(buf []string, entry string) ([]string, int) {
	buf = append(buf, entry)
	dropped := len(buf) - captureCap
	if dropped <= 0 {
		return buf, 0
	}
	return slices.Delete(buf, 0, dropped), dropped
}

// RecordCDPDomain marks a footprint-adding CDP domain as enabled for this
// session (Phase 30 D-04 instrumentation). The CALLER MUST HOLD stateLock — every
// set-point (the console/request subscription branches in preparePage, which run
// under initial()'s lock, and ensureInterceptorEnabled, called from AddRoute /
// switchTo under lock) already does.
func (ctx *Context) RecordCDPDomain(domain string) {
	if ctx.cdpDomains == nil {
		ctx.cdpDomains = make(map[string]bool)
	}
	ctx.cdpDomains[domain] = true
}

// GetEnabledCDPDomains returns a copy of the per-session CDP domain-enable ledger
// (Phase 30 D-04): the footprint-adding CDP domains this session enabled beyond
// go-rod's structural requirements. A plain session (no capture flags, no mock
// routes, no plugins) returns an EMPTY map — the falsifiable CDP-01 baseline that
// the offline harness asserts. Lock-safe.
func (ctx *Context) GetEnabledCDPDomains() map[string]bool {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	out := make(map[string]bool, len(ctx.cdpDomains))
	for k, v := range ctx.cdpDomains {
		if v {
			out[k] = true
		}
	}
	return out
}

// Accessors
// GetConsoleLogs returns the captured console entries and how many older
// entries the cap dropped since the last clear.
func (ctx *Context) GetConsoleLogs() ([]string, int) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	return slices.Clone(ctx.consoleLogs), ctx.consoleDropped
}

// GetRequests returns the captured requests and how many older entries the
// cap dropped since the last clear.
func (ctx *Context) GetRequests() ([]string, int) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	return slices.Clone(ctx.requests), ctx.requestsDropped
}

func (ctx *Context) ClearConsoleLogs() {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	ctx.consoleLogs = nil
	ctx.consoleDropped = 0
}

func (ctx *Context) ClearRequests() {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	ctx.requests = nil
	ctx.requestsDropped = 0
}

// GetCDPProxy returns the CDP proxy instance (nil if proxy is not enabled).
func (ctx *Context) GetCDPProxy() *cdpproxy.Proxy {
	return ctx.cdpProxy
}

// updateInterceptorRules rebuilds the interceptor's rule set from the current
// mock routes. The always-on identity catch-all rule was REMOVED in Phase 30
// (CDP-01): header coherence now rides on Emulation.setUserAgentOverride (see
// applyEmulationIdentity), which costs no CDP footprint. The interceptor exists
// ONLY to serve mock routes now, so it carries only Mock rules; unmatched requests
// fall through godoll's default replay (which preserves Chrome's Emulation-set
// identity headers). No-op when the interceptor has not been lazily created yet.
//
// Intentional behavior change: the old catch-all also force-deleted X-Requested-With
// (ModifiedHeaders[""]). That strip is dropped — Chrome never emits X-Requested-With
// natively (WIRE-VERIFY confirmed it absent on the plain Emulation path), so there
// is nothing to delete on the baseline; Emulation carries no header-deletion knob.
func (ctx *Context) updateInterceptorRules() {
	if ctx.interceptor == nil {
		return
	}

	ctx.interceptor.ClearRules()

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
}

// ensureInterceptorEnabled lazily creates + enables the network interceptor
// (godoll Fetch.enable) bound to the given page and applies the current mock-route
// rules, so the Fetch footprint follows the feature (CDP-01). No-op when the
// interceptor already exists or page is nil. Caller MUST hold stateLock.
func (ctx *Context) ensureInterceptorEnabled(page *rod.Page) {
	if ctx.interceptor != nil || page == nil {
		return
	}
	ic := network.NewInterceptor(page)
	ctx.interceptor = ic
	ctx.updateInterceptorRules()
	if err := ic.Enable(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to enable network interceptor: %v\n", err)
		ctx.interceptor = nil
		return
	}
	ctx.RecordCDPDomain(CDPDomainFetch)
}

// AddRoute registers a mock route, lazily creating + enabling the network
// interceptor on first use (footprint follows the feature, CDP-01). A session that
// never adds a route never enables Fetch. A route added BEFORE the first goto is
// stored and activates when switchTo re-establishes it on the new page.
func (ctx *Context) AddRoute(pattern, body string) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	ctx.routes[pattern] = body
	if ctx.interceptor == nil {
		ctx.ensureInterceptorEnabled(ctx.page)
		return
	}
	ctx.updateInterceptorRules()
}

// RemoveRoute drops a mock route. When the last route is removed the interceptor
// is disabled and discarded so Fetch interception stops (a later AddRoute lazily
// recreates it), keeping the footprint scoped to the feature's active lifetime.
func (ctx *Context) RemoveRoute(pattern string) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	delete(ctx.routes, pattern)
	if len(ctx.routes) == 0 && ctx.interceptor != nil {
		ctx.interceptor.Disable()
		ctx.interceptor = nil
		return
	}
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
	return ctx.closeBrowser()
}
