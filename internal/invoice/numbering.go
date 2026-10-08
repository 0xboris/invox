package invoice

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/numbering"
)

type NumberingSettings = numbering.Settings

func (h Host) ResolveNumberingSettings() (NumberingSettings, error) {
	settings := NumberingSettings{
		Pattern: numbering.DefaultPattern,
		Start:   numbering.DefaultStart,
	}

	cfg, err := h.Config()
	if err != nil {
		return NumberingSettings{}, err
	}
	if cfg.Numbering.Pattern != "" {
		settings.Pattern = string(cfg.Numbering.Pattern)
	}
	if cfg.Numbering.Start != nil {
		settings.Start = int64(*cfg.Numbering.Start)
	}

	if err := settings.Validate(); err != nil {
		return NumberingSettings{}, fmt.Errorf("%s: %w", cfg.File, err)
	}
	return settings, nil
}

func (h Host) NextInvoiceNumber(customerID, issueDate string, customer Customer, minimumCounter int64) (string, []string, error) {
	settings, err := h.ResolveNumberingSettings()
	if err != nil {
		return "", nil, err
	}
	start, err := effectiveNumberingStart(customerID, customer, settings.Start)
	if err != nil {
		return "", nil, err
	}

	baseCounter, skipped, err := h.highestArchivedCounter(settings.Pattern, customerID, issueDate, customer)
	if err != nil {
		return "", nil, err
	}
	nextCounter := numbering.Next(start, max(baseCounter, minimumCounter))
	invoiceNumber, err := numbering.Format(settings.Pattern, customerID, customer.Numbering.Code.Trim(), issueDate, nextCounter)
	if err != nil {
		return "", nil, err
	}
	return invoiceNumber, skipped, nil
}

func effectiveNumberingStart(customerID string, customer Customer, globalStart int64) (int64, error) {
	if !customer.Numbering.Start.isSet() {
		return globalStart, nil
	}
	if start := customer.Numbering.Start.Int(); start > 0 {
		return start, nil
	}
	return 0, fmt.Errorf("customers.%s.numbering.start: must be >= 1", customerID)
}

func (h Host) CounterFromInvoiceNumber(invoiceNumber, customerID, issueDate string, customer Customer) (int64, error) {
	settings, err := h.ResolveNumberingSettings()
	if err != nil {
		return 0, err
	}
	return numbering.Parse(settings.Pattern, invoiceNumber, customerID, customer.Numbering.Code.Trim(), issueDate)
}

func (h Host) highestArchivedCounter(pattern, customerID, issueDate string, customer Customer) (int64, []string, error) {
	store, err := h.archiveStore()
	if err != nil {
		return 0, nil, err
	}

	var highest int64
	var skipped []string
	err = store.Walk(func(path string) error {
		identity, ok, err := readArchivedIdentity(path)
		if err != nil || !ok || identity.InvoiceNumber == "" {
			return err
		}

		counter, err := numbering.Parse(pattern, identity.InvoiceNumber, customerID, customer.Numbering.Code.Trim(), issueDate)
		if err != nil {
			if identity.CustomerID == customerID && numbering.InPeriod(pattern, identity.IssueDate, issueDate) {
				skipped = append(skipped, path)
			}
			return nil
		}
		if counter > highest {
			highest = counter
		}
		return nil
	})
	if err != nil {
		return 0, nil, err
	}
	return highest, skipped, nil
}

// highestDraftCounter returns the highest counter used by unarchived invoices
// (status draft or built) directly inside dirs, so that two drafts created
// before either is archived do not get the same number. Files that cannot be
// parsed, are not invoices, or do not match the numbering pattern are ignored.
func (h Host) highestDraftCounter(dirs []string, customerID, issueDate string, customer Customer) (int64, error) {
	settings, err := h.ResolveNumberingSettings()
	if err != nil {
		return 0, err
	}

	seen := make(map[string]bool, len(dirs))
	var highest int64
	for _, dir := range dirs {
		if strings.TrimSpace(dir) == "" || seen[dir] {
			continue
		}
		seen[dir] = true

		entries, err := os.ReadDir(dir)
		if err != nil {
			// The draft scan is best effort: the archive check still
			// guarantees uniqueness, so an unreadable directory must not
			// stop `new`.
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil || info.Size() > maxDraftScanSize {
				continue
			}
			switch strings.ToLower(filepath.Ext(entry.Name())) {
			case ".yaml", ".yml":
			default:
				continue
			}

			invoiceNumber, ok := draftInvoiceNumber(filepath.Join(dir, entry.Name()))
			if !ok {
				continue
			}
			counter, err := numbering.Parse(settings.Pattern, invoiceNumber, customerID, customer.Numbering.Code.Trim(), issueDate)
			if err != nil {
				continue
			}
			if counter > highest {
				highest = counter
			}
		}
	}
	return highest, nil
}

// maxDraftScanSize skips large YAML files in the draft scan; invoices are
// far smaller.
const maxDraftScanSize = 1 << 20

func draftInvoiceNumber(path string) (string, bool) {
	var identity invoiceIdentity
	if err := decodeYAMLFile(path, &identity, false); err != nil || identity.Invoice == nil {
		return "", false
	}
	switch identity.Invoice.Status.Trim() {
	case "draft", "built":
	default:
		return "", false
	}
	invoiceNumber := identity.Invoice.Number.Trim()
	return invoiceNumber, invoiceNumber != ""
}
