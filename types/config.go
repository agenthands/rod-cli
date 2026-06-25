package types

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/agenthands/godoll/stealth"
	"github.com/agenthands/rod-cli/utils"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const ConfigName = "rod-cli.yaml"

// StealthConfig is the cohesive, session-persistent home for every stealth knob
// rod-cli exposes. It is resolved ONCE at daemon-spawn time (see ResolveStealth)
// and frozen into the per-session daemon's Config at NewContext, so it is both
// session-persistent and session-isolated (one daemon process per named session).
//
// Phase 25 populated the proxy + profile-selection fields; Phase 26 implements
// the configurable-fingerprint identity pins (UserAgent/Locale/Timezone/Platform/
// Screen/AcceptLanguage/Languages/HardwareConcurrency/DeviceMemory/Vendor/
// SpoofClientHints) declared below. They overlay onto the same precedence
// resolver, so the flag → forward → Config.Stealth → NewContext path is reused.
// Phase 27 implements the canvas/WebGL/WebRTC hardening toggles
// (WebRTCLeakProtection, CanvasNoise) declared below. Phase 28 still reserves
// its fields:
//
//	Reserved for Phase 28 (human-behavior tuning):
//	  Typing/typo/jitter/mouse/scroll humanize knobs.
//
// When those land they overlay onto the same precedence resolver below, so the
// flag → forward → Config.Stealth → NewContext path established here is reused
// without re-architecting.
type StealthConfig struct {
	// Proxy is the proxy URL including scheme (http://host:port, socks5://host:port).
	// Authoritative source for the egress proxy; the deprecated Config.Proxy field
	// is bridged from this value for backward compatibility until Plan 02 rewires
	// the launchBrowser call site.
	Proxy string `yaml:"proxy" json:"proxy"`

	// ProxyAuth holds proxy credentials as "user:pass". Handled via CDP, never
	// URL-embedded. Credential-sensitive: it is set only at runtime from the
	// out-of-band ROD_CLI_PROXY_AUTH env var and is NEVER serialized — the
	// `yaml:"-" json:"-"` tags keep it out of any config/state file on disk.
	ProxyAuth string `yaml:"-" json:"-"`

	// ProfilePath is the resolved path to the stealth.Profile JSON file selected
	// via --profile. Empty when no profile was requested.
	ProfilePath string `yaml:"profilePath" json:"profilePath"`

	// --- Phase 26: configurable fingerprint identity pins ---
	// These overlay onto cfg.Stealth in ResolveStealth (profile at Tier 2, the
	// curated --user-agent/--locale/--timezone/--platform flags at Tier 1) and
	// are validated for coherence by deriveAndValidateFingerprint before launch.

	// UserAgent is the navigator.userAgent / HTTP UA string. It is the single
	// derivation anchor: the Chrome major version drives Client-Hints + the
	// userAgentData brand version, and the OS token drives Platform + the
	// Sec-Ch-Ua-Platform value when those are unset.
	UserAgent string `yaml:"userAgent" json:"userAgent"`

	// Locale is the BCP-47 locale (e.g. "en-US"). Derived from Languages[0] when unset.
	Locale string `yaml:"locale" json:"locale"`

	// Timezone is the IANA timezone ID (e.g. "America/New_York").
	Timezone string `yaml:"timezone" json:"timezone"`

	// Platform is navigator.platform (e.g. "Win32", "MacIntel", "Linux").
	// Auto-derived from the UA OS token when unset.
	Platform string `yaml:"platform" json:"platform"`

	// AcceptLanguage is the HTTP Accept-Language header value.
	AcceptLanguage string `yaml:"acceptLanguage" json:"acceptLanguage"`

	// Languages is navigator.languages (e.g. ["en-US", "en"]).
	Languages []string `yaml:"languages" json:"languages"`

	// HardwareConcurrency is navigator.hardwareConcurrency (CPU cores).
	HardwareConcurrency int `yaml:"hardwareConcurrency" json:"hardwareConcurrency"`

	// DeviceMemory is navigator.deviceMemory (GB).
	DeviceMemory int `yaml:"deviceMemory" json:"deviceMemory"`

	// Vendor is navigator.vendor (e.g. "Google Inc.").
	Vendor string `yaml:"vendor" json:"vendor"`

	// SpoofClientHints enables injection of Sec-Ch-Ua headers + navigator.userAgentData.
	SpoofClientHints bool `yaml:"spoofClientHints" json:"spoofClientHints"`

	// --- Phase 27: canvas/WebGL/WebRTC hardening toggles ---

	// WebRTCLeakProtection prevents the real host IP from leaking past a proxy via
	// WebRTC ICE candidates. When true (the default) launchBrowser sets the
	// disable-non-proxied-UDP browser preference AND createPage injects godoll's
	// EvadeWebRTC JS wrapper. A *bool so a yaml-persisted explicit false survives
	// ResolveStealth (nil = unset = resolve to the default true); a plain bool
	// could not distinguish "unset" from a deliberate file-set false. Read it via
	// boolVal(cfg.Stealth.WebRTCLeakProtection, true) at consumers.
	WebRTCLeakProtection *bool `yaml:"webRTCLeakProtection" json:"webRTCLeakProtection"`

	// CanvasNoise enables seeded, stable-per-session canvas/WebGL/audio readback
	// noise (HARDEN-02). When true (the default) the active profile's SpoofCanvas
	// and SpoofAudioContext are enabled so canvas + audio fingerprints carry a
	// session-stable per-pixel/per-sample delta. A *bool for the same round-trip
	// reason as WebRTCLeakProtection (nil = unset = resolve to the default true).
	CanvasNoise *bool `yaml:"canvasNoise" json:"canvasNoise"`

	// --- Phase 28: human-behavior tuning knobs (HUMANIZE-01) ---
	//
	// Every tunable is a POINTER so nil = "unset" is distinguishable from an
	// explicit value (including an explicit zero/false). nil means NO godoll
	// option is emitted at the action call site, so godoll's own default applies
	// and behavior is byte-for-byte the current default (the zero-regression
	// invariant). A plain value could not carry "unset", and a yaml-persisted
	// explicit value would be clobbered by a baseline default — the Phase-27 CR-02
	// lesson, carried forward.
	//
	// Speed ranges (TypingSpeed*, MouseSpeed*) are exposed as two fields each; the
	// corresponding godoll min/max option is emitted ONLY when BOTH ends are set
	// (otherwise nil → godoll default). Note: "delay jitter" has no dedicated
	// godoll option — it IS the variance produced by the TypingSpeed min/max
	// spread, so there is deliberately no separate jitter field.

	// TypingSpeedMin / TypingSpeedMax are the per-keystroke delay range in
	// milliseconds (humanize.WithTypingSpeed(min, max)). Both must be set to emit
	// the option. A wider spread = more "delay jitter".
	TypingSpeedMin *int `yaml:"typingSpeedMin" json:"typingSpeedMin"`
	TypingSpeedMax *int `yaml:"typingSpeedMax" json:"typingSpeedMax"`

	// TypoRate is the probability (0.0-1.0) of an injected typo per keystroke
	// (humanize.WithTypoRate).
	TypoRate *float32 `yaml:"typoRate" json:"typoRate"`

	// MouseTremor enables/disables microscopic tremor on the mouse path
	// (humanize.WithMouseTremor). A *bool so an explicit false (disable tremor)
	// is honored distinctly from unset (godoll default).
	MouseTremor *bool `yaml:"mouseTremor" json:"mouseTremor"`

	// MouseSteps is the number of interpolation steps along the mouse path
	// (humanize.WithMouseSteps); more steps = smoother/slower motion.
	MouseSteps *int `yaml:"mouseSteps" json:"mouseSteps"`

	// MouseSpeedMin / MouseSpeedMax are the mouse speed range in pixels/second
	// (humanize.WithMouseSpeed(min, max)). Both must be set to emit the option.
	MouseSpeedMin *int `yaml:"mouseSpeedMin" json:"mouseSpeedMin"`
	MouseSpeedMax *int `yaml:"mouseSpeedMax" json:"mouseSpeedMax"`

	// MouseDeviation is the mouse-path randomness factor 0.0-1.0
	// (humanize.WithMouseDeviation).
	MouseDeviation *float64 `yaml:"mouseDeviation" json:"mouseDeviation"`

	// ScrollDuration is the base scroll animation duration in milliseconds
	// (humanize.WithDuration).
	ScrollDuration *int `yaml:"scrollDuration" json:"scrollDuration"`

	// ScrollPhysics requests physics-based (cubic-bezier-eased) scrolling
	// (humanize.WithPhysics). NOTE: godoll's WithPhysics() can only ENABLE
	// physics — godoll has no option to disable it and physics is godoll's own
	// default — so a true value emits WithPhysics() and a false/nil value emits
	// nothing (godoll's default physics still applies). Disabling physics would
	// require a godoll signature change (out of scope for v1.6). A *bool is kept
	// for round-trip symmetry and forward-compat.
	ScrollPhysics *bool `yaml:"scrollPhysics" json:"scrollPhysics"`

	// Screen holds the spoofed screen geometry.
	Screen struct {
		Width             int     `yaml:"width" json:"width"`
		Height            int     `yaml:"height" json:"height"`
		DeviceScaleFactor float64 `yaml:"deviceScaleFactor" json:"deviceScaleFactor"`
	} `yaml:"screen" json:"screen"`
}

type Config struct {
	Mode           Mode          `yaml:"mode" json:"mode"`
	CDPEndpoint    string        `yaml:"cdpEndpoint" json:"cdpEndpoint"`
	BrowserBinPath string        `yaml:"browserBinPath" json:"browserBinPath"`
	Headless       bool          `yaml:"headless" json:"headless"`
	BrowserTempDir string        `yaml:"browserTempDir" json:"browserTempDir"`
	NoSandbox      bool          `yaml:"noSandbox" json:"noSandbox"`
	// Proxy is DEPRECATED: it is bridged from Stealth.Proxy by ResolveStealth and
	// kept only so the in-flight types/context.go launchBrowser call site still
	// compiles. Plan 02 removes that call site; prefer Stealth.Proxy everywhere.
	Proxy        string        `yaml:"proxy" json:"proxy"`
	Stealth      StealthConfig `yaml:"stealth" json:"stealth"`
	LoggerConfig LoggerConfig  `yaml:"loggerConfig" json:"loggerConfig"`
	Raw          bool          `yaml:"raw" json:"raw"`
	Json         bool          `yaml:"json" json:"json"`
}

// StealthFlags carries the raw CLI flag values for the stealth surface, captured
// off the cli.Context at daemon spawn. It is the highest-precedence input to
// ResolveStealth.
type StealthFlags struct {
	// Proxy is the --proxy value (proxy URL with scheme).
	Proxy string
	// ProxyAuth is the --proxy-auth value ("user:pass").
	ProxyAuth string
	// Profile is the --profile value (a bare name or a path to a JSON profile).
	Profile string
	// UserAgent is the --user-agent value (the fingerprint derivation anchor).
	UserAgent string
	// Locale is the --locale value (BCP-47, e.g. "en-US").
	Locale string
	// Timezone is the --timezone value (IANA ID, e.g. "America/New_York").
	Timezone string
	// Platform is the --platform value (navigator.platform, e.g. "Win32").
	Platform string
	// WebRTCLeakProtection is the --webrtc-protection value; nil when unset
	// (default-on). A *bool so "unset" (keep the default-true) is distinguishable
	// from an explicit "--webrtc-protection=false".
	WebRTCLeakProtection *bool
	// CanvasNoise is the --canvas-noise value; nil when unset (default-on). A
	// *bool so "unset" is distinguishable from an explicit "--canvas-noise=false".
	CanvasNoise *bool

	// --- Phase 28: human-behavior tuning flags (HUMANIZE-01) ---
	// Each is a pointer captured only when the corresponding flag IsSet, so an
	// unset flag stays nil and ResolveStealth leaves cfg.Stealth's value alone
	// (no zero-override). Mirror of the StealthConfig humanize fields above.
	TypingSpeedMin *int
	TypingSpeedMax *int
	TypoRate       *float32
	MouseTremor    *bool
	MouseSteps     *int
	MouseSpeedMin  *int
	MouseSpeedMax  *int
	MouseDeviation *float64
	ScrollDuration *int
	ScrollPhysics  *bool
}

// resolveProfilePath maps a --profile value to a concrete file path. An empty
// value yields an empty path (no profile). A value that already looks like a
// path (contains a separator or a .json suffix, or exists on disk) is used
// verbatim; otherwise a bare name is resolved under the default profiles dir
// ~/.rod-cli/profiles/<name>.json. If the home dir cannot be determined, the
// name is resolved relative to ./profiles/<name>.json.
func resolveProfilePath(profile string) string {
	if profile == "" {
		return ""
	}
	if strings.ContainsRune(profile, os.PathSeparator) ||
		strings.HasSuffix(profile, ".json") ||
		strings.ContainsRune(profile, '/') {
		return profile
	}
	if _, err := os.Stat(profile); err == nil {
		return profile
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".rod-cli", "profiles", profile+".json")
	}
	return filepath.Join("profiles", profile+".json")
}

// ResolveStealth populates cfg.Stealth using the precedence
//
//	CLI flag > profile file > built-in default
//
// It is the single funnel for stealth config and is intended to run exactly once
// per daemon, before NewContext freezes Config. A missing or malformed --profile
// file is a LOUD failure: the error is returned and the caller must abort rather
// than silently ship a default identity.
func ResolveStealth(cfg *Config, flags *StealthFlags) error {
	if cfg == nil {
		return errors.New("ResolveStealth: nil config")
	}
	if flags == nil {
		flags = &StealthFlags{}
	}

	// Tier 3: built-in defaults. DefaultProfile() is the identity default source;
	// the proxy fields default to empty (no proxy).
	_ = stealth.DefaultProfile()

	// Tier 2: profile file. A bad load is loud — do NOT swallow and fall back.
	if flags.Profile != "" {
		path := resolveProfilePath(flags.Profile)
		prof, err := stealth.LoadProfile(path)
		if err != nil {
			return errors.Wrapf(err, "load stealth profile %q", path)
		}
		cfg.Stealth.ProfilePath = path
		// Overlay the loaded profile's identity fields onto cfg.Stealth so the
		// consistency validator (and downstream injectors) see one identity.
		cfg.Stealth.UserAgent = prof.UserAgent
		cfg.Stealth.Platform = prof.Platform
		cfg.Stealth.AcceptLanguage = prof.AcceptLanguage
		cfg.Stealth.Languages = prof.Languages
		cfg.Stealth.Timezone = prof.Timezone
		cfg.Stealth.Locale = prof.Locale
		cfg.Stealth.Vendor = prof.Vendor
		cfg.Stealth.HardwareConcurrency = prof.HardwareConcurrency
		cfg.Stealth.DeviceMemory = prof.DeviceMemory
		cfg.Stealth.SpoofClientHints = prof.SpoofClientHints
		cfg.Stealth.Screen.Width = prof.Screen.Width
		cfg.Stealth.Screen.Height = prof.Screen.Height
		cfg.Stealth.Screen.DeviceScaleFactor = prof.Screen.DeviceScaleFactor
	}

	// Phase-27 hardening toggles, resolved precedence:
	//   explicit --flag (StealthFlags *bool) > yaml-loaded cfg value (*bool, honored
	//   when non-nil — INCLUDING an explicit false) > built-in default true.
	// We do NOT unconditionally re-baseline to true: a config file that persisted
	// `canvasNoise: false` arrives here as a non-nil *bool(false) and must survive.
	// Only a nil (omitted key, no flag) resolves to the hardened default true.
	if flags.WebRTCLeakProtection != nil {
		cfg.Stealth.WebRTCLeakProtection = flags.WebRTCLeakProtection
	} else if cfg.Stealth.WebRTCLeakProtection == nil {
		cfg.Stealth.WebRTCLeakProtection = boolPtr(true)
	}
	if flags.CanvasNoise != nil {
		cfg.Stealth.CanvasNoise = flags.CanvasNoise
	} else if cfg.Stealth.CanvasNoise == nil {
		cfg.Stealth.CanvasNoise = boolPtr(true)
	}

	// Phase-28 humanize tuning, resolved precedence:
	//   explicit --flag (non-nil StealthFlags pointer) > yaml-loaded cfg value
	//   (non-nil) > unset (nil, LEFT nil).
	// Unlike the Phase-27 toggles there is NO default-true baseline: nil is the
	// load-bearing "emit no godoll option ⇒ godoll's own default applies ⇒
	// byte-for-byte current behavior" signal (the zero-regression invariant).
	// We only OVERRIDE when the flag is set; a nil flag preserves whatever the
	// yaml/profile already put on cfg.Stealth (which is also nil when omitted).
	if flags.TypingSpeedMin != nil {
		cfg.Stealth.TypingSpeedMin = flags.TypingSpeedMin
	}
	if flags.TypingSpeedMax != nil {
		cfg.Stealth.TypingSpeedMax = flags.TypingSpeedMax
	}
	if flags.TypoRate != nil {
		cfg.Stealth.TypoRate = flags.TypoRate
	}
	if flags.MouseTremor != nil {
		cfg.Stealth.MouseTremor = flags.MouseTremor
	}
	if flags.MouseSteps != nil {
		cfg.Stealth.MouseSteps = flags.MouseSteps
	}
	if flags.MouseSpeedMin != nil {
		cfg.Stealth.MouseSpeedMin = flags.MouseSpeedMin
	}
	if flags.MouseSpeedMax != nil {
		cfg.Stealth.MouseSpeedMax = flags.MouseSpeedMax
	}
	if flags.MouseDeviation != nil {
		cfg.Stealth.MouseDeviation = flags.MouseDeviation
	}
	if flags.ScrollDuration != nil {
		cfg.Stealth.ScrollDuration = flags.ScrollDuration
	}
	if flags.ScrollPhysics != nil {
		cfg.Stealth.ScrollPhysics = flags.ScrollPhysics
	}

	// Fail fast on out-of-range / inverted / incomplete humanize tuning BEFORE the
	// browser launches. godoll's rand.RandomDuration/RandomInt PANIC on negative or
	// min>max input (../godoll/internal/rand/rand.go), and that panic would fire
	// per-keystroke deep inside a frozen daemon session — an opaque, unrecoverable
	// break of every type/fill/mouse action. Rejecting here (the same spawn-time
	// seam as deriveAndValidateFingerprint) turns it into a clear refusal to spawn.
	if err := validateHumanizeTuning(&cfg.Stealth); err != nil {
		return err
	}

	// Tier 1: CLI flags win over the profile and the defaults.
	if flags.Proxy != "" {
		cfg.Stealth.Proxy = flags.Proxy
	}
	if flags.ProxyAuth != "" {
		cfg.Stealth.ProxyAuth = flags.ProxyAuth
	}
	if flags.UserAgent != "" {
		cfg.Stealth.UserAgent = flags.UserAgent
	}
	if flags.Locale != "" {
		cfg.Stealth.Locale = flags.Locale
	}
	if flags.Timezone != "" {
		cfg.Stealth.Timezone = flags.Timezone
	}
	if flags.Platform != "" {
		cfg.Stealth.Platform = flags.Platform
	}

	// Consistency gate: derive unset dependents from the UA anchor and loudly
	// reject user-set contradictions BEFORE the browser launches. The error
	// propagates up through main.go's ResolveStealth call and aborts the daemon
	// rather than shipping a mismatched identity. Track which fields the user set
	// explicitly so the validator only rejects user-set conflicts (derive-when-
	// unset, reject-when-user-conflicts).
	userSet := userSetFingerprint{
		Platform: flags.Platform != "",
		Locale:   flags.Locale != "",
	}
	if err := deriveAndValidateFingerprint(cfg, userSet); err != nil {
		return err
	}

	// Bridge the deprecated compatibility shim.
	cfg.Proxy = cfg.Stealth.Proxy
	return nil
}

// chromeMajorRe extracts the integer Chrome major version from a UA string.
var chromeMajorRe = regexp.MustCompile(`Chrome/(\d+)`)

// parseChromeMajor returns the integer after "Chrome/" in a UA string. ok is
// false when the UA contains no Chrome token. This is the single derivation
// anchor for Client-Hints + navigator.userAgentData brand versions (the actual
// brand-string formatting belongs to the runtime injector, not here).
func parseChromeMajor(ua string) (int, bool) {
	m := chromeMajorRe.FindStringSubmatch(ua)
	if len(m) < 2 {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return n, true
}

// uaOSToPlatform maps the OS token in a UA string to navigator.platform and the
// Sec-Ch-Ua-Platform value. ok is false when no known OS token is present.
func uaOSToPlatform(ua string) (platform string, chPlatform string, ok bool) {
	switch {
	case strings.Contains(ua, "Windows NT"):
		return "Win32", "Windows", true
	case strings.Contains(ua, "Macintosh"), strings.Contains(ua, "Mac OS X"):
		return "MacIntel", "macOS", true
	case strings.Contains(ua, "X11"), strings.Contains(ua, "Linux"):
		return "Linux", "Linux", true
	}
	return "", "", false
}

// knownPlatforms is the set of navigator.platform values rod-cli can emit a
// coherent identity for. Used by WR-03 to validate a user-pinned --platform when
// the UA carries no recognized OS token (so the contradiction check can't run).
var knownPlatforms = []string{"Win32", "MacIntel", "Linux"}

// isKnownPlatform reports whether p is one of the platforms rod-cli recognizes.
func isKnownPlatform(p string) bool {
	for _, k := range knownPlatforms {
		if p == k {
			return true
		}
	}
	return false
}

// rejectUnsafeFingerprintValue rejects a fingerprint string field that carries a
// control character (incl. \r/\n) or a double-quote. Such a value would need
// escaping at the JS injection boundary (godoll's injected script literals) and
// could smuggle/break interceptor headers, so it is almost certainly malformed:
// fail loud and early, naming the field, rather than shipping it. An empty value
// is always safe (means "unset/derive downstream").
func rejectUnsafeFingerprintValue(field, val string) error {
	if val == "" {
		return nil
	}
	for _, r := range val {
		if r == '"' {
			return errors.Errorf("fingerprint field %s contains a double-quote, which is not allowed: %q", field, val)
		}
		// Reject ASCII control characters (C0, DEL) including CR/LF/tab.
		if r < 0x20 || r == 0x7f {
			return errors.Errorf("fingerprint field %s contains a control character (0x%02x), which is not allowed: %q", field, r, val)
		}
	}
	return nil
}

// userSetFingerprint records which fingerprint fields the user explicitly set via
// a CLI flag, so the validator rejects only genuine user-set contradictions and
// silently auto-derives the rest.
type userSetFingerprint struct {
	Platform bool
	Locale   bool
}

// deriveAndValidateFingerprint enforces the consistency policy on the already-
// overlaid cfg.Stealth: derive-when-unset, reject-when-user-conflicts. It runs at
// daemon-spawn config-resolution time, before NewContext freezes Config and the
// browser launches, so any contradiction fails fast on stderr (via the returned
// error, which main.go surfaces) rather than shipping a mismatched lie.
func deriveAndValidateFingerprint(cfg *Config, userSet userSetFingerprint) error {
	s := &cfg.Stealth

	// Defense-in-depth (CR-02 / WR-02): reject any string field that carries a
	// control character (incl. \r/\n) or a double-quote BEFORE it can reach the JS
	// injection boundary (godoll interpolates these into injected script literals)
	// or the interceptor header map. godoll now json.Marshal-escapes at that
	// boundary, but a value that needs escaping is almost certainly malformed, so
	// we fail loud and early — naming the field — rather than silently shipping a
	// corrupted override or a smuggled header.
	for _, f := range []struct {
		name string
		val  string
	}{
		{"--user-agent", s.UserAgent},
		{"--platform", s.Platform},
		{"--timezone", s.Timezone},
		{"vendor", s.Vendor},
		{"--locale", s.Locale},
		{"acceptLanguage", s.AcceptLanguage},
	} {
		if err := rejectUnsafeFingerprintValue(f.name, f.val); err != nil {
			return err
		}
	}
	for i, lang := range s.Languages {
		if err := rejectUnsafeFingerprintValue(fmt.Sprintf("languages[%d]", i), lang); err != nil {
			return err
		}
	}

	// The UA is the derivation anchor. With no UA set, DefaultProfile supplies a
	// coherent identity downstream — leave validation a no-op.
	ua := s.UserAgent
	if ua == "" {
		// Still range-check explicitly-set hardware/screen values even without a UA.
		return validateHardwareAndScreen(s)
	}

	// Platform ↔ UA OS token. Derive when unset; reject a user-set contradiction.
	if derivedPlatform, derivedCH, ok := uaOSToPlatform(ua); ok {
		if userSet.Platform && s.Platform != "" && s.Platform != derivedPlatform {
			return errors.Errorf(
				"platform %q contradicts UA OS %q — remove --platform to auto-derive, or fix --user-agent",
				s.Platform, derivedCH)
		}
		if s.Platform == "" {
			s.Platform = derivedPlatform
		}
	} else if userSet.Platform && s.Platform != "" {
		// WR-03: the UA carries no recognized OS token, so platform↔UA coherence
		// cannot be derived/verified. Do NOT fail open — a user-pinned platform that
		// silently ships an unverifiable mismatch is a fingerprint tell. Validate the
		// pin against the known navigator.platform set and reject anything else,
		// naming the field so the user can fix the UA or the platform.
		if !isKnownPlatform(s.Platform) {
			return errors.Errorf(
				"platform %q could not be verified against UA %q (UA carries no recognized OS token) "+
					"and is not one of the known platforms %v — fix --user-agent so its OS token is "+
					"recognized, or set --platform to a known value",
				s.Platform, ua, knownPlatforms)
		}
	}

	// Locale ↔ languages ↔ Accept-Language. Reject only when the user explicitly
	// set a Locale that contradicts the first navigator.languages entry; otherwise
	// derive Locale from Languages[0] when unset.
	if len(s.Languages) > 0 && s.Languages[0] != "" {
		if userSet.Locale && s.Locale != "" && !localeMatchesLanguage(s.Locale, s.Languages[0]) {
			return errors.Errorf(
				"locale %q contradicts navigator.languages[0] %q — remove --locale to auto-derive, or fix the profile languages",
				s.Locale, s.Languages[0])
		}
		if s.Locale == "" {
			s.Locale = s.Languages[0]
		}
	}

	if err := validateHardwareAndScreen(s); err != nil {
		return err
	}

	// timezone ↔ proxy-geo is WARN-ONLY: geo-IP resolution needs network access
	// and would break the offline-deterministic harness. Emit a single stderr line
	// (never stdout, never a hard failure) when both are set.
	if s.Timezone != "" && s.Proxy != "" {
		fmt.Fprintf(os.Stderr,
			"warning: --timezone %q is set alongside a proxy; rod-cli does not verify the timezone matches the proxy's geo-IP (offline-deterministic)\n",
			s.Timezone)
	}

	return nil
}

// localeMatchesLanguage reports whether a locale and a navigator.languages entry
// agree. Both are normalized to their primary subtag (case-insensitive) so e.g.
// "en-US" matches "en-GB" at the language level but "fr-FR" does not match "en-US".
func localeMatchesLanguage(locale, language string) bool {
	primary := func(s string) string {
		s = strings.TrimSpace(s)
		if i := strings.IndexAny(s, "-_"); i >= 0 {
			s = s[:i]
		}
		return strings.ToLower(s)
	}
	return primary(locale) == primary(language)
}

// validateHardwareAndScreen rejects implausible explicitly-set hardware/screen
// values with a field-naming message (fail-fast, before browser launch). Zero
// values mean "unset/derive downstream" and are not range-checked.
func validateHardwareAndScreen(s *StealthConfig) error {
	if s.Screen.Width < 0 || s.Screen.Height < 0 {
		return errors.Errorf("implausible screen geometry: width=%d height=%d — both must be positive",
			s.Screen.Width, s.Screen.Height)
	}
	if (s.Screen.Width > 0) != (s.Screen.Height > 0) {
		return errors.Errorf("incomplete screen geometry: width=%d height=%d — set both or neither",
			s.Screen.Width, s.Screen.Height)
	}
	if s.HardwareConcurrency != 0 && (s.HardwareConcurrency < 1 || s.HardwareConcurrency > 256) {
		return errors.Errorf("implausible hardwareConcurrency %d — must be between 1 and 256", s.HardwareConcurrency)
	}
	if s.DeviceMemory != 0 && (s.DeviceMemory < 1 || s.DeviceMemory > 64) {
		return errors.Errorf("implausible deviceMemory %d — must be between 1 and 64 (GB)", s.DeviceMemory)
	}
	return nil
}

// validateHumanizeTuning rejects out-of-range, inverted, or incomplete Phase-28
// humanize tuning BEFORE it can reach a godoll option at an action call site.
// This is fail-fast at config-resolution time (the daemon-spawn seam): a bad
// value caught here returns an error main.go surfaces on stderr and aborts the
// daemon, rather than panicking per-keystroke inside a frozen session (godoll's
// rand.RandomDuration/RandomInt panic on negative or min>max input). nil fields
// are "unset" and skipped — only explicitly-set values are checked.
func validateHumanizeTuning(s *StealthConfig) error {
	// Typing speed: a min/max pair feeding godoll's WithTypingSpeed →
	// rand.RandomDuration(min,max). Both ends or neither (a lone end is silently
	// inert at the builder, which is a confusing no-op — reject it explicitly).
	// Require 0 <= min <= max.
	if err := validateSpeedPair("--typing-speed", s.TypingSpeedMin, s.TypingSpeedMax); err != nil {
		return err
	}
	// Mouse speed: same pair shape, feeds WithMouseSpeed.
	if err := validateSpeedPair("--mouse-speed", s.MouseSpeedMin, s.MouseSpeedMax); err != nil {
		return err
	}

	if s.TypoRate != nil && (*s.TypoRate < 0 || *s.TypoRate > 1) {
		return errors.Errorf("typo-rate %v out of range — must be between 0.0 and 1.0", *s.TypoRate)
	}
	if s.MouseDeviation != nil && (*s.MouseDeviation < 0 || *s.MouseDeviation > 1) {
		return errors.Errorf("mouse-deviation %v out of range — must be between 0.0 and 1.0", *s.MouseDeviation)
	}
	if s.MouseSteps != nil && *s.MouseSteps < 1 {
		return errors.Errorf("mouse-steps %d invalid — must be a positive integer", *s.MouseSteps)
	}
	if s.ScrollDuration != nil && *s.ScrollDuration < 1 {
		return errors.Errorf("scroll-duration %d invalid — must be a positive integer (milliseconds)", *s.ScrollDuration)
	}
	return nil
}

// validateSpeedPair enforces that a min/max humanize speed range is either fully
// unset or fully set with 0 <= min <= max. An incomplete pair is rejected (the
// builders pair-gate it to a silent no-op, which is a confusing half-honored
// input); an inverted or negative pair would panic godoll's rand at the call
// site, so it is rejected here at spawn time.
func validateSpeedPair(name string, min, max *int) error {
	if (min == nil) != (max == nil) {
		return errors.Errorf("incomplete %s range — set both %s-min and %s-max or neither", name, name, name)
	}
	if min == nil { // both nil → unset
		return nil
	}
	if *min < 0 || *max < 0 {
		return errors.Errorf("%s range %d..%d invalid — both ends must be non-negative", name, *min, *max)
	}
	if *min > *max {
		return errors.Errorf("%s range %d..%d invalid — min must be <= max", name, *min, *max)
	}
	return nil
}

// boolPtr returns a pointer to b. Used for the *bool default-true hardening
// toggles (WebRTCLeakProtection / CanvasNoise) so "unset" (nil) is distinguishable
// from a deliberate false.
func boolPtr(b bool) *bool { return &b }

// boolVal dereferences a *bool toggle, returning def when the pointer is nil
// (unset). Consumers of the hardening toggles read them through this so a nil
// (never-resolved) value still falls back to the hardened default.
func boolVal(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

var (
	DefaultBrowserTempDir = "./rod/browser"

	DefaultConfig = Config{
		BrowserBinPath: "",
		Headless:       false,
		BrowserTempDir: DefaultBrowserTempDir,
		NoSandbox:      false,
		Proxy:          "",
		// Hardened-by-default: a config loaded with zero StealthFlags still gets
		// WebRTC leak protection and stable canvas/audio noise (Phase 27).
		Stealth: StealthConfig{WebRTCLeakProtection: boolPtr(true), CanvasNoise: boolPtr(true)},
		LoggerConfig:   DefaultLoggerConfig,
		Mode:           Text,
		Raw:            false,
		Json:           false,
	}
)

// InitDefaultConfig Generate the default configuration file
func InitDefaultConfig() error {

	// First, check if the configuration file exists at the default path. If it exists, do not generate the default configuration file.
	defaultConfigPath := filepath.Join("./", ConfigName)
	if exist, _ := utils.PathExists(defaultConfigPath); exist {
		return nil
	}

	// if default config file not exist, create it
	defaultConfig, err := os.Create(defaultConfigPath)
	if err != nil {
		return err
	}

	encoder := yaml.NewEncoder(defaultConfig)
	defer encoder.Close()

	err = encoder.Encode(DefaultConfig)
	if err != nil {
		return err
	}
	return nil
}

// LoadConfig Actually load the configuration file
// if ConfigPath is empty, generate the default configuration file in the current directory
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = filepath.Join("./", ConfigName)
		if err := InitDefaultConfig(); err != nil {
			return nil, errors.Wrapf(err, "init default config failed")
		}
	}

	// check if config file exist
	exist, err := utils.PathExists(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not open config file")
	}

	if exist {
		// validate config file name
		fileName := utils.FileName(configPath)
		if strings.Contains(ConfigName, fileName) {
			file, err := os.Open(configPath)
			if err != nil {
				return nil, err
			}
			defer file.Close()

			decoder := yaml.NewDecoder(file)
			var config Config
			if err := decoder.Decode(&config); err != nil {
				return nil, err
			}
			return &config, nil
		}
		return nil, errors.Wrapf(err, "config file name is wrong")
	}
	return nil, errors.Wrapf(err, "config path %s not found", configPath)
}
