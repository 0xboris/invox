package invoice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateNewInvoicePrefillsDatesAndNumber(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.Local)

	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	h := writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	writeArchivedInvoiceMarkdown(t, archiveDir, "2026-03-05.md", "CUST-001-001")
	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")

	created, err := h.CreateNewInvoice(
		now,
		t.TempDir(),
		defaultsPath,
		outputPath,
		customersPath,
		issuerPath,
		"CUST-001",
		false,
	)
	if err != nil {
		t.Fatalf("CreateNewInvoice returned error: %v", err)
	}
	if created.Number != "CUST-001-002" {
		t.Fatalf("invoiceNumber = %q, want %q", created.Number, "CUST-001-002")
	}

	source, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(source)
	for _, want := range []string{
		"customer_id: CUST-001",
		"number: CUST-001-002",
		"issue_date: \"2026-03-06\"",
		"due_date: \"2026-04-05\"",
		"period: \"Leistungszeitraum: \"",
		"vat_percent: 20",
		"paid_amount: \"0\"",
		"name: Beispielposition",
		"description: Beschreibung der Leistung",
		"unit_price: 100",
		"quantity: 1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("created invoice does not contain %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "payment_terms_text:") {
		t.Fatalf("created invoice should not contain payment_terms_text anymore:\n%s", text)
	}
	for _, forbidden := range []string{"period_label:", "vat_rate_percent:", "line_items:"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("created invoice should not contain legacy key %q:\n%s", forbidden, text)
		}
	}
}

func TestCreateNewInvoicePrefillsVATFromCustomerDefaultWhenMissing(t *testing.T) {
	t.Parallel()

	h := isolatedHost(t)
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.Local)

	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	if err := os.WriteFile(customersPath, []byte(strings.TrimSpace(`
CUST-001:
  name: Appsters GmbH
  tax:
    default_vat_rate: 13
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}
	if err := os.WriteFile(defaultsPath, []byte(strings.TrimSpace(`
invoice:
  period: "Leistungszeitraum: "
positions:
  - name: Beispielposition
    description: Beschreibung der Leistung
    unit_price: 100
    quantity: 1
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(invoice_defaults.yaml) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")
	_, err := h.CreateNewInvoice(now, t.TempDir(), defaultsPath, outputPath, customersPath, issuerPath, "CUST-001", false)
	if err != nil {
		t.Fatalf("CreateNewInvoice returned error: %v", err)
	}

	source, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	if !strings.Contains(string(source), "vat_percent: \"13\"") {
		t.Fatalf("created invoice does not contain customer VAT default:\n%s", string(source))
	}
}

func TestCreateNewInvoicePreservesSourceVATWhenCustomerHasDefault(t *testing.T) {
	t.Parallel()

	h := isolatedHost(t)
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.Local)

	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	if err := os.WriteFile(customersPath, []byte(strings.TrimSpace(`
CUST-001:
  name: Appsters GmbH
  tax:
    default_vat_rate: 13
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")
	_, err := h.CreateNewInvoice(now, t.TempDir(), defaultsPath, outputPath, customersPath, issuerPath, "CUST-001", false)
	if err != nil {
		t.Fatalf("CreateNewInvoice returned error: %v", err)
	}

	source, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(source)
	if !strings.Contains(text, "vat_percent: 20") {
		t.Fatalf("created invoice does not preserve source VAT:\n%s", text)
	}
	if strings.Contains(text, "vat_percent: \"13\"") {
		t.Fatalf("created invoice should not overwrite source VAT with customer default:\n%s", text)
	}
}

func TestCreateNewInvoiceStartsFromConfiguredStartWhenArchiveHasNoMatch(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.Local)

	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	h := writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 7\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")
	created, err := h.CreateNewInvoice(
		now,
		t.TempDir(),
		defaultsPath,
		outputPath,
		customersPath,
		issuerPath,
		"CUST-001",
		false,
	)
	if err != nil {
		t.Fatalf("CreateNewInvoice returned error: %v", err)
	}
	if created.Number != "CUST-001-007" {
		t.Fatalf("invoiceNumber = %q, want %q", created.Number, "CUST-001-007")
	}
}

func TestCreateNewInvoiceFailsWhenArchiveContainsInvalidFrontMatter(t *testing.T) {
	t.Parallel()

	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	h := writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	archivePath := filepath.Join(archiveDir, "2026-03-05.md")
	if err := os.WriteFile(archivePath, []byte(strings.Join([]string{
		"---",
		"invoice:",
		"  number: CUST-001-010",
		"  broken: [",
		"---",
		"",
		"# Archived invoice",
		"",
	}, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", archivePath, err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")
	_, err := h.CreateNewInvoice(
		time.Now(),
		t.TempDir(),
		defaultsPath,
		outputPath,
		customersPath,
		issuerPath,
		"CUST-001",
		false,
	)
	if err == nil {
		t.Fatal("CreateNewInvoice returned nil error for invalid archived invoice front matter")
	}
	if !strings.Contains(err.Error(), archivePath) {
		t.Fatalf("error %q does not contain archived invoice path %q", err.Error(), archivePath)
	}
	if !strings.Contains(err.Error(), "yaml") {
		t.Fatalf("error %q does not contain YAML parse context", err.Error())
	}
	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Fatalf("output file should not be created, stat error = %v", statErr)
	}
}

func TestCreateNewInvoiceRejectsLegacyDefaultKeys(t *testing.T) {
	t.Parallel()

	h := isolatedHost(t)
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.Local)

	customersPath, issuerPath, defaultsPath := writeLegacyDraftFixtures(t)
	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")

	_, err := h.CreateNewInvoice(
		now,
		t.TempDir(),
		defaultsPath,
		outputPath,
		customersPath,
		issuerPath,
		"CUST-001",
		false,
	)
	if err == nil {
		t.Fatal("CreateNewInvoice returned nil error for legacy default keys")
	}
	for _, want := range []string{
		defaultsPath + ":2: invoice.period_label: unsupported key; use invoice.period",
		defaultsPath + ":3: invoice.vat_rate_percent: unsupported key; use invoice.vat_percent",
		defaultsPath + ":4: line_items: unsupported key; use positions",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err.Error(), want)
		}
	}
}

func TestCreateNewInvoiceUsesCustomerSpecificStartWhenArchiveHasNoMatch(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.Local)

	customersPath, issuerPath, defaultsPath := writeDraftFixturesWithCustomerStart(t, "7")
	archiveDir := t.TempDir()
	h := writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")
	created, err := h.CreateNewInvoice(
		now,
		t.TempDir(),
		defaultsPath,
		outputPath,
		customersPath,
		issuerPath,
		"CUST-001",
		false,
	)
	if err != nil {
		t.Fatalf("CreateNewInvoice returned error: %v", err)
	}
	if created.Number != "CUST-001-007" {
		t.Fatalf("invoiceNumber = %q, want %q", created.Number, "CUST-001-007")
	}
}

func TestCreateNewInvoiceFromLastArchivedInvoice(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.Local)

	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	h := writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	olderArchivePath := filepath.Join(archiveDir, "2026-03-01.yaml")
	if err := os.WriteFile(olderArchivePath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-001
  issue_date: 2026-03-01
  due_date: 2026-03-31
  status: archived
  period: February 2026
  vat_percent: 19
  paid_amount: 500
positions:
  - name: Older position
    description: From older invoice
    unit_price: 50
    quantity: 1
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(olderArchivePath) returned error: %v", err)
	}

	latestArchivePath := filepath.Join(archiveDir, "2026-03-08.yaml")
	if err := os.WriteFile(latestArchivePath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-002
  issue_date: 2026-03-08
  due_date: 2026-04-07
  status: archived
  period: March 2026
  vat_percent: 10
  paid_amount: 999
positions:
  - name: Latest position
    description: From latest invoice
    unit_price: 120
    quantity: 2
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(latestArchivePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")
	created, err := h.CreateNewInvoice(
		now,
		t.TempDir(),
		defaultsPath,
		outputPath,
		customersPath,
		issuerPath,
		"CUST-001",
		true,
	)
	if err != nil {
		t.Fatalf("CreateNewInvoice returned error: %v", err)
	}
	if created.Number != "CUST-001-003" {
		t.Fatalf("invoiceNumber = %q, want %q", created.Number, "CUST-001-003")
	}

	source, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	text := string(source)
	for _, want := range []string{
		"number: CUST-001-003",
		"issue_date: \"2026-03-10\"",
		"due_date: \"2026-04-09\"",
		"status: draft",
		"paid_amount: \"0\"",
		"period: March 2026",
		"vat_percent: 10",
		"name: Latest position",
		"description: From latest invoice",
		"unit_price: 120",
		"quantity: 2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("created invoice does not contain %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{
		"Older position",
		"From older invoice",
		"status: archived",
		"paid_amount: 999",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("created invoice should not contain %q:\n%s", forbidden, text)
		}
	}
}

func TestCreateNewInvoiceFromLastRequiresArchivedInvoiceForCustomer(t *testing.T) {
	t.Parallel()

	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")
	_, err := h.CreateNewInvoice(
		time.Now(),
		t.TempDir(),
		defaultsPath,
		outputPath,
		customersPath,
		issuerPath,
		"CUST-001",
		true,
	)
	if err == nil {
		t.Fatal("CreateNewInvoice returned nil error without archived invoice match")
	}
	if !strings.Contains(err.Error(), "no archived invoice found for customer_id `CUST-001`") {
		t.Fatalf("error %q does not contain missing archived invoice message", err.Error())
	}
}

func TestCreateNewInvoiceFromLastRejectsLegacyArchivedInvoiceKeys(t *testing.T) {
	t.Parallel()

	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	archivePath := filepath.Join(archiveDir, "2026-03-08.yaml")
	if err := os.WriteFile(archivePath, []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-002
  issue_date: 2026-03-08
  due_date: 2026-04-07
  status: archived
  period_label: March 2026
  vat_rate_percent: 10
  paid_amount: 999
line_items:
  - name: Latest position
    description: From latest invoice
    unit_price: 120
    quantity: 2
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(archivePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.yaml")
	_, err := h.CreateNewInvoice(time.Now(), t.TempDir(), defaultsPath, outputPath, customersPath, issuerPath, "CUST-001", true)
	if err == nil {
		t.Fatal("CreateNewInvoice returned nil error for legacy archived invoice keys")
	}
	for _, want := range []string{
		archivePath + ":7: invoice.period_label: unsupported key; use invoice.period",
		archivePath + ":8: invoice.vat_rate_percent: unsupported key; use invoice.vat_percent",
		archivePath + ":10: line_items: unsupported key; use positions",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err.Error(), want)
		}
	}
}

func TestIncrementInvoiceNumberAdvancesCurrentInvoice(t *testing.T) {
	t.Parallel()

	customersPath, _, _ := writeDraftFixtures(t)
	invoicePath := writeMinimalInvoice(t, "CUST-001-009")
	archiveDir := t.TempDir()
	h := writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	writeArchivedInvoiceMarkdown(t, archiveDir, "2026-03-05.md", "CUST-001-011")

	incremented, err := h.IncrementInvoiceNumber(
		invoicePath,
		customersPath,
	)
	if err != nil {
		t.Fatalf("IncrementInvoiceNumber returned error: %v", err)
	}
	if incremented.CustomerID != "CUST-001" {
		t.Fatalf("customerID = %q, want %q", incremented.CustomerID, "CUST-001")
	}
	if incremented.OldNumber != "CUST-001-009" {
		t.Fatalf("oldNumber = %q, want %q", incremented.OldNumber, "CUST-001-009")
	}
	if incremented.NewNumber != "CUST-001-012" {
		t.Fatalf("newNumber = %q, want %q", incremented.NewNumber, "CUST-001-012")
	}

	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	if !strings.Contains(string(source), "number: CUST-001-012") {
		t.Fatalf("invoice file does not contain the incremented invoice number: %q", string(source))
	}
}

func TestResolveNumberingSettingsUsesConfigAndDefaults(t *testing.T) {
	t.Parallel()

	h := writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{year}-{counter:04}'\n  start: 5\n")

	settings, err := h.ResolveNumberingSettings()
	if err != nil {
		t.Fatalf("ResolveNumberingSettings returned error: %v", err)
	}
	if settings.Pattern != "{customer_id}-{year}-{counter:04}" {
		t.Fatalf("Pattern = %q, want %q", settings.Pattern, "{customer_id}-{year}-{counter:04}")
	}
	if settings.Start != 5 {
		t.Fatalf("Start = %d, want %d", settings.Start, 5)
	}
}
