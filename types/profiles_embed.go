package types

import (
	"embed"
	"encoding/json"
	"sort"
	"strings"
	"sync"

	"github.com/agenthands/godoll/stealth"
	"github.com/pkg/errors"
)

// builtinProfilesFS embeds the vetted, Chrome-only desktop profile library that
// ships inside the binary (PROF-01). Each file is a godoll stealth.Profile JSON.
// The embed site lives in package types (next to resolveProfilePath/ResolveStealth)
// so there is exactly ONE embed site and the built-in lookup is co-located with
// the profile-resolution funnel that consumes it.
//
//go:embed profiles/*.json
var builtinProfilesFS embed.FS

// builtinProfilesDir is the directory the profiles are embedded under (relative to
// this Go file's package directory).
const builtinProfilesDir = "profiles"

// builtinProfileNames is the sorted set of built-in profile names (file stems with
// the .json suffix stripped), computed once from the embedded FS at first use.
var (
	builtinProfileNamesOnce sync.Once
	builtinProfileNames     []string
)

// loadBuiltinProfileNames reads the embedded profiles dir once and caches the
// sorted name set. The embedded FS ships in the binary, so a read failure here is
// a build/packaging defect, not a runtime condition — but we degrade to an empty
// set rather than panic at init (callers treat an unknown name as "not a built-in"
// and fall back to the user-dir path, preserving the loud-failure discipline for a
// genuinely missing profile).
func loadBuiltinProfileNames() {
	builtinProfileNamesOnce.Do(func() {
		entries, err := builtinProfilesFS.ReadDir(builtinProfilesDir)
		if err != nil {
			return
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasSuffix(name, ".json") {
				continue
			}
			names = append(names, strings.TrimSuffix(name, ".json"))
		}
		sort.Strings(names)
		builtinProfileNames = names
	})
}

// BuiltinProfileNames returns the sorted list of built-in profile names available
// for `--profile=<name>` selection and `--profile=list` discovery. The returned
// slice is a copy so callers cannot mutate the cached set.
func BuiltinProfileNames() []string {
	loadBuiltinProfileNames()
	out := make([]string, len(builtinProfileNames))
	copy(out, builtinProfileNames)
	return out
}

// isBuiltinProfile reports whether name is one of the embedded built-in profiles.
func isBuiltinProfile(name string) bool {
	loadBuiltinProfileNames()
	for _, n := range builtinProfileNames {
		if n == name {
			return true
		}
	}
	return false
}

// LoadBuiltinProfile resolves a bare profile name to an embedded built-in
// stealth.Profile. It returns:
//
//   - (profile, true, nil)  when name is a built-in and parses cleanly;
//   - (nil, false, nil)     when name is NOT a built-in (so the caller falls back
//     to the existing ~/.rod-cli/profiles/<name>.json user-dir path);
//   - (nil, true, err)      when name IS a built-in but its embedded JSON is
//     malformed — a hard error, because a built-in ships in the binary and must
//     never be unparseable (it would mean a corrupt build).
func LoadBuiltinProfile(name string) (*stealth.Profile, bool, error) {
	if !isBuiltinProfile(name) {
		return nil, false, nil
	}
	data, err := builtinProfilesFS.ReadFile(builtinProfilesDir + "/" + name + ".json")
	if err != nil {
		return nil, true, errors.Wrapf(err, "read built-in profile %q", name)
	}
	var prof stealth.Profile
	if err := json.Unmarshal(data, &prof); err != nil {
		return nil, true, errors.Wrapf(err, "parse built-in profile %q (corrupt build)", name)
	}
	return &prof, true, nil
}
