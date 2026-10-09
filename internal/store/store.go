package store

import (
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/config"
	"github.com/0xboris/invox/internal/fsutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/numbering"
)

// Files are the support files named for one run, relative to the working
// directory or absolute, or "" to look them up.
type Files struct {
	Customers string
	Issuer    string
	Defaults  string
}

// Store reads and writes invox's files. It implements billing.Directory and
// billing.Invoices.
type Store struct {
	Host  Host
	Getwd func() (string, error)
	// Protected reports whether path is an archived invoice, which Create
	// never overwrites.
	Protected func(path string) (bool, error)
	Files     Files
}

var (
	_ billing.Directory = (*Store)(nil)
	_ billing.Invoices  = (*Store)(nil)
)

var supportKinds = map[billing.File]SupportFile{
	billing.CustomersFile: Customers,
	billing.IssuerFile:    Issuer,
	billing.DefaultsFile:  Defaults,
	billing.TemplateFile:  Template,
}

func (s *Store) workDir() (string, error) {
	cwd, err := s.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(cwd), nil
}

func (s *Store) override(f billing.File) string {
	switch f {
	case billing.CustomersFile:
		return s.Files.Customers
	case billing.IssuerFile:
		return s.Files.Issuer
	case billing.DefaultsFile:
		return s.Files.Defaults
	}
	return ""
}

// Locate returns the absolute path of the support file f: the one named
// for this run, else the one found from the working directory.
func (s *Store) Locate(f billing.File) (string, error) {
	baseDir, err := s.workDir()
	if err != nil {
		return "", err
	}
	path := s.override(f)
	if strings.TrimSpace(path) == "" {
		found, err := s.Host.ResolveSupportFile(supportKinds[f], baseDir)
		if err != nil {
			return "", err
		}
		path = found.Path
	}
	if strings.TrimSpace(path) == "" {
		return "", &billing.FileNotFoundError{File: f, Default: s.Host.globalPath(f)}
	}
	return fsutil.Abs(baseDir, path), nil
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
	return LoadCustomer(path, id)
}

// Customers reads customers.yaml.
func (s *Store) Customers() (billing.CustomerTable, error) {
	path, err := s.Locate(billing.CustomersFile)
	if err != nil {
		return nil, err
	}
	return loadCustomerTable(path)
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
	if strings.TrimSpace(ref) == "" {
		ref, err = s.Locate(billing.TemplateFile)
		if err != nil {
			return billing.Template{}, err
		}
	}
	path, err := s.Host.ResolveTemplateReference(baseDir, ref)
	if err != nil {
		return billing.Template{}, &billing.TemplateLookupError{Err: err}
	}
	return s.template(fsutil.Abs(baseDir, path)), nil
}

func (s *Store) template(path string) billing.Template {
	return billing.Template{Name: filepath.Base(path), Path: path}
}

// Templates lists the template catalog and returns its directory.
func (s *Store) Templates() ([]billing.Template, string, error) {
	summaries, err := s.Host.ListTemplates()
	if err != nil {
		return nil, "", err
	}
	dir, err := s.Host.TemplateCatalogDir()
	if err != nil {
		return nil, "", err
	}
	templates := make([]billing.Template, 0, len(summaries))
	for _, summary := range summaries {
		t := s.template(summary.Path)
		t.Name = summary.Name
		templates = append(templates, t)
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
	reports, err := s.Host.Paths(start)
	if err != nil {
		return nil, err
	}
	out := make([]billing.PathReport, 0, len(reports))
	for _, r := range reports {
		out = append(out, billing.PathReport{Name: r.Name, Path: r.Path, Source: r.Source})
	}
	return out, nil
}

// EditablePath returns f for an editor.
func (s *Store) EditablePath(f billing.File) (string, error) {
	if f == billing.ConfigFile {
		return s.Host.EditableConfigPath()
	}
	return s.Locate(f)
}

// Init creates the config directory and the starter files it lacks.
func (s *Store) Init() (string, []billing.InitFile, error) {
	dir, results, err := s.Host.InitializeConfigDir()
	if err != nil {
		return "", nil, err
	}
	files := make([]billing.InitFile, 0, len(results))
	for _, r := range results {
		files = append(files, billing.InitFile(r))
	}
	return dir, files, nil
}

// Settings reads the parts of config.yaml the use cases need.
func (h Host) Settings() (billing.Settings, error) {
	cfg, err := h.Config()
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
