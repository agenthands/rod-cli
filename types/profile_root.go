package types

import (
	"os"
	"path/filepath"
)

// DefaultProfileRoot is where per-session Chrome profiles live unless the
// config names a browserTempDir: on disk in the user cache dir
// (~/.cache/rod-cli/profiles on Linux), never under the working directory.
func DefaultProfileRoot() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "rod-cli", "profiles")
}
