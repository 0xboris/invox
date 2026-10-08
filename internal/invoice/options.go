package invoice

import (
	"os"
	"path/filepath"
)

type Options struct {
	BaseDir           string
	CustomersPath     string
	IssuerPath        string
	DefaultsPath      string
	InvoicePath       string
	PDFPath           string
	TemplatePath      string
	OutputPath        string
	OverwriteOutput   bool
	EmailTo           string
	EmailSubject      string
	ArchiveAfterBuild bool
	AssumeYes         bool
	FromLastInvoice   bool
	EditNewInvoice    bool
}

// NormalizeOptions makes every path in opts absolute, resolving relative
// paths against opts.BaseDir, the working directory the CLI was given.
func NormalizeOptions(opts *Options) {
	opts.BaseDir = filepath.Clean(opts.BaseDir)
	for _, path := range []*string{
		&opts.CustomersPath,
		&opts.IssuerPath,
		&opts.DefaultsPath,
		&opts.InvoicePath,
		&opts.PDFPath,
		&opts.TemplatePath,
		&opts.OutputPath,
	} {
		if *path != "" {
			*path = absPath(opts.BaseDir, *path)
		}
	}
}

// absPath is filepath.Abs with base in place of the process working
// directory.
func absPath(base, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	if path != "" && os.IsPathSeparator(path[0]) {
		// A rooted path without a volume, such as \x on Windows, stays on
		// base's drive, as filepath.Abs keeps it on the current drive.
		return filepath.Join(filepath.VolumeName(base), path)
	}
	return filepath.Join(base, path)
}
