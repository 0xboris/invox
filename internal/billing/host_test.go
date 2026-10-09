package billing_test

import (
	"testing"
	"time"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/adapters/run/runtest"
	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/store"
	"github.com/0xboris/invox/internal/testfixture"
)

// service returns the use cases as invox wires them for h, with the
// support files named as on a command line, run in workDir at now. The
// zero now is factorytest's clock.
func service(t *testing.T, h testfixture.Host, files cmdutil.Files, workDir string, now time.Time) *billing.Service {
	t.Helper()
	return factorytest.New(t, nil, factorytest.Options{
		Home:  h.Home,
		Vars:  map[string]string{"XDG_CONFIG_HOME": h.ConfigHome},
		Getwd: func() (string, error) { return workDir, nil },
		Now:   now,
	}).Service(files)
}

// mailService is service with a runner whose opener records the drafts it
// is asked to open instead of starting the mail app.
func mailService(t *testing.T, h testfixture.Host, files cmdutil.Files, workDir string, now time.Time) (*billing.Service, *[]string) {
	t.Helper()
	stub := runtest.NewStub(t)
	opened := new([]string)
	stub.Register("xdg-open", func(cmd run.Cmd) error {
		*opened = append(*opened, cmd.Args[0])
		return nil
	})
	return factorytest.New(t, nil, factorytest.Options{
		Home:   h.Home,
		Vars:   map[string]string{"XDG_CONFIG_HOME": h.ConfigHome},
		Getwd:  func() (string, error) { return workDir, nil },
		Now:    now,
		Runner: stub,
	}).Service(files), opened
}

// setInvoiceNumber sets invoice.number in the invoice at path, as editing
// the file would.
func setInvoiceNumber(path, number string) error {
	return (&store.Store{}).Update(path, func(inv *invoice.Invoice) error {
		inv.Header.Number = invoice.Text(number)
		return nil
	})
}
