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
	summaries := make([]CustomerSummary, 0, len(customers))
	for _, entry := range customers {
		if err := withoutUnknownKeys(entry.Err); err != nil {
			return CustomerList{}, err
		}
		summaries = append(summaries, CustomerSummary{
			ID:       entry.ID,
			Name:     entry.Customer.DisplayName(),
			Status:   entry.Customer.Status.Trim(),
			Email:    entry.Customer.InvoiceEmail(),
			Currency: entry.Customer.BillingCurrency(),
		})
	}
	return CustomerList{File: path, Customers: summaries}, nil
}

// TemplateList is the template catalog.
type TemplateList struct {
	// Dir is the directory `template list` reads.
	Dir       string
	Templates []Template
	// Default is the template a command uses when none is named, "" when
	// there is none or it was not asked for.
	Default string
}

// ListTemplates lists the .tex files of the template catalog and, with
// withDefault, the default template, which takes a search from the working
// directory.
func (s *Service) ListTemplates(withDefault bool) (TemplateList, error) {
	templates, dir, err := s.Directory.Templates()
	if err != nil || !withDefault {
		return TemplateList{Dir: dir, Templates: templates}, err
	}
	def, err := s.Directory.Locate(TemplateFile)
	var notFound *FileNotFoundError
	if errors.As(err, &notFound) && notFound.Path == "" {
		def, err = "", nil
	}
	if err != nil {
		return TemplateList{}, err
	}
	return TemplateList{Dir: dir, Templates: templates, Default: def}, nil
}

// Paths reports where each file comes from for this run, in a fixed order:
// config-dir, config, the support files, then archive.
func (s *Service) Paths() ([]PathReport, error) {
	return s.Directory.Paths()
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

// EditablePath returns the file f for an editor: customers.yaml where the
// commands find it, or config.yaml, created from its template when missing.
func (s *Service) EditablePath(f File) (string, error) {
	return s.Directory.EditablePath(f)
}
