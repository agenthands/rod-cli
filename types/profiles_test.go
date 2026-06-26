package types

import (
	"testing"

	"github.com/agenthands/godoll/stealth"
)

// overlayProfile mirrors how ResolveStealth's Tier-2 block copies a loaded
// stealth.Profile onto cfg.Stealth, so the consistency gate validates exactly the
// identity the daemon would ship.
func overlayProfile(cfg *Config, prof *stealth.Profile) {
	s := &cfg.Stealth
	s.UserAgent = prof.UserAgent
	s.Platform = prof.Platform
	s.AcceptLanguage = prof.AcceptLanguage
	s.Languages = prof.Languages
	s.Timezone = prof.Timezone
	s.Locale = prof.Locale
	s.Vendor = prof.Vendor
	s.HardwareConcurrency = prof.HardwareConcurrency
	s.DeviceMemory = prof.DeviceMemory
	s.SpoofClientHints = prof.SpoofClientHints
	s.Screen.Width = prof.Screen.Width
	s.Screen.Height = prof.Screen.Height
	s.Screen.DeviceScaleFactor = prof.Screen.DeviceScaleFactor
}

// TestBuiltinProfilesAreVetted is the PROF-02 consistency gate: it iterates EVERY
// built-in profile and asserts each is internally coherent enough to ship. An
// incoherent profile (UA-OS-token ↔ platform, locale ↔ languages[0], implausible
// screen/hardware, or a UA with no Chrome major) FAILS the build — that is the
// "vetted" in PROF-01/02: a future edit that breaks a profile turns the build red.
//
// Coverage note on userSet: when ResolveStealth overlays a Tier-2 profile it passes
// an empty userSetFingerprint, so deriveAndValidateFingerprint would silently
// DERIVE an unset platform/locale rather than reject a contradiction. A built-in,
// however, PINS platform + locale explicitly, and a vetting gate must REJECT a
// pinned value that contradicts the UA. So this gate deliberately passes
// userSetFingerprint{Platform:true, Locale:true} — the stronger semantic — so a
// built-in whose platform disagrees with its UA OS token, or whose locale disagrees
// with languages[0], fails here. Plus belt-and-suspenders structural assertions.
func TestBuiltinProfilesAreVetted(t *testing.T) {
	names := BuiltinProfileNames()
	if len(names) < 5 || len(names) > 10 {
		t.Fatalf("PROF-01: built-in profile count %d outside the 5-10 range: %v", len(names), names)
	}

	for _, name := range names {
		name := name
		t.Run(name, func(t *testing.T) {
			prof, ok, err := LoadBuiltinProfile(name)
			if err != nil {
				t.Fatalf("LoadBuiltinProfile(%q) errored (corrupt embedded build?): %v", name, err)
			}
			if !ok {
				t.Fatalf("LoadBuiltinProfile(%q) returned ok=false for a name from BuiltinProfileNames", name)
			}

			// (1) The v1.6 consistency validator — the load-bearing gate.
			cfg := DefaultConfig
			overlayProfile(&cfg, prof)
			userSet := userSetFingerprint{Platform: true, Locale: true}
			if err := deriveAndValidateFingerprint(&cfg, userSet); err != nil {
				t.Errorf("profile %q FAILED the consistency validator: %v", name, err)
			}

			// (2) UA must carry a parseable Chrome major (the Client-Hints derivation
			// anchor). No Chrome token = no coherent CH/userAgentData downstream.
			if _, ok := parseChromeMajor(prof.UserAgent); !ok {
				t.Errorf("profile %q UA has no Chrome major token (CH cannot be derived): %q",
					name, prof.UserAgent)
			}

			// (3) Platform must MATCH the UA OS token (not merely be derivable). This
			// catches a Windows UA paired with a MacIntel platform even though both are
			// "known" platforms.
			if dp, _, ok := uaOSToPlatform(prof.UserAgent); ok {
				if prof.Platform != dp {
					t.Errorf("profile %q platform %q contradicts its UA OS token (expected %q): UA=%q",
						name, prof.Platform, dp, prof.UserAgent)
				}
			} else {
				t.Errorf("profile %q UA carries no recognized OS token: %q", name, prof.UserAgent)
			}

			// (4) navigator.languages must be pinned. Besides being a headless tell when
			// empty, this GUARANTEES the validator's locale↔languages[0] coherence branch
			// (config.go, gated on len(Languages)>0) actually engages — without this a
			// future built-in with a contradictory locale but an omitted languages array
			// would skip that check and pass "vetted" vacuously.
			if len(prof.Languages) == 0 || prof.Languages[0] == "" {
				t.Errorf("profile %q must pin navigator.languages (empty is a tell and disables the locale check)", name)
			}

			// (5) Built-ins must PIN plausible screen + hardware (a vetting bar stricter
			// than the validator, which treats 0 as "unset/derive" for the flag path).
			// A built-in shipping deviceMemory:0 / hardwareConcurrency:0 / screen{0,0} is
			// itself a tell.
			if prof.Screen.Width <= 0 || prof.Screen.Height <= 0 {
				t.Errorf("profile %q must pin a positive screen geometry, got %dx%d",
					name, prof.Screen.Width, prof.Screen.Height)
			}
			if prof.HardwareConcurrency <= 0 {
				t.Errorf("profile %q must pin hardwareConcurrency > 0, got %d", name, prof.HardwareConcurrency)
			}
			if prof.DeviceMemory <= 0 {
				t.Errorf("profile %q must pin deviceMemory > 0, got %d", name, prof.DeviceMemory)
			}

			// (6) All built-ins are real-Chrome desktop identities: Chrome vendor and
			// the v1.6 hardening toggles on (canvas/audio noise). No spoofed TLS exists
			// in the profile schema at all (real Chrome only).
			if prof.Vendor != "Google Inc." {
				t.Errorf("profile %q vendor %q is not Chrome's 'Google Inc.'", name, prof.Vendor)
			}
			if !prof.SpoofCanvas || !prof.SpoofAudioContext {
				t.Errorf("profile %q must enable canvas+audio noise (spoofCanvas=%v spoofAudioContext=%v)",
					name, prof.SpoofCanvas, prof.SpoofAudioContext)
			}
		})
	}
}

// TestBuiltinProfileResolvesBeforeUserDir proves a bare built-in name resolves to
// the embedded profile via the same loadSelectedProfile the daemon uses (built-in
// FIRST), and that a path-like value is NOT treated as a built-in (PROF-04).
func TestBuiltinProfileResolvesBeforeUserDir(t *testing.T) {
	prof, label, err := loadSelectedProfile("windows-11-chrome")
	if err != nil {
		t.Fatalf("loadSelectedProfile(builtin) errored: %v", err)
	}
	if label != "builtin:windows-11-chrome" {
		t.Errorf("built-in label = %q, want %q", label, "builtin:windows-11-chrome")
	}
	if prof.Platform != "Win32" {
		t.Errorf("resolved built-in platform = %q, want Win32", prof.Platform)
	}

	// A path-like value must bypass the built-in lookup (so a custom file named like
	// a built-in is still reachable via a path). It will fail to load (no such file),
	// which proves it took the verbatim-path branch, NOT the built-in branch.
	if _, _, err := loadSelectedProfile("./windows-11-chrome.json"); err == nil {
		t.Errorf("a path-like --profile should load verbatim (and here fail to open), not hit the built-in")
	}
}
