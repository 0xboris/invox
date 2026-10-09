package billing

import (
	"errors"
	"fmt"
	"time"

	"github.com/0xboris/invox/internal/invoice"
)

// NewRequest says which invoice New drafts and where.
type NewRequest struct {
	CustomerID string
	// WorkDir is searched for unarchived drafts when numbering, and holds
	// the new invoice when Output is "".
	WorkDir string
	Output  string
	// FromLast starts from the customer's last archived invoice instead of
	// invoice_defaults.yaml.
	FromLast bool
	// Overwrite replaces an existing file at Output, unless it is in the
	// archive directory.
	Overwrite bool
	// DryRun runs every check and writes nothing.
	DryRun bool
}

// NewResult is the invoice New drafted.
type NewResult struct {
	Number string
	Path   string
	// Skipped are archived invoices of the customer whose numbers do not
	// match numbering.pattern, so they did not count towards Number.
	Skipped []string
	// Unread is what the archive walk could not read.
	Unread Unread
}

// New drafts the next invoice of a customer from invoice_defaults.yaml or
// the customer's last archived invoice: numbered, dated today, status draft.
func (s *Service) New(req NewRequest) (NewResult, error) {
	_, issuerPath, err := s.locateParties()
	if err != nil {
		return NewResult{}, err
	}
	defaultsPath, err := s.Directory.Defaults()
	var notFound *FileNotFoundError
	if req.FromLast && errors.As(err, &notFound) {
		// --from-last copies the last archived invoice instead.
		defaultsPath, err = "", nil
	}
	if err != nil {
		return NewResult{}, err
	}
	customer, err := s.Directory.Customer(req.CustomerID)
	if err != nil {
		return NewResult{}, err
	}
	issuer, err := s.Directory.Issuer()
	if err != nil {
		return NewResult{}, err
	}
	if issuer.Payment == nil {
		return NewResult{}, fmt.Errorf("%s: missing `payment` mapping", issuerPath)
	}
	from, err := s.newSource(defaultsPath, req.CustomerID, req.FromLast)
	if err != nil {
		return NewResult{}, err
	}

	now := s.Now().In(time.Local)
	issueDate := now.Format(time.DateOnly)
	draftCounter, err := s.highestDraftCounter(req.WorkDir, req.Output, req.CustomerID, issueDate, customer)
	if err != nil {
		return NewResult{}, err
	}
	number, skipped, unread, err := s.nextNumber(req.CustomerID, issueDate, customer, draftCounter)
	if err != nil {
		return NewResult{}, err
	}
	dueDays, err := issuerDueDays(issuerPath, *issuer.Payment)
	if err != nil {
		return NewResult{}, err
	}

	issued, _ := invoice.ParseDate(issueDate)
	due, _ := invoice.ParseDate(now.AddDate(0, 0, dueDays).Format(time.DateOnly))
	paid, _ := invoice.ParseDecimal("0")
	// A rate fills in only where the source has none.
	rate, _ := invoice.ParseRate(trimmedRate(customer.Tax.DefaultVATRate))
	draft := invoice.Invoice{
		CustomerID: invoice.Text(req.CustomerID),
		Header: &invoice.Header{
			Number:     invoice.Text(number),
			IssueDate:  issued,
			DueDate:    due,
			Status:     invoice.Draft,
			PaidAmount: paid,
			VATPercent: rate,
		},
		Positions: []invoice.Position{},
	}
	// An archived invoice is a record: keys it has that invox does not
	// know are copied over as they are, for validate to report.
	check := CheckStrict
	if req.FromLast {
		check = CheckLenient
	}
	output, err := s.Invoices.Create(req.Output, from, draft, CreateOptions{Dir: req.WorkDir, Overwrite: req.Overwrite, DryRun: req.DryRun, Check: check})
	if err != nil {
		return NewResult{}, err
	}
	return NewResult{Number: number, Path: output, Skipped: skipped, Unread: unread}, nil
}

// newSource returns the document a new invoice starts from, after checking
// that it can be read.
func (s *Service) newSource(defaultsPath, customerID string, fromLast bool) (string, error) {
	if !fromLast {
		if _, err := s.Invoices.Load(defaultsPath); err != nil && !isDecodeError(err) {
			return "", err
		}
		return defaultsPath, nil
	}
	archivePath, ok, err := s.latestArchived(customerID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("no archived invoice found for customer_id `%s`", customerID)
	}
	if _, err := s.Invoices.Load(archivePath); err != nil && !isDecodeError(err) {
		return "", err
	}
	return archivePath, nil
}

func issuerDueDays(issuerPath string, payment invoice.Payment) (int, error) {
	if !payment.DueDays.IsSet() {
		return 0, fmt.Errorf("%s: payment.due_days: missing value", issuerPath)
	}
	if payment.DueDays.Int() < 0 {
		return 0, fmt.Errorf("%s: payment.due_days: must be >= 0", issuerPath)
	}
	return int(payment.DueDays.Int()), nil
}
