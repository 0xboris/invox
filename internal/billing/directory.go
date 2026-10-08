package billing

import "errors"

// CustomerSummary is one line of `customer list`.
type CustomerSummary struct {
	ID string
	// Name is name, else legal_company_name.
	Name   string
	Status string
	// Email is where invoices go, as Customer.InvoiceEmail.
	Email    string
	Currency string
}

// CustomerList is customers.yaml, sorted by ID.
type CustomerList struct {
	File      string
	Customers []CustomerSummary
}

// ListCustomers lists customers.yaml sorted by ID. It reads only names and
// statuses, so keys a customer should not have are no reason to fail.
func (s *Service) ListCustomers() (CustomerList, error) {
	path, err := s.Directory.Locate(CustomersFile)
	if err != nil {
		return CustomerList{}, err
	}
	customers, err := s.Directory.Customers()
	if err != nil {
		return CustomerList{}, err
	}
	ids := customers.IDs()
	summaries := make([]CustomerSummary, 0, len(ids))
	for _, id := range ids {
		customer, _, err := customers.Lookup(id, false)
		if err != nil {
			return CustomerList{}, err
		}
		summaries = append(summaries, CustomerSummary{
			ID:       id,
			Name:     customer.DisplayName(),
			Status:   customer.Status.Trim(),
			Email:    customer.InvoiceEmail(),
			Currency: customer.BillingCurrency(),
		})
	}
	return CustomerList{File: path, Customers: summaries}, nil
}

// TemplateList is the template catalog.
type TemplateList struct {
	// Dir is the directory `template list` reads.
	Dir       string
	Templates []Template
}

// ListTemplates lists the .tex files of the template catalog.
func (s *Service) ListTemplates() (TemplateList, error) {
	templates, dir, err := s.Directory.Templates()
	if err != nil {
		return TemplateList{}, err
	}
	return TemplateList{Dir: dir, Templates: templates}, nil
}

// DefaultTemplate returns the template a command uses when none is named,
// "" when there is none.
func (s *Service) DefaultTemplate() (string, error) {
	path, err := s.Directory.Locate(TemplateFile)
	var notFound *FileNotFoundError
	if errors.As(err, &notFound) {
		return "", nil
	}
	return path, err
}

// Paths reports where each file comes from for a command run in start, in a
// fixed order: config-dir, config, the support files, then archive.
func (s *Service) Paths(start string) ([]PathReport, error) {
	return s.Directory.Paths(start)
}

// InitResult is what Init set up.
type InitResult struct {
	ConfigDir string
	Files     []InitFile
}

// Init creates the config directory with config.yaml and the starter files
// it lacks. It never replaces a file.
func (s *Service) Init() (InitResult, error) {
	dir, files, err := s.Directory.Init()
	if err != nil {
		return InitResult{}, err
	}
	return InitResult{ConfigDir: dir, Files: files}, nil
}

// LegacyFiles returns the files of the deprecated config directory that the
// config directory lacks.
func (s *Service) LegacyFiles() ([]string, error) {
	return s.Directory.LegacyFiles()
}

// CopyLegacyFiles copies LegacyFiles into the config directory and returns
// them. It never replaces a file and leaves the legacy directory as it is.
func (s *Service) CopyLegacyFiles() ([]string, error) {
	return s.Directory.CopyLegacy()
}

// EditablePath returns the file f for an editor: customers.yaml where the
// commands find it, or config.yaml, created from its template when missing.
func (s *Service) EditablePath(f File) (string, error) {
	return s.Directory.EditablePath(f)
}
