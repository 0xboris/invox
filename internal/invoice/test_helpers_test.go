package invoice

import (
	"path/filepath"
	"testing"
)

func chdirForTest(t *testing.T, dir string) {
	t.Helper()
	t.Chdir(dir)
}

// testHost returns a Linux Host whose config directory is under configHome
// and whose archive directory is under home/.local/share.
func testHost(configHome, home string) Host {
	return NewHost(HostInputs{GOOS: "linux", Home: home, XDGConfigHome: configHome})
}

// isolatedHost returns a Host whose user directories are all under a fresh
// temporary directory, so a test never reads the developer's config or archive.
func isolatedHost(t *testing.T) Host {
	t.Helper()
	root := t.TempDir()
	return testHost(filepath.Join(root, "config-home"), filepath.Join(root, "home"))
}
