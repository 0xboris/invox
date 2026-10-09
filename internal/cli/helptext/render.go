package helptext

import (
	"io"
	"text/template"
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

// Locations are where invox keeps its files by default, for help texts.
// They come from the environment only; nothing is read.
type Locations struct {
	ConfigDir  string
	ConfigFile string
	Customers  string
	Issuer     string
	Defaults   string
	Template   string
	ArchiveDir string
	// ConfigTemplate is the text a new config.yaml starts with.
	ConfigTemplate string
}

// Render writes text, a command's Long, with its {{...}} actions filled in
// from l. They can read the fields of Locations and call the methods of
// data.
func Render(w io.Writer, text string, l Locations) error {
	tmpl, err := template.New("").Option("missingkey=error").Parse(text)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, data{l})
}

type data struct {
	Locations
}

func (d data) GlobalConfigPath() string          { return d.ConfigFile }
func (d data) GlobalCustomersPath() string       { return d.Customers }
func (d data) GlobalIssuerPath() string          { return d.Issuer }
func (d data) GlobalInvoiceDefaultsPath() string { return d.Defaults }
func (d data) GlobalTemplatePath() string        { return d.Template }
func (d data) DefaultArchiveDir() string         { return d.ArchiveDir }
