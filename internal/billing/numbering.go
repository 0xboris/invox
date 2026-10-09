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

// nextNumber returns the next invoice number of the customer on issueDate:
// above every archived invoice's counter, minimumCounter and the start
// before it. skipped are the customer's archived invoices from the same
// period whose numbers do not match the pattern, and unread what the
// archive walk could not read.
func (s *Service) nextNumber(customerID, issueDate string, customer invoice.Customer, minimumCounter int64) (number string, skipped []string, unread Unread, err error) {
	settings, err := s.numberingSettings()
	if err != nil {
		return "", nil, Unread{}, err
	}
	start, err := numberingStart(customerID, customer, settings.Start)
	if err != nil {
		return "", nil, Unread{}, err
	}
	highest, skipped, unread, err := s.highestArchivedCounter(settings.Pattern, customerID, issueDate, customer)
	if err != nil {
		return "", nil, Unread{}, err
	}
	next := numbering.Next(start, max(highest, minimumCounter))
	number, err = numbering.Format(settings.Pattern, customerID, customer.Numbering.Code.Trim(), issueDate, next)
	if err != nil {
		return "", nil, Unread{}, err
	}
	return number, skipped, unread, nil
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

func (s *Service) highestArchivedCounter(pattern, customerID, issueDate string, customer invoice.Customer) (int64, []string, Unread, error) {
	entries, unread, err := s.Archives.Entries()
	if err != nil {
		return 0, nil, Unread{}, err
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
	return highest, skipped, unread, nil
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
	for _, draft := range s.Invoices.Drafts(workDir, output) {
		number := draft.Header.Number.Trim()
		if !draft.Header.Status.Allows(invoice.Numbering) || number == "" {
			continue
		}
		counter, err := numbering.Parse(settings.Pattern, number, customerID, customer.Numbering.Code.Trim(), issueDate)
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
	// Unread is what the archive walk could not read.
	Unread Unread
}

// Increment writes the next invoice number into the invoice at path. With
// dryRun it returns the same result and writes nothing.
func (s *Service) Increment(path string, dryRun bool) (IncrementResult, error) {
	if _, err := s.Directory.Locate(CustomersFile); err != nil {
		return IncrementResult{}, err
	}
	inv, err := s.Invoices.Load(path)
	// Only the fields numbering reads have to decode.
	read := map[string]bool{"customer_id": true, "invoice.number": true, "invoice.issue_date": true}
	if err := keepDecodeErrors(err, func(e *DecodeError) bool { return read[e.Field] }); err != nil {
		return IncrementResult{}, err
	}
	customerID := inv.CustomerID.Trim()
	switch {
	case customerID == "":
		return IncrementResult{}, fmt.Errorf("%s: missing `customer_id`", path)
	case inv.Header == nil:
		return IncrementResult{}, fmt.Errorf("%s: missing `invoice` mapping", path)
	case !inv.Header.IssueDate.IsSet():
		return IncrementResult{}, fmt.Errorf("%s: invoice.issue_date: missing value", path)
	}
	issueDate, number := inv.Header.IssueDate.String(), inv.Header.Number.Trim()
	if number == "" {
		return IncrementResult{}, fmt.Errorf("%s: invoice.number: missing value", path)
	}

	customer, err := s.Directory.Customer(customerID)
	if err != nil {
		return IncrementResult{}, err
	}
	settings, err := s.numberingSettings()
	if err != nil {
		return IncrementResult{}, err
	}
	current, err := numbering.Parse(settings.Pattern, number, customerID, customer.Numbering.Code.Trim(), issueDate)
	if err != nil {
		return IncrementResult{}, err
	}
	next, skipped, unread, err := s.nextNumber(customerID, issueDate, customer, current)
	if err != nil {
		return IncrementResult{}, err
	}
	result := IncrementResult{CustomerID: customerID, OldNumber: number, NewNumber: next, Skipped: skipped, Unread: unread}
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
