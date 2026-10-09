package store

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/config"
	"github.com/0xboris/invox/internal/fsutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/numbering"
)

// Store reads and writes invox's files. It implements billing.Directory and
// billing.Invoices.
type Store struct {
	Host  Host
	Getwd func() (string, error)
	// Protected reports whether path is an archived invoice, which Create
	// never overwrites.
	Protected func(path string) (bool, error)
	// Files are the support files named for this run, relative to the
	// working directory or absolute. A file missing or "" is looked up.
	Files map[billing.File]string
}

var (
	_ billing.Directory = (*Store)(nil)
	_ billing.Invoices  = (*Store)(nil)
)

func (s *Store) workDir() (string, error) {
	cwd, err := s.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(cwd), nil
}

// Locate returns the absolute path of the support file f: the one named
// for this run, else the one found from the working directory. A file
// named for this run or by a paths.* setting must exist; a missing one
// named by the setting is a problem with config.yaml.
func (s *Store) Locate(f billing.File) (string, error) {
	found, err := s.locate(f)
	if err != nil {
		return "", err
	}
	missing := mustExist(f, found.Path)
	if missing == nil {
		return found.Path, nil
	}
	if found.Source != billing.SourceConfig {
		return "", missing
	}
	cfg, err := s.Host.config()
	if err != nil {
		return "", err
	}
	missing.Config = cfg.File
	return "", &billing.ConfigError{Err: missing}
}

// locate is Locate without the check that the file exists. A file named
// for this run has SourceExplicit.
func (s *Store) locate(f billing.File) (resolved, error) {
	baseDir, err := s.workDir()
	if err != nil {
		return resolved{}, err
	}
	found := resolved{Path: s.Files[f], Source: billing.SourceExplicit}
	if strings.TrimSpace(found.Path) == "" {
		found, err = s.Host.resolveSupportFile(f, baseDir)
		if err != nil {
			return resolved{}, err
		}
	}
	if strings.TrimSpace(found.Path) == "" {
		return resolved{}, &billing.FileNotFoundError{File: f, Default: s.Host.globalPath(f)}
	}
	found.Path = fsutil.Abs(baseDir, found.Path)
	return found, nil
}

// mustExist returns a *billing.FileNotFoundError when path, the file f,
// does not exist, and nil otherwise.
func mustExist(f billing.File, path string) *billing.FileNotFoundError {
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return &billing.FileNotFoundError{File: f, Path: path}
	}
	return nil
}

func (h Host) globalPath(f billing.File) string {
	switch f {
	case billing.CustomersFile:
		return h.GlobalCustomersPath()
	case billing.IssuerFile:
		return h.GlobalIssuerPath()
	case billing.DefaultsFile:
		return h.GlobalInvoiceDefaultsPath()
	case billing.TemplateFile:
		return h.GlobalTemplatePath()
	}
	return h.GlobalConfigPath()
}

// Customer decodes the entry of id in customers.yaml.
func (s *Store) Customer(id string) (invoice.Customer, error) {
	path, err := s.Locate(billing.CustomersFile)
	if err != nil {
		return invoice.Customer{}, err
	}
	entries, err := loadCustomerEntries(path)
	if err != nil {
		return invoice.Customer{}, err
	}
	entry, ok := entries[id]
	if !ok {
		return invoice.Customer{}, &invoice.UnknownCustomerError{Path: path, CustomerID: id}
	}
	return decodeCustomer(path, id, entry)
}

// Customers decodes every entry of customers.yaml, sorted by ID.
func (s *Store) Customers() ([]billing.CustomerEntry, error) {
	path, err := s.Locate(billing.CustomersFile)
	if err != nil {
		return nil, err
	}
	entries, err := loadCustomerEntries(path)
	if err != nil {
		return nil, err
	}
	customers := make([]billing.CustomerEntry, 0, len(entries))
	for _, id := range slices.Sorted(maps.Keys(entries)) {
		customer, err := decodeCustomer(path, id, entries[id])
		customers = append(customers, billing.CustomerEntry{ID: id, Customer: customer, Err: err})
	}
	return customers, nil
}

// Issuer decodes issuer.yaml.
func (s *Store) Issuer() (invoice.Issuer, error) {
	path, err := s.Locate(billing.IssuerFile)
	if err != nil {
		return invoice.Issuer{}, err
	}
	var issuer invoice.Issuer
	err = decodeYAMLFile(path, &issuer, true)
	return issuer, err
}

// Defaults returns invoice_defaults.yaml.
func (s *Store) Defaults() (string, error) {
	return s.Locate(billing.DefaultsFile)
}

// Template resolves ref, a path or a template name, or the default
// template when ref is "".
func (s *Store) Template(ref string) (billing.Template, error) {
	baseDir, err := s.workDir()
	if err != nil {
		return billing.Template{}, err
	}
	named := strings.TrimSpace(ref) != ""
	if !named {
		ref, err = s.Locate(billing.TemplateFile)
		if err != nil {
			return billing.Template{}, err
		}
	}
	path, err := s.Host.resolveTemplateReference(baseDir, ref)
	if err != nil {
		return billing.Template{}, &billing.TemplateLookupError{Err: err}
	}
	path = fsutil.Abs(baseDir, path)
	if named {
		if err := mustExist(billing.TemplateFile, path); err != nil {
			return billing.Template{}, err
		}
	}
	return billing.Template{Name: filepath.Base(path), Path: path}, nil
}

// Templates lists the template catalog and returns its directory.
func (s *Store) Templates() ([]billing.Template, string, error) {
	templates, err := s.Host.listTemplates()
	if err != nil {
		return nil, "", err
	}
	dir, err := s.Host.templateCatalogDir()
	if err != nil {
		return nil, "", err
	}
	return templates, dir, nil
}

// Paths reports where each file comes from for a command run in the
// working directory.
func (s *Store) Paths() ([]billing.PathReport, error) {
	start, err := s.workDir()
	if err != nil {
		return nil, err
	}
	return s.Host.paths(start)
}

// EditablePath returns f for an editor.
func (s *Store) EditablePath(f billing.File) (string, error) {
	if f == billing.ConfigFile {
		return s.Host.editableConfigPath()
	}
	found, err := s.locate(f)
	return found.Path, err
}

// Init creates the config directory and the starter files it lacks.
func (s *Store) Init() (string, []billing.InitFile, error) {
	return s.Host.initConfigDir()
}

// Settings reads the parts of config.yaml the use cases need.
func (h Host) Settings() (billing.Settings, error) {
	cfg, err := h.config()
	if err != nil {
		return billing.Settings{}, err
	}
	return settingsOf(cfg), nil
}

func settingsOf(cfg *config.Config) billing.Settings {
	settings := billing.Settings{
		File:         cfg.File,
		Numbering:    numbering.Settings{Pattern: numbering.DefaultPattern, Start: numbering.DefaultStart},
		EmailSubject: string(cfg.Email.Subject),
		EmailBody:    string(cfg.Email.Body),
	}
	if cfg.Numbering.Pattern != "" {
		settings.Numbering.Pattern = string(cfg.Numbering.Pattern)
	}
	if cfg.Numbering.Start != nil {
		settings.Numbering.Start = int64(*cfg.Numbering.Start)
	}
	return settings
}
