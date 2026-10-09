package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// userDirEnvKeys are the variables invox derives the config directory and the
// default archive directory from on Linux, macOS and Windows.
var userDirEnvKeys = []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "APPDATA", "HOME", "USERPROFILE"}

// processUserDirEnv holds the values the test process started with, so
// isolateUserDirs can tell which variables a test has set itself.
var processUserDirEnv = func() map[string]string {
	values := make(map[string]string, len(userDirEnvKeys))
	for _, key := range userDirEnvKeys {
		values[key] = os.Getenv(key)
	}
	return values
}()

// isolateUserDirs points every config- and data-directory variable the test
// has not set itself at a temporary directory, so commands never read the
// developer's real config or archive.
func isolateUserDirs(t *testing.T) {
	t.Helper()

	var root string
	for _, key := range userDirEnvKeys {
		if os.Getenv(key) != processUserDirEnv[key] {
			continue
		}
		if root == "" {
			root = t.TempDir()
		}
		t.Setenv(key, filepath.Join(root, strings.ToLower(key)))
	}
	if value := os.Getenv("INVOX_CONFIG_DIR"); value != "" && value == processInvoxConfigDir {
		t.Setenv("INVOX_CONFIG_DIR", "")
	}
}

// processInvoxConfigDir is INVOX_CONFIG_DIR as the test process started, so
// isolateUserDirs can clear a developer's own setting.
var processInvoxConfigDir = os.Getenv("INVOX_CONFIG_DIR")
