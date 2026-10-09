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

// Head is what numbering and the archive read of an invoice, as written. It
// is decoded leniently, so a file with keys invox does not know, or with
// values it cannot read, still counts.
type Head struct {
	CustomerID string
	Number     string
	IssueDate  string
	Status     invoice.Status
	// HasHeader is false when the `invoice` key is missing or null.
	HasHeader bool
	// Header is what the `invoice` key holds as written. Archiving rewrites
	// that mapping in place, so an alias to a mapping is HeaderOther.
	Header HeaderShape
	// ArchivePath and ReplacePath are the `_invox` link of a working copy
	// from `archive edit`, relative to the archive, "" when it has none.
	ArchivePath string
	ReplacePath string
}

// HeaderShape is what the `invoice` key of an invoice file holds.
type HeaderShape int

const (
	HeaderMissing HeaderShape = iota // no `invoice` key
	HeaderMapping                    // a mapping
	HeaderOther                      // null, a scalar, a list or an alias
)

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
	// Overwrite replaces an existing file.
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
	// ArchivedHead is Head for an archived invoice, which may be the front
	// matter of a Markdown file. It decodes the whole invoice strictly, as
	// Load does, and returns those errors.
	ArchivedHead(path string) (Head, error)
	// Head reads what numbering and the archive need of the invoice at
	// path. Values that do not decode are left unset and reported as
	// *DecodeError values.
	Head(path string) (Head, error)
	// Drafts returns the invoices directly in workDir and, when output is
	// set, in the directory output is in, best effort: files that cannot
	// be read are left out.
	Drafts(workDir, output string) []Head
	// Destination returns the file a new invoice is written to: path, or
	// when path is "", <number>.yaml in workDir. It returns an
	// *OutputIsDirError when that is a directory and, unless overwrite is
	// set, an *OutputExistsError when it exists.
	Destination(path, workDir, number string, overwrite bool) (string, error)
	// Create writes a new invoice to path from the document at from, which
	// may be an archived invoice, keeping its comments and keys. Every
	// customer and header field set in inv is written, as text. Positions
	// is added when inv's is not nil and from has none. The `_invox` link
	// is replaced by inv's.
	Create(path, from string, inv invoice.Invoice, opts CreateOptions) error
	// Update rewrites the invoice at path with change applied, writing
	// back only the fields that changed.
	Update(path string, change func(*invoice.Invoice) error) error
	// Exists reports whether path is a file.
	Exists(path string) bool
}

// CustomerTable is customers.yaml. An entry is decoded only when it is
// looked up, so a problem in one entry never stops another customer's
// invoice.
type CustomerTable interface {
	// IDs returns the customer IDs, sorted.
	IDs() []string
	// Lookup decodes the entry of id. ok is false when there is none. A
	// strict lookup rejects keys a customer does not have.
	Lookup(id string, strict bool) (c invoice.Customer, ok bool, err error)
}

// Template is a LaTeX template.
type Template struct {
	Name string
	Path string
	// FindAsset returns the file or directory rel that the template uses:
	// next to the template, else in the config directories. It returns ""
	// when there is none.
	FindAsset func(rel string, dir bool) string
}

// Source says where a resolved path came from.
type Source int

const (
	SourceNone     Source = iota // nothing found
	SourceExplicit               // the config file the user named
	SourceEnvDir                 // the config directory the user chose, or a file in it
	SourceDefault                // the OS default directory, or a file in it
	SourceLegacy                 // the deprecated invoice-tool directory, or a file in it
	SourceProject                // found by the upward search from the working directory
	SourceConfig                 // a paths.* or archive.dir setting in the config file
)

// PathReport is one line of `config paths`.
type PathReport struct {
	Name   string
	Path   string // "" only with SourceNone
	Source Source
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
	// LegacyDir is the deprecated invoice-tool directory, "" when it is
	// not read.
	LegacyDir string
	// ConfigTemplate is the text a new config.yaml starts with.
	ConfigTemplate string
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
	Customers() (CustomerTable, error)
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
	Paths(start string) ([]PathReport, error)
	// EditablePath returns f for an editor, creating config.yaml from its
	// template when f is ConfigFile and it does not exist.
	EditablePath(f File) (string, error)
	// Init creates the config directory and the starter files it lacks,
	// and returns the directory.
	Init() (string, []InitFile, error)
	// LegacyFiles returns the files of the legacy directory, relative to
	// it, that the config directory lacks.
	LegacyFiles() ([]string, error)
	// CopyLegacy copies LegacyFiles into the config directory and returns
	// them.
	CopyLegacy() ([]string, error)
	// LegacyFilesUsed returns the files read from the legacy directory so
	// far.
	LegacyFilesUsed() []string
	Locations() Locations
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
	Status string
	Number string
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
}

// Placement says where Archive.Add puts an invoice.
type Placement struct {
	// Path is the archived file to write.
	Path string
	// Overwrite allows Path to exist, when a working copy is re-archived.
	Overwrite bool
	// Replaced are the archived files to back up first.
	Replaced []string
	// Remove is an archived file the invoice supersedes, removed after it
	// is written, or "".
	Remove string
	Now    time.Time
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
	// Entries reads every archived invoice, in the lexical order of a
	// directory walk.
	Entries() ([]ArchiveEntry, error)
	// Add moves the invoice at src into the archive, as p says.
	Add(src string, p Placement) (ArchiveResult, error)
	// Checkout resolves ref, an archived invoice relative to the archive
	// directory, and says where its working copy in workDir goes.
	Checkout(ref, workDir string) (Checkout, error)
	// Dir returns the archive directory, "" when there is none.
	Dir() (string, error)
	// HistoryDir returns where backups are kept.
	HistoryDir() (string, error)
	// Resolve turns a name relative to the archive directory into a path.
	Resolve(name string) (string, error)
	// Existing returns those of paths that exist; a directory is an error.
	Existing(paths ...string) ([]string, error)
	// Protects reports whether path is an existing file inside the archive
	// directory, which nothing but re-archiving overwrites.
	Protects(path string) (bool, error)
	// Source returns the invoice YAML file the PDF at pdf was built from:
	// the one with its name next to it, else the one in the archive. It
	// returns "" when there is none, and an error when the archive has
	// several.
	Source(pdf string) (string, error)
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

// Renderer turns an invoice into the source the Compiler reads.
type Renderer interface {
	// Render checks template t and fills it in. It writes nothing.
	Render(t Template, inv *invoice.Context, epc EPC) (string, error)
	// Write writes source to path and copies t's assets next to it.
	Write(t Template, source, path string) error
	// Build writes source with t's assets to a scratch directory, compiles
	// it there with c, and copies the PDF to output.
	Build(ctx context.Context, c Compiler, t Template, source, output string) error
}

// Compiler turns rendered source into a PDF and returns its path.
type Compiler interface {
	Compile(ctx context.Context, sourcePath string) (string, error)
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

// Draft is where Mailer.Draft put the draft.
type Draft struct {
	// Path is the .eml file, "" when a mail app opened the draft.
	Path string
	// Discard removes a temporary draft; nil for one that is kept.
	Discard func()
}

// Mailer drafts an email with the PDF attached.
type Mailer interface {
	Draft(ctx context.Context, m Message) (Draft, error)
	// Check runs the checks Draft runs on m.Output without writing.
	Check(m Message) error
	// CheckAttachment returns why the file at path cannot be attached, or
	// nil.
	CheckAttachment(path string) error
}
