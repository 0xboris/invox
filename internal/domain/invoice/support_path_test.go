package invoice

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSupportPathNotFoundNamesTheRemedy(t *testing.T) {
	tests := []struct {
		kind       SupportFile
		global     func(Host) string
		wantError  string
		wantRemedy string
	}{
		{Customers, Host.GlobalCustomersPath, "customers file not found", "set paths.customers in config.yaml, or place customers.yaml at "},
		{Issuer, Host.GlobalIssuerPath, "issuer file not found", "set paths.issuer in config.yaml, or place issuer.yaml at "},
		{Defaults, Host.GlobalInvoiceDefaultsPath, "defaults file not found", "set paths.defaults in config.yaml, or place invoice_defaults.yaml at "},
		{Template, Host.GlobalTemplatePath, "template file not found", "set paths.template in config.yaml, or place template.tex at "},
	}
	for _, tc := range tests {
		t.Run(tc.wantError, func(t *testing.T) {
			h := isolatedHost(t)
			_, err := h.SupportPath(tc.kind, "", t.TempDir())

			var notFound *SupportFileNotFoundError
			if !errors.As(err, &notFound) {
				t.Fatalf("SupportPath error = %v, want *SupportFileNotFoundError", err)
			}
			global := tc.global(h)
			if notFound.Error() != tc.wantError || notFound.Remedy() != tc.wantRemedy+global {
				t.Fatalf("error = %q, remedy = %q; want %q, %q", notFound, notFound.Remedy(), tc.wantError, tc.wantRemedy+global)
			}
		})
	}
}

func TestSupportPathMakesTheGivenPathAbsolute(t *testing.T) {
	h := isolatedHost(t)
	work := t.TempDir()
	got, err := h.SupportPath(Customers, filepath.Join("sub", "c.yaml"), work)
	if err != nil || got != filepath.Join(work, "sub", "c.yaml") {
		t.Fatalf("SupportPath = %q, %v; want %q", got, err, filepath.Join(work, "sub", "c.yaml"))
	}

	found := filepath.Join(work, "issuer.yaml")
	if err := os.WriteFile(found, []byte("name: x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err = h.SupportPath(Issuer, " ", work)
	if err != nil || got != found {
		t.Fatalf("SupportPath = %q, %v; want the file found in the working directory %q", got, err, found)
	}
}
