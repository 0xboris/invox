package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func chdirForTest(t *testing.T, dir string) {
	t.Helper()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() returned error: %v", err)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) returned error: %v", dir, err)
	}
	if err := os.Chdir(absDir); err != nil {
		t.Fatalf("os.Chdir(%q) returned error: %v", absDir, err)
	}
	oldPWD, hadPWD := os.LookupEnv("PWD")
	if err := os.Setenv("PWD", absDir); err != nil {
		t.Fatalf("os.Setenv(PWD) returned error: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("restore working directory to %q: %v", oldDir, err)
		}
		if hadPWD {
			if err := os.Setenv("PWD", oldPWD); err != nil {
				t.Errorf("restore PWD to %q: %v", oldPWD, err)
			}
			return
		}
		if err := os.Unsetenv("PWD"); err != nil {
			t.Errorf("unset PWD: %v", err)
		}
	})
}

// dataDirEnvKeys are the variables invox derives the default archive
// directory from on Linux, macOS and Windows.
var dataDirEnvKeys = []string{"XDG_DATA_HOME", "APPDATA", "HOME", "USERPROFILE"}

// processDataDirEnv holds the values the test process started with, so
// isolateDataDirs can tell which variables a test has set itself.
var processDataDirEnv = func() map[string]string {
	values := make(map[string]string, len(dataDirEnvKeys))
	for _, key := range dataDirEnvKeys {
		values[key] = os.Getenv(key)
	}
	return values
}()

// isolateDataDirs points every data-directory variable the test has not set
// itself at a temporary directory, so commands never read the developer's
// real archive.
func isolateDataDirs(t *testing.T) {
	t.Helper()

	var root string
	for _, key := range dataDirEnvKeys {
		if os.Getenv(key) != processDataDirEnv[key] {
			continue
		}
		if root == "" {
			root = t.TempDir()
		}
		t.Setenv(key, filepath.Join(root, strings.ToLower(key)))
	}
}
