package invoice

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ArchivedInvoiceSummary struct {
	Filename   string
	CustomerID string
	IssueDate  string
	Status     string
}

type archivedInvoiceRecord struct {
	Path          string
	Filename      string
	CustomerID    string
	IssueDate     string
	Status        string
	InvoiceNumber string
}

func ListArchivedInvoices() ([]ArchivedInvoiceSummary, error) {
	records, err := collectArchivedInvoiceRecords()
	if err != nil {
		return nil, err
	}

	summaries := make([]ArchivedInvoiceSummary, 0, len(records))
	for _, record := range records {
		summaries = append(summaries, ArchivedInvoiceSummary{
			Filename:   record.Filename,
			CustomerID: record.CustomerID,
			IssueDate:  record.IssueDate,
			Status:     record.Status,
		})
	}
	return summaries, nil
}

func latestArchivedInvoicePath(customerID string) (string, bool, error) {
	records, err := collectArchivedInvoiceRecords()
	if err != nil {
		return "", false, err
	}

	var latest archivedInvoiceRecord
	found := false
	for _, record := range records {
		if record.CustomerID != customerID {
			continue
		}
		if !found || archivedInvoiceIsNewer(record, latest) {
			latest = record
			found = true
		}
	}
	if !found {
		return "", false, nil
	}
	return latest.Path, true, nil
}

func collectArchivedInvoiceRecords() ([]archivedInvoiceRecord, error) {
	archiveDir, err := ResolveArchiveDir()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(archiveDir) == "" {
		return nil, nil
	}

	info, err := os.Stat(archiveDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s: archive.dir must point to a directory", archiveDir)
	}

	records := make([]archivedInvoiceRecord, 0)
	err = filepath.WalkDir(archiveDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !isArchivedInvoicePath(path) {
			return nil
		}

		record, ok, err := archivedInvoiceRecordFromPath(path, archiveDir)
		if err != nil || !ok {
			return err
		}
		records = append(records, record)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Filename < records[j].Filename
	})
	return records, nil
}

func archivedInvoiceValue(path string) (any, bool, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		value, err := loadYAML(path)
		if err != nil {
			return nil, false, err
		}
		return value, true, nil
	case ".md", ".markdown":
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, false, err
		}
		frontMatter, ok := markdownFrontMatter(source)
		if !ok {
			return nil, false, nil
		}

		value, err := parseYAMLSource(frontMatter, "front matter in "+path)
		if err != nil {
			return nil, false, err
		}
		return value, true, nil
	default:
		return nil, false, nil
	}
}

func archivedInvoiceRecordFromPath(path, archiveDir string) (archivedInvoiceRecord, bool, error) {
	value, ok, err := archivedInvoiceValue(path)
	if err != nil || !ok {
		return archivedInvoiceRecord{}, ok, err
	}

	root, ok := value.(map[string]any)
	if !ok {
		return archivedInvoiceRecord{}, false, nil
	}
	invoice, ok := root["invoice"].(map[string]any)
	if !ok {
		return archivedInvoiceRecord{}, false, nil
	}

	filename, err := filepath.Rel(archiveDir, path)
	if err != nil {
		filename = filepath.Base(path)
	}
	status := strings.TrimSpace(asString(invoice["status"]))
	if status == "" {
		status = "archived"
	}

	return archivedInvoiceRecord{
		Path:          path,
		Filename:      filename,
		CustomerID:    strings.TrimSpace(asString(root["customer_id"])),
		IssueDate:     strings.TrimSpace(asString(invoice["issue_date"])),
		Status:        status,
		InvoiceNumber: strings.TrimSpace(asString(invoice["number"])),
	}, true, nil
}

func archivedInvoiceIsNewer(left, right archivedInvoiceRecord) bool {
	leftDate, leftOK := parseArchivedIssueDate(left.IssueDate)
	rightDate, rightOK := parseArchivedIssueDate(right.IssueDate)

	switch {
	case leftOK && !rightOK:
		return true
	case !leftOK && rightOK:
		return false
	case leftOK && rightOK && !leftDate.Equal(rightDate):
		return leftDate.After(rightDate)
	}

	switch {
	case left.InvoiceNumber != right.InvoiceNumber:
		return left.InvoiceNumber > right.InvoiceNumber
	case left.Filename != right.Filename:
		return left.Filename > right.Filename
	default:
		return left.Path > right.Path
	}
}

func parseArchivedIssueDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func resolveArchiveInputPath(name string) (string, string, error) {
	archiveDir, err := ResolveArchiveDir()
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(archiveDir) == "" {
		return "", "", fmt.Errorf("archive directory is unavailable")
	}

	targetPath, relativePath, err := resolveArchivePath(archiveDir, name)
	if err != nil {
		return "", "", err
	}
	info, err := os.Stat(targetPath)
	if errors.Is(err, os.ErrNotExist) {
		return "", "", fmt.Errorf("%s does not exist in %s", relativePath, archiveDir)
	}
	if err != nil {
		return "", "", err
	}
	if info.IsDir() {
		return "", "", fmt.Errorf("%s: archived invoice must be a file", targetPath)
	}
	return targetPath, relativePath, nil
}

func resolveArchiveTargetPath(archiveDir, relativePath string) (string, error) {
	targetPath, _, err := resolveArchivePath(archiveDir, relativePath)
	return targetPath, err
}

func resolveArchivePath(archiveDir, name string) (string, string, error) {
	archiveDir, err := filepath.Abs(archiveDir)
	if err != nil {
		return "", "", err
	}

	cleanName := filepath.Clean(strings.TrimSpace(name))
	if cleanName == "" || cleanName == "." {
		return "", "", fmt.Errorf("archive filename must not be empty")
	}
	if filepath.IsAbs(cleanName) {
		return "", "", fmt.Errorf("archive filename must be relative to archive.dir, got %s", cleanName)
	}

	targetPath, err := filepath.Abs(filepath.Join(archiveDir, cleanName))
	if err != nil {
		return "", "", err
	}
	relativePath, err := filepath.Rel(archiveDir, targetPath)
	if err != nil {
		return "", "", err
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("%s must stay within %s", cleanName, archiveDir)
	}
	return targetPath, filepath.Clean(relativePath), nil
}
