package billing_test

import (
	"testing"
	"time"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
)

// configBrokenIssuer is a Directory whose Issuer fails on config.yaml, as
// one that reads the config file only when it looks up issuer.yaml would.
type configBrokenIssuer struct {
	billing.Directory
	err error
}

func (d configBrokenIssuer) Issuer() (invoice.Issuer, error) { return invoice.Issuer{}, d.err }

// A config problem is reported on its own, never as a problem of the file
// being read, which validation would go on to check field by field.
func TestValidateStopsOnAConfigDecodeError(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	svc := isolatedHost(t).service(t, cmdutil.Files{Customers: customersPath, Issuer: issuerPath}, t.TempDir(), time.Time{})
	configErr := &billing.ConfigError{Err: &billing.DecodeError{File: "config.yaml", Line: 2, Path: "numbering.start", Problem: "expected an integer, got `x`"}}
	svc.Directory = configBrokenIssuer{Directory: svc.Directory, err: configErr}

	_, err := svc.Validate(invoicePath)

	if err != error(configErr) {
		t.Fatalf("Validate error = %q, want only %q", err, configErr)
	}
}
