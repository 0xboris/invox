package billing_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/store"
)

// host is a user's config home and home directory: where invox finds
// config.yaml and, by default, the archive.
type host struct {
	configHome string
	home       string
}

func testHost(configHome, home string) host {
	return host{configHome: configHome, home: home}
}

// isolatedHost returns a host whose directories are all under a fresh
// temporary directory, so a test never reads the developer's config or
// archive.
func isolatedHost(t *testing.T) host {
	t.Helper()
	root := t.TempDir()
	return testHost(filepath.Join(root, "config-home"), filepath.Join(root, "home"))
}

// writeConfigFile returns a host under fresh temporary directories whose
// config.yaml is source.
func writeConfigFile(t *testing.T, source string) host {
	t.Helper()

	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	h := testHost(configHome, filepath.Join(t.TempDir(), "home"))

	path := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}
	return h
}

// service returns the use cases as invox wires them for h, with the
// support files named as on a command line, run in workDir at now. The
// zero now is factorytest's clock.
func (h host) service(t *testing.T, files cmdutil.Files, workDir string, now time.Time) *billing.Service {
	t.Helper()
	return factorytest.New(t, nil, factorytest.Options{
		Home:  h.home,
		Vars:  map[string]string{"XDG_CONFIG_HOME": h.configHome},
		Getwd: func() (string, error) { return workDir, nil },
		Now:   now,
	}).Service(files)
}

// loadContext loads the invoice at invoicePath with its customer and
// issuer, as validate does.
func loadContext(t *testing.T, customersPath, issuerPath, invoicePath string) (*invoice.Context, error) {
	t.Helper()
	result, err := isolatedHost(t).service(t, cmdutil.Files{Customers: customersPath, Issuer: issuerPath}, t.TempDir(), time.Time{}).Validate(invoicePath)
	return result.Context, err
}

// setInvoiceNumber sets invoice.number in the invoice at path, as editing
// the file would.
func setInvoiceNumber(path, number string) error {
	return (&store.Store{}).Update(path, func(inv *invoice.Invoice) error {
		inv.Header.Number = invoice.Text(number)
		return nil
	})
}
