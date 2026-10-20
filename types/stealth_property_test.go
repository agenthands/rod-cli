package types

import (
	"fmt"
	"strings"
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

// TestResolveStealthConsistencyGate pins types:T2: each reject and derive
// branch of the fingerprint gate, through ResolveStealth.
func TestResolveStealthConsistencyGate(t *testing.T) {
	const (
		macUA   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"
		winUA   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"
		linuxUA = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"
		bareUA  = "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"
	)
	for _, tc := range []struct {
		name     string
		stealth  StealthConfig
		flags    StealthFlags
		wantErr  string
		wantPlat string
		wantLoc  string
		w, h     int // screen geometry
	}{
		{name: "mac UA vs --platform Win32", flags: StealthFlags{UserAgent: macUA, Platform: "Win32"}, wantErr: "contradicts UA OS"},
		{name: "no OS token, Solaris", flags: StealthFlags{UserAgent: bareUA, Platform: "SunOS"}, wantErr: "could not be verified"},
		{name: "no OS token, Linux", flags: StealthFlags{UserAgent: bareUA, Platform: "Linux"}, wantPlat: "Linux"},
		{name: "locale fr-FR vs en-US", stealth: StealthConfig{Languages: []string{"en-US"}}, flags: StealthFlags{UserAgent: winUA, Locale: "fr-FR"}, wantErr: "contradicts navigator.languages[0]"},
		{name: "locale en_GB vs en-US", stealth: StealthConfig{Languages: []string{"en-US"}}, flags: StealthFlags{UserAgent: winUA, Locale: "en_GB"}, wantPlat: "Win32", wantLoc: "en_GB"},
		{name: "no locale derives en-US", stealth: StealthConfig{Languages: []string{"en-US"}}, flags: StealthFlags{UserAgent: winUA}, wantPlat: "Win32", wantLoc: "en-US"},
		{name: "linux UA derives platform", flags: StealthFlags{UserAgent: linuxUA}, wantPlat: "Linux"},
		{name: "no UA, half screen", w: 800, wantErr: "incomplete screen geometry"},
		{name: "no UA, deviceMemory 16", stealth: StealthConfig{DeviceMemory: 16}, wantErr: "implausible deviceMemory"},
		{name: "UA, negative screen", w: -1, h: 600, flags: StealthFlags{UserAgent: winUA}, wantErr: "implausible screen geometry"},
		{name: "UA, hardwareConcurrency 512", stealth: StealthConfig{HardwareConcurrency: 512}, flags: StealthFlags{UserAgent: winUA}, wantErr: "implausible hardwareConcurrency"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{Stealth: tc.stealth}
			cfg.Stealth.Screen.Width, cfg.Stealth.Screen.Height = tc.w, tc.h
			err := ResolveStealth(cfg, &tc.flags)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want one containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Stealth.Platform != tc.wantPlat || (tc.wantLoc != "" && cfg.Stealth.Locale != tc.wantLoc) {
				t.Fatalf("platform=%q locale=%q, want %q %q", cfg.Stealth.Platform, cfg.Stealth.Locale, tc.wantPlat, tc.wantLoc)
			}
		})
	}
}

// TestResolveStealthRejectsUnsafeFingerprintValue pins types:T3: one bad rune
// in any fingerprint string field fails the gate, naming that field.
func TestResolveStealthRejectsUnsafeFingerprintValue(t *testing.T) {
	const clean = `[ -!#-\[\]-~]{0,40}`
	bad := []rune{'"', '\\', 0x7f}
	for r := rune(0); r <= 0x1f; r++ {
		bad = append(bad, r)
	}
	fields := []string{"--user-agent", "--timezone", "--locale", "--platform", "vendor", "acceptLanguage", "languages"}

	t.Run("bad rune", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			s := rapid.StringMatching(clean).Draw(t, "clean")
			at := rapid.IntRange(0, len(s)).Draw(t, "at")
			val := s[:at] + string(rapid.SampledFrom(bad).Draw(t, "bad")) + s[at:]
			field := rapid.SampledFrom(fields).Draw(t, "field")

			cfg, flags, label := &Config{}, &StealthFlags{}, field
			switch field {
			case "--user-agent":
				flags.UserAgent = val
			case "--timezone":
				flags.Timezone = val
			case "--locale":
				flags.Locale = val
			case "--platform":
				flags.Platform = val
			case "vendor":
				cfg.Stealth.Vendor = val
			case "acceptLanguage":
				cfg.Stealth.AcceptLanguage = val
			case "languages":
				k := rapid.IntRange(0, 2).Draw(t, "k")
				cfg.Stealth.Languages = []string{"en-US", "en", "de"}
				cfg.Stealth.Languages[k] = val
				label = fmt.Sprintf("languages[%d]", k)
			}
			err := ResolveStealth(cfg, flags)
			if err == nil || !strings.HasPrefix(err.Error(), "fingerprint field "+label+" ") {
				t.Fatalf("%s = %q: err = %v, want prefix %q", label, val, err, "fingerprint field "+label+" ")
			}
		})
	})

	t.Run("clean timezone", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			tz := rapid.StringMatching(clean).Draw(t, "tz")
			if err := ResolveStealth(&Config{}, &StealthFlags{Timezone: tz}); err != nil {
				t.Fatalf("timezone %q: unexpected error %v", tz, err)
			}
		})
	})
}
