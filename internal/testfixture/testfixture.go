// Package testfixture writes the files tests run invox on: customers,
// issuer, invoices, defaults, templates and config. The files live under
// testdata; each writer copies them into a temporary directory of the test.
// It imports only the standard library, so the tests of every package can
// use it.
package testfixture

import (
	"embed"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// T is the part of testing.TB the writers use. testfixture does not import
// testing, which would pull flag parsing into a package outside the CLI.
type T interface {
	Helper()
	TempDir() string
	Cleanup(func())
	Fatalf(format string, args ...any)
}

//go:embed testdata
var files embed.FS

// Source returns the fixture file name, a path below testdata. A checkout
// with CRLF line endings reads the same as one with LF.
func Source(name string) string {
	data, err := files.ReadFile(path.Join("testdata", name))
	if err != nil {
		panic(err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

// Context is a customer, an issuer, an invoice for them and a template with
// a logo and a font next to it, all in one directory.
type Context struct {
	Dir       string
	Customers string
	Issuer    string
	Invoice   string
	Template  string
	Logo      string
	Font      string
}

// WriteContext writes testdata/context to a new temporary directory.
func WriteContext(t T) Context {
	t.Helper()
	dir := copyDir(t, "context")
	return Context{
		Dir:       dir,
		Customers: filepath.Join(dir, "customers.yaml"),
		Issuer:    filepath.Join(dir, "issuer.yaml"),
		Invoice:   filepath.Join(dir, "invoice.yaml"),
		Template:  filepath.Join(dir, "invoice_template.tex"),
		Logo:      filepath.Join(dir, "logo.png"),
		Font:      filepath.Join(dir, "fonts", "Ubuntu-Regular.ttf"),
	}
}

// WriteBuiltContext is WriteContext with the invoice built: Invoice is
// BL00210001.yaml, the context's invoice with status built, next to its PDF
// BL00210001.pdf in a directory of its own.
func WriteBuiltContext(t T) Context {
	t.Helper()
	fx := WriteContext(t)
	dir := t.TempDir()
	fx.Invoice = filepath.Join(dir, "BL00210001.yaml")
	WriteContextInvoice(t, fx.Invoice, "built")
	WriteFile(t, filepath.Join(dir, "BL00210001.pdf"), "%PDF-1.4\nfake")
	return fx
}

// WriteContextInvoice writes the context's invoice to path with
// invoice.status set to status.
func WriteContextInvoice(t T, path, status string) {
	t.Helper()
	WriteFile(t, path, strings.Replace(Source("context/invoice.yaml"), "  paid_amount: 0\n", "  paid_amount: 0\n  status: "+status+"\n", 1))
}

// Draft is what new drafts an invoice from: a customer with a name only,
// an issuer with payment terms only, and invoice defaults.
type Draft struct {
	Customers string
	Issuer    string
	Defaults  string
}

// WriteDraft writes testdata/draft to a new temporary directory.
func WriteDraft(t T) Draft {
	t.Helper()
	dir := copyDir(t, "draft")
	return Draft{
		Customers: filepath.Join(dir, "customers.yaml"),
		Issuer:    filepath.Join(dir, "issuer.yaml"),
		Defaults:  filepath.Join(dir, "invoice_defaults.yaml"),
	}
}

// WriteDraftWithStart is WriteDraft with the customer's numbering.start set
// to start.
func WriteDraftWithStart(t T, start string) Draft {
	t.Helper()
	draft := WriteDraft(t)
	WriteFile(t, draft.Customers, Source("draft/customers.yaml")+"  numbering:\n    start: "+start+"\n")
	return draft
}

// WriteArchivedInvoice writes dir/name, an archived invoice that has only a
// number, and returns its path.
func WriteArchivedInvoice(t T, dir, name, number string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	WriteFile(t, path, "invoice:\n  number: "+number+"\n")
	return path
}

// WriteNumberedInvoice writes dir/name, an invoice of CUST-001 with number,
// status and one position, and returns its path.
func WriteNumberedInvoice(t T, dir, name, number, status string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	WriteFile(t, path, strings.Join([]string{
		"customer_id: CUST-001",
		"invoice:",
		"  number: " + number,
		"  issue_date: \"2026-03-06\"",
		"  due_date: \"2026-04-05\"",
		"  status: " + status,
		"  vat_percent: 20",
		"  paid_amount: 0",
		"positions:",
		"  - name: Development",
		"    unit_price: 100",
		"    quantity: 1",
		"",
	}, "\n"))
	return path
}

// NumberedInvoiceSource is an invoice of customerID that has only a number
// and an issue date.
func NumberedInvoiceSource(customerID, number, issueDate string) string {
	return "customer_id: " + customerID + "\ninvoice:\n  number: " + number + "\n  issue_date: \"" + issueDate + "\"\n"
}

// Host is a user's config home and home directory: where invox finds
// config.yaml and, by default, the archive.
type Host struct {
	ConfigHome string
	Home       string
}

// NewHost returns a Host whose directories are under a new temporary
// directory, so a test never reads the developer's config or archive.
func NewHost(t T) Host {
	t.Helper()
	root := t.TempDir()
	return Host{ConfigHome: filepath.Join(root, "config-home"), Home: filepath.Join(root, "home")}
}

// HostWithConfig returns a new Host whose config.yaml is source.
func HostWithConfig(t T, source string) Host {
	t.Helper()
	h := NewHost(t)
	h.WriteConfig(t, source)
	return h
}

// ConfigDir is the invox config directory of h.
func (h Host) ConfigDir() string {
	return filepath.Join(h.ConfigHome, "invox")
}

// WriteConfig writes source as h's config.yaml and returns its path.
func (h Host) WriteConfig(t T, source string) string {
	t.Helper()
	path := filepath.Join(h.ConfigDir(), "config.yaml")
	WriteFile(t, path, source)
	return path
}

// WriteFile writes content to path, creating its directory.
func WriteFile(t T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) returned error: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", path, err)
	}
}

// ReadFile returns the content of path.
func ReadFile(t T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) returned error: %v", path, err)
	}
	return string(data)
}

// QuoteYAML single-quotes value for YAML, where backslashes, as in Windows
// paths, are literal.
func QuoteYAML(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// Executable returns the path of the running test binary, which tests start
// again as a child process.
func Executable(t T) string {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() returned error: %v", err)
	}
	return self
}

// copyDir copies testdata/name to a new temporary directory and returns it.
func copyDir(t T, name string) string {
	t.Helper()
	dir := t.TempDir()
	root := path.Join("testdata", name)
	err := fs.WalkDir(files, root, func(p string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel := strings.TrimPrefix(p, root+"/")
		WriteFile(t, filepath.Join(dir, filepath.FromSlash(rel)), Source(path.Join(name, rel)))
		return nil
	})
	if err != nil {
		t.Fatalf("copy %s: %v", root, err)
	}
	return dir
}
