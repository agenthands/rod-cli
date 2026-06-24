package types

import (
	"github.com/agenthands/godoll/stealth"
	"github.com/agenthands/rod-cli/utils"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
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
// Phases 27–28 still reserve their fields:
//
//	Reserved for Phase 27 (canvas/WebGL/WebRTC hardening):
//	  WebRTC (leak-protection toggle), CanvasNoise.
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
		if _, err := stealth.LoadProfile(path); err != nil {
			return errors.Wrapf(err, "load stealth profile %q", path)
		}
		cfg.Stealth.ProfilePath = path
		// Identity fields from the loaded profile are reserved for Phase 26 — this
		// phase only tracks that a profile was selected (ProfilePath). The proxy
		// is not a profile field, so nothing else is overlaid here.
	}

	// Tier 1: CLI flags win over the profile and the defaults.
	if flags.Proxy != "" {
		cfg.Stealth.Proxy = flags.Proxy
	}
	if flags.ProxyAuth != "" {
		cfg.Stealth.ProxyAuth = flags.ProxyAuth
	}

	// Bridge the deprecated compatibility shim.
	cfg.Proxy = cfg.Stealth.Proxy
	return nil
}

var (
	DefaultBrowserTempDir = "./rod/browser"

	DefaultConfig = Config{
		BrowserBinPath: "",
		Headless:       false,
		BrowserTempDir: DefaultBrowserTempDir,
		NoSandbox:      false,
		Proxy:          "",
		Stealth:        StealthConfig{},
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
