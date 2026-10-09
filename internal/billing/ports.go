package billing

import (
	"context"
	"time"

	"github.com/0xboris/invox/internal/invoice"
)

// File is a file invox reads besides the invoice.
type File int

const (
	CustomersFile File = iota
	IssuerFile
	DefaultsFile
	TemplateFile
	ConfigFile
)

func (f File) String() string {
	return [...]string{"customers", "issuer", "defaults", "template", "config"}[f]
}

// Check says how Create checks the invoice it writes.
type Check int

const (
	// CheckStrict rejects keys an invoice does not have.
	CheckStrict Check = iota
	// CheckLenient keeps them, as an archived invoice is a record.
	CheckLenient
	// CheckNone checks nothing.
	CheckNone
)

// CreateOptions control Invoices.Create.
type CreateOptions struct {
	// Dir holds the new invoice, named after its number, when the path is
	// "".
	Dir string
	// Overwrite replaces an existing file, but never an archived invoice:
	// that is an *ArchivedOutputError.
	Overwrite bool
	// DryRun runs every check and writes nothing.
	DryRun bool
	Check  Check
}

// Invoices reads and writes invoice files. Update keeps the file's comments
// and layout; TestInvoiceWritesKeepComments pins that.
type Invoices interface {
	// Load decodes the invoice at path strictly. Values that do not fit
	// come back as joined *DecodeError values, with the rest decoded.
	Load(path string) (invoice.Invoice, error)
	// Drafts returns the customer and the header's number and status of
	// the invoices directly in workDir and, when output is set, in the
	// directory output is in, best effort: files that cannot be read are
	// left out.
	Drafts(workDir, output string) []invoice.Invoice
	// Create writes a new invoice to path, or when path is "", to
	// <number>.yaml in opts.Dir, from the document at from, which may be
	// an archived invoice, keeping its comments and keys, and returns the
	// file. Every customer and header field set in inv is written, as
	// text. Positions is added when inv's is not nil and from has none.
	// The `_invox` link is replaced by inv's. It returns an
	// *OutputIsDirError when the file is a directory and, unless
	// opts.Overwrite is set, an *OutputExistsError when it exists.
	Create(path, from string, inv invoice.Invoice, opts CreateOptions) (string, error)
	// Update rewrites the invoice at path with change applied, writing
	// back only the fields that changed.
	Update(path string, change func(*invoice.Invoice) error) error
}

// CustomerEntry is one customer of customers.yaml. Each entry is decoded
// on its own, so a problem in one never stops another customer's invoice.
type CustomerEntry struct {
	ID       string
	Customer invoice.Customer
	// Err is what in the entry did not decode, as *DecodeError values,
	// with the rest of Customer decoded. A key a customer does not have
	// is one, with UnknownKey set.
	Err error
}

// Template is a LaTeX template.
type Template struct {
	Name string
	Path string
}

// Source says where a resolved path came from.
type Source int

const (
	SourceNone     Source = iota // nothing found
	SourceExplicit               // the config file the user named
	SourceEnvDir                 // the config directory the user chose, or a file in it
	SourceDefault                // the OS default directory, or a file in it
	SourceProject                // found by the upward search from the working directory
	SourceConfig                 // a paths.* or archive.dir setting in the config file
)

// PathReport is one line of `config paths`.
type PathReport struct {
	Name   string
	Path   string // "" only with SourceNone
	Source Source
}

// InitFile is a file Init made sure exists.
type InitFile struct {
	Path    string
	Created bool
}

// Directory answers which customers, issuer, defaults and template apply.
type Directory interface {
	// Locate returns the support file f: the one named for this run, else
	// the one the project search, config.yaml or the config directory
	// finds. Finding none is a *FileNotFoundError.
	Locate(f File) (string, error)
	Customer(id string) (invoice.Customer, error)
	// Customers decodes every entry of customers.yaml, sorted by ID.
	Customers() ([]CustomerEntry, error)
	// Issuer decodes issuer.yaml strictly; values that do not fit come
	// back as *DecodeError values.
	Issuer() (invoice.Issuer, error)
	// Defaults returns invoice_defaults.yaml.
	Defaults() (string, error)
	// Template resolves ref, a path or a template name, or the default
	// template when ref is "". A ref that resolves to nothing is a
	// *TemplateLookupError.
	Template(ref string) (Template, error)
	// Templates lists the template catalog and returns its directory.
	Templates() ([]Template, string, error)
	// Paths reports where each file comes from for this run.
	Paths() ([]PathReport, error)
	// EditablePath returns f for an editor, creating config.yaml from its
	// template when f is ConfigFile and it does not exist.
	EditablePath(f File) (string, error)
	// Init creates the config directory and the starter files it lacks,
	// and returns the directory.
	Init() (string, []InitFile, error)
}

// ArchiveEntry is an archived invoice: where it is and what it says about
// itself.
type ArchiveEntry struct {
	// Path is the file, below the archive directory as configured.
	Path string
	// Filename is Path relative to the archive directory.
	Filename   string
	CustomerID string
	IssueDate  string
	// Status is `archived` when the file has none.
	Status invoice.Status
	Number string
}

// Unread is what the archive holds that invox no longer reads.
type Unread struct {
	// Dir is the archive directory, "" when there is none.
	Dir string
	// Markdown are the archived invoices stored as Markdown, sorted.
	Markdown []string
}

// Backup is an archived file that re-archiving replaced, and where its
// previous version was kept.
type Backup struct {
	Path       string
	BackupPath string
}

// ArchiveResult describes what Archive wrote.
type ArchiveResult struct {
	// Path is the archived invoice.
	Path string
	// Replaced lists the archived files the invoice replaced.
	Replaced []Backup
	// HistoryDir is where the replaced files' previous versions are kept.
	HistoryDir string
	// Unread is what the archive walk of the duplicate check could not
	// read.
	Unread Unread
}

// AddOptions control Archive.Add.
type AddOptions struct {
	// Replace allows writing over archived files. Without it, Add returns
	// an *ArchiveReplaceError when it would replace any.
	Replace bool
	// DryRun runs every check and returns the result without writing. The
	// result's backups have no BackupPath.
	DryRun bool
	// Change is applied to the invoice as it is archived, keeping its
	// comments and layout, so the archived file is written once.
	Change func(*invoice.Invoice) error
}

// Checkout is where the working copy of an archived invoice goes.
type Checkout struct {
	// Archived is the archived invoice.
	Archived string
	// Path is its working copy.
	Path string
	// Link is what the working copy records about where it goes back to.
	Link invoice.ArchiveLink
}

// Archive holds finished invoices.
type Archive interface {
	// Entries reads every archived invoice, sorted by Filename, and
	// returns what the archive holds that invox no longer reads.
	Entries() ([]ArchiveEntry, Unread, error)
	// Duplicate returns the archived invoice, in file name order, that has
	// inv's number, other than src and, for a working copy, the archived
	// file it replaces. It returns "" when there is none, when inv has no
	// number, or when there is no archive directory, and what its walk
	// could not read.
	Duplicate(src string, inv invoice.Invoice) (string, Unread, error)
	// Add moves inv, the invoice at src, into the archive with opts.Change
	// applied, in one write: over the archived file a working copy names,
	// after backing it up, else under src's name, which must not exist
	// yet. It refuses src when it is that file already, and an archive
	// directory that is not configured. Without opts.Replace it refuses to
	// replace an archived file.
	Add(src string, inv invoice.Invoice, opts AddOptions) (ArchiveResult, error)
	// Checkout resolves ref, an archived invoice relative to the archive
	// directory, and says where its working copy in workDir goes.
	Checkout(ref, workDir string) (Checkout, error)
	// Source returns the invoice YAML file the PDF at pdf was built from:
	// the one with its name next to it, else the one in the archive. It
	// returns "" when there is none, and an error when the archive has
	// several, with what a walk of the archive could not read.
	Source(pdf string) (string, Unread, error)
}

// EPC is what a template's EPC QR code placeholders need.
type EPC struct {
	// Payload is the code's content; nil when the invoice is not paid by
	// EPC transfer or Err is set.
	Payload []byte
	// Err is why an invoice that qualifies has no valid payload.
	Err error
	// Label is the text shown with the code.
	Label string
}

// Renderer turns an invoice into a document: source to write, or a PDF.
type Renderer interface {
	// Render checks template t and fills it in. It writes nothing.
	Render(t Template, inv *invoice.Context, epc EPC) (string, error)
	// Write writes source to path and copies t's assets next to it.
	Write(t Template, source, path string) error
	// Build compiles source, with t's assets, into a PDF at output.
	Build(ctx context.Context, t Template, source, output string) error
}

// Message is an invoice email with the PDF to attach.
type Message struct {
	To          string
	Subject     string
	Body        string
	FromName    string
	FromAddress string
	Attachment  string
	// Output is the .eml file to write. A temporary draft takes only its
	// name and goes to a new temporary directory.
	Output    string
	Temporary bool
	// Overwrite replaces an existing Output.
	Overwrite bool
	Date      time.Time
}

// Mailer drafts an email with the PDF attached.
type Mailer interface {
	// Draft checks that m's attachment can be read and, for a kept draft,
	// that its file can be written, then drafts m and opens it unless
	// dryRun is set. It returns the draft's file, "" when a mail app holds
	// the draft or in a dry run.
	Draft(ctx context.Context, m Message, dryRun bool) (string, error)
}
