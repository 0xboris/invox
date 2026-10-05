package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func chdirForTest(t *testing.T, dir string) {
	t.Helper()
	t.Chdir(dir)
}

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
}
