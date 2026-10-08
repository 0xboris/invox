package helptext

import (
	"io"
	"path/filepath"
	"text/template"

	"github.com/0xboris/invox/internal/store"
)

// The lines of a command's "Default lookup:" section. Like every Long, they
// are templates that Render fills in from the user's directories.
const (
	LookupCustomers = "  customers.yaml: upward project search, then {{.GlobalCustomersPath}}\n" +
		"  schema/docs: run `invox help customers`\n"
	LookupIssuer = "  issuer.yaml: upward project search, then {{.GlobalIssuerPath}}\n" +
		"  schema/docs: run `invox help issuer`\n"
	LookupDefaults = "  invoice_defaults.yaml: upward project search, then {{.GlobalInvoiceDefaultsPath}}\n" +
		"  schema/docs: run `invox help defaults`\n"
	LookupTemplate = "  template.tex: upward project search, then {{.GlobalTemplatePath}}\n"
	LookupArchive  = "  archive.dir: config.yaml, then {{.DefaultArchiveDir}}\n"
)

// ReplacingArchived is the section on replacing an archived invoice, for
// archive, and for build when withArchiveFlag is set.
func ReplacingArchived(withArchiveFlag bool) string {
	text := "Replacing an archived invoice:\n"
	if withArchiveFlag {
		text += "  An invoice with invoice.status archived keeps that status when its PDF is rebuilt.\n" +
			"  With --archive, a working copy from `invox archive edit` replaces the archived invoice it came from.\n"
	} else {
		text += "  Archiving a working copy from `invox archive edit` replaces the archived invoice it came from.\n"
	}
	return text + "  On a terminal you are asked to confirm; otherwise pass --yes. Declining exits with status 2.\n" +
		"  The previous version is kept as archive.dir/.history/<path>.<UTC timestamp>.<ext>,\n" +
		"  which archive list, numbering and the duplicate-number check ignore.\n" +
		"  --yes only answers the question; every other check still applies.\n"
}

// Render writes text, a command's Long, with its {{...}} actions filled in
// from h. They can call the methods of store.Host and of data.
func Render(w io.Writer, text string, h store.Host) error {
	tmpl, err := template.New("").Option("missingkey=error").Parse(text)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, data{h})
}

type data struct {
	store.Host
}

// LegacyConfigFile is config.yaml in the legacy directory, or "none".
func (d data) LegacyConfigFile() string {
	if dir := d.LegacyConfigDir(); dir != "" {
		return filepath.Join(dir, "config.yaml")
	}
	return "none"
}
