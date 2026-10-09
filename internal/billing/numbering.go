package billing

import (
	"fmt"
	"strings"

	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/numbering"
)

// numberingSettings returns numbering.pattern and numbering.start, checked.
func (s *Service) numberingSettings() (numbering.Settings, error) {
	settings, err := s.Settings()
	if err != nil {
		return numbering.Settings{}, err
	}
	if err := settings.Numbering.Validate(); err != nil {
		return numbering.Settings{}, fmt.Errorf("%s: %w", settings.File, err)
	}
	return settings.Numbering, nil
}

// NextNumber returns the next invoice number of the customer on issueDate:
// above every archived invoice's counter, minimumCounter and the start
// before it. skipped are the customer's archived invoices from the same
// period whose numbers do not match the pattern.
func (s *Service) NextNumber(customerID, issueDate string, customer invoice.Customer, minimumCounter int64) (string, []string, error) {
	settings, err := s.numberingSettings()
	if err != nil {
		return "", nil, err
	}
	start, err := numberingStart(customerID, customer, settings.Start)
	if err != nil {
		return "", nil, err
	}
	highest, skipped, err := s.highestArchivedCounter(settings.Pattern, customerID, issueDate, customer)
	if err != nil {
		return "", nil, err
	}
	next := numbering.Next(start, max(highest, minimumCounter))
	number, err := numbering.Format(settings.Pattern, customerID, customer.Numbering.Code.Trim(), issueDate, next)
	if err != nil {
		return "", nil, err
	}
	return number, skipped, nil
}

func numberingStart(customerID string, customer invoice.Customer, globalStart int64) (int64, error) {
	if !customer.Numbering.Start.IsSet() {
		return globalStart, nil
	}
	if start := customer.Numbering.Start.Int(); start > 0 {
		return start, nil
	}
	return 0, fmt.Errorf("customers.%s.numbering.start: must be >= 1", customerID)
}

func (s *Service) highestArchivedCounter(pattern, customerID, issueDate string, customer invoice.Customer) (int64, []string, error) {
	entries, err := s.Archives.Entries()
	if err != nil {
		return 0, nil, err
	}
	var highest int64
	var skipped []string
	for _, entry := range entries {
		if entry.Number == "" {
			continue
		}
		counter, err := numbering.Parse(pattern, entry.Number, customerID, customer.Numbering.Code.Trim(), issueDate)
		if err != nil {
			if entry.CustomerID == customerID && numbering.InPeriod(pattern, entry.IssueDate, issueDate) {
				skipped = append(skipped, entry.Path)
			}
			continue
		}
		highest = max(highest, counter)
	}
	return highest, skipped, nil
}

// highestDraftCounter returns the highest counter used by unarchived
// invoices (status draft or built) where a new invoice goes, workDir and
// the directory of output, so that two drafts created before either is
// archived do not get the same number. Files that cannot be read or do not
// match the numbering pattern are ignored.
func (s *Service) highestDraftCounter(workDir, output, customerID, issueDate string, customer invoice.Customer) (int64, error) {
	settings, err := s.numberingSettings()
	if err != nil {
		return 0, err
	}
	var highest int64
	for _, head := range s.Invoices.Drafts(workDir, output) {
		if !head.Status.Allows(invoice.Numbering) || head.Number == "" {
			continue
		}
		counter, err := numbering.Parse(settings.Pattern, head.Number, customerID, customer.Numbering.Code.Trim(), issueDate)
		if err != nil {
			continue
		}
		highest = max(highest, counter)
	}
	return highest, nil
}

// IncrementResult is the number Increment gave an invoice.
type IncrementResult struct {
	CustomerID string
	OldNumber  string
	NewNumber  string
	// Skipped are archived invoices of the customer whose numbers do not
	// match numbering.pattern, so they did not count towards NewNumber.
	Skipped []string
}

// Increment writes the next invoice number into the invoice at path. With
// dryRun it returns the same result and writes nothing.
func (s *Service) Increment(path string, dryRun bool) (IncrementResult, error) {
	if _, err := s.Directory.Locate(CustomersFile); err != nil {
		return IncrementResult{}, err
	}
	head, err := s.Invoices.Head(path)
	if err != nil {
		return IncrementResult{}, err
	}
	switch {
	case head.CustomerID == "":
		return IncrementResult{}, fmt.Errorf("%s: missing `customer_id`", path)
	case !head.HasHeader:
		return IncrementResult{}, fmt.Errorf("%s: missing `invoice` mapping", path)
	case head.IssueDate == "":
		return IncrementResult{}, fmt.Errorf("%s: invoice.issue_date: missing value", path)
	}
	if _, err := invoice.ParseDate(head.IssueDate); err != nil {
		return IncrementResult{}, fmt.Errorf("%s: invoice.issue_date: expected YYYY-MM-DD, got `%s`", path, head.IssueDate)
	}
	if head.Number == "" {
		return IncrementResult{}, fmt.Errorf("%s: invoice.number: missing value", path)
	}

	customer, err := s.Directory.Customer(head.CustomerID)
	if err != nil {
		return IncrementResult{}, err
	}
	settings, err := s.numberingSettings()
	if err != nil {
		return IncrementResult{}, err
	}
	current, err := numbering.Parse(settings.Pattern, head.Number, head.CustomerID, customer.Numbering.Code.Trim(), head.IssueDate)
	if err != nil {
		return IncrementResult{}, err
	}
	next, skipped, err := s.NextNumber(head.CustomerID, head.IssueDate, customer, current)
	if err != nil {
		return IncrementResult{}, err
	}
	result := IncrementResult{CustomerID: head.CustomerID, OldNumber: head.Number, NewNumber: next, Skipped: skipped}
	if dryRun {
		return result, nil
	}
	err = s.Invoices.Update(path, func(inv *invoice.Invoice) error {
		inv.Header.Number = invoice.Text(next)
		return nil
	})
	if err != nil {
		return IncrementResult{}, err
	}
	return result, nil
}

// trimmedRate is rate as written without its % sign.
func trimmedRate(rate invoice.Rate) string {
	return strings.TrimSuffix(rate.String(), "%")
}
