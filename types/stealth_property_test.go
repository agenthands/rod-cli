package types

import (
	"fmt"
	"testing"

	"pgregory.net/rapid"
)

// uaOSTable maps a literal UA OS token to the navigator.platform real Chrome
// reports for it.
var uaOSTable = []struct{ token, platform string }{
	{"Windows NT 10.0; Win64; x64", "Win32"},
	{"Macintosh; Intel Mac OS X 10_15_7", "MacIntel"},
	{"X11; Linux x86_64", "Linux"},
}

// TestResolveStealthPlatformFollowsFlagUA pins types:T1: whatever profile is
// selected, a --user-agent flag decides the platform, so a foreign UA never
// ships with the profile's platform.
func TestResolveStealthPlatformFollowsFlagUA(t *testing.T) {
	profiles := append([]string{""}, BuiltinProfileNames()...)
	rapid.Check(t, func(t *rapid.T) {
		profile := rapid.SampledFrom(profiles).Draw(t, "profile")
		os := rapid.SampledFrom(uaOSTable).Draw(t, "os")
		major := rapid.IntRange(90, 199).Draw(t, "major")
		ua := fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%d.0.0.0 Safari/537.36", os.token, major)

		cfg, err := defaultConfigCopy()
		if err != nil {
			t.Fatalf("defaults: %v", err)
		}
		err = ResolveStealth(cfg, &StealthFlags{Profile: profile, UserAgent: ua})
		if profile == "" && err != nil {
			t.Fatalf("no profile, UA %q: unexpected error %v", ua, err)
		}
		if err == nil && cfg.Stealth.Platform != os.platform {
			t.Fatalf("profile %q, UA %q: platform = %q, want %q", profile, ua, cfg.Stealth.Platform, os.platform)
		}
	})
}
