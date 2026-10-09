package archive_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/factory/factorytest"
)

// host is a user's config home and home directory.
type host struct {
	configHome string
	home       string
}

// writeConfigFile returns a host under fresh temporary directories whose
// config.yaml is source.
func writeConfigFile(t *testing.T, source string) host {
	t.Helper()

	h := host{configHome: filepath.Join(t.TempDir(), "config-home"), home: filepath.Join(t.TempDir(), "home")}
	configDir := filepath.Join(h.configHome, "invox")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}
	return h
}

// service returns the use cases as invox wires them for h, so its
// Archives is the archive of h's config.
func (h host) service(t *testing.T) *billing.Service {
	t.Helper()
	return factorytest.New(t, nil, factorytest.Options{
		Home: h.home,
		Vars: map[string]string{"XDG_CONFIG_HOME": h.configHome},
	}).Service(cmdutil.Files{})
}

func quoteYAMLString(value string) string {
	return strconv.Quote(value)
}
