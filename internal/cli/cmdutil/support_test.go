package cmdutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/store"
)

func TestSupportPath(t *testing.T) {
	work := t.TempDir()
	configDir := t.TempDir()
	host := store.NewHost(store.HostInputs{GOOS: "linux", Home: t.TempDir(), ConfigDir: configDir})

	got, err := SupportPath(host, "customer list", store.Customers, filepath.Join("sub", "c.yaml"), work)
	if err != nil || got != filepath.Join(work, "sub", "c.yaml") {
		t.Errorf("flag value: got %q, %v; want %q", got, err, filepath.Join(work, "sub", "c.yaml"))
	}

	_, err = SupportPath(host, "customer list", store.Customers, "", work)
	var flagErr *FlagError
	want := "customers file not found; pass -c/--customers, set paths.customers in config.yaml, or place customers.yaml at " + filepath.Join(configDir, "customers.yaml")
	if !errors.As(err, &flagErr) || flagErr.Command != "customer list" || err.Error() != want {
		t.Errorf("nothing found: got %#v, want FlagError for customer list: %q", err, want)
	}

	if err := os.WriteFile(filepath.Join(configDir, "issuer.yaml"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = SupportPath(host, "validate", store.Issuer, " ", work)
	if err != nil || got != filepath.Join(configDir, "issuer.yaml") {
		t.Errorf("found in config dir: got %q, %v; want %q", got, err, filepath.Join(configDir, "issuer.yaml"))
	}
}
