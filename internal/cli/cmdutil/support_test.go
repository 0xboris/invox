package cmdutil

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/billing"
)

func TestSupportPath(t *testing.T) {
	work := t.TempDir()
	if got, want := AbsFlag(work, filepath.Join("sub", "c.yaml")), filepath.Join(work, "sub", "c.yaml"); got != want {
		t.Errorf("flag value: got %q, want %q", got, want)
	}
	if got := AbsFlag(work, " "); got != "" {
		t.Errorf("blank flag value: got %q, want \"\"", got)
	}

	global := filepath.Join(t.TempDir(), "customers.yaml")
	err := UsageError("customer list", &billing.FileNotFoundError{File: billing.CustomersFile, Default: global})
	var flagErr *FlagError
	want := "customers file not found; pass -c/--customers, set paths.customers in config.yaml, or place customers.yaml at " + global
	if !errors.As(err, &flagErr) || flagErr.Command != "customer list" || err.Error() != want {
		t.Errorf("nothing found: got %#v, want FlagError for customer list: %q", err, want)
	}

	err = UsageError("render", &billing.TemplateLookupError{Err: &billing.TemplateNotFoundError{Name: "fancy.tex"}})
	want = `template "fancy.tex" not found; run 'invox template list' to see the templates`
	if !errors.As(err, &flagErr) || flagErr.Command != "render" || err.Error() != want {
		t.Errorf("unknown template: got %#v, want FlagError for render: %q", err, want)
	}

	other := errors.New("broken")
	if got := UsageError("validate", other); got != other {
		t.Errorf("other error: got %#v, want it unchanged", got)
	}
}
