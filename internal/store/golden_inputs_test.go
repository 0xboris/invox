package store

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update-golden", false, "rewrite testdata/golden/*/want-* from the current code")

// TestGoldenInputs pins how the fixtures in testdata/golden load and render.
// The want files were written by the code before typed models, so a change
// in any of them is a change in what invox does with an existing file.
func TestGoldenInputs(t *testing.T) {
	root := filepath.Join("testdata", "golden")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	templates := map[string]string{
		"want-all.tex":     filepath.Join(root, "all_placeholders.tex"),
		"want-starter.tex": filepath.Join("starter", "template.tex"),
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(root, name)
			customersPath := filepath.Join(dir, "customers.yaml")
			issuerPath := filepath.Join(dir, "issuer.yaml")
			invoicePath := filepath.Join(dir, "invoice.yaml")
			configFile, err := filepath.Abs(filepath.Join(root, "config.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			tmp := t.TempDir()
			h := NewHost(HostInputs{GOOS: "linux", Home: filepath.Join(tmp, "home"), XDGConfigHome: filepath.Join(tmp, "config"), ConfigFile: configFile})

			ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
			if err != nil {
				t.Fatalf("LoadContext: %v", err)
			}
			checkGolden(t, filepath.Join(dir, "want-summary.txt"), fmt.Sprintf(
				"customer_id=%s\nnumber=%s\ncurrency=%s\nemail=%s\nitems=%d\nsubtotal=%d\nvat=%d\ntotal=%d\npaid=%d\noutstanding=%d\n",
				ctx.CustomerID, ctx.InvoiceNumber, ctx.Currency, ctx.CustomerEmail, len(ctx.LineItems),
				ctx.SubtotalCents, ctx.VATAmountCents, ctx.TotalCents, ctx.PaidAmountCents, ctx.OutstandingCents,
			))

			for want, source := range templates {
				// A checkout may turn the templates' line endings into CRLF.
				text, err := os.ReadFile(source)
				if err != nil {
					t.Fatal(err)
				}
				templatePath := filepath.Join(tmp, filepath.Base(source))
				if err := os.WriteFile(templatePath, []byte(strings.ReplaceAll(string(text), "\r\n", "\n")), 0o644); err != nil {
					t.Fatal(err)
				}
				outputPath := filepath.Join(tmp, want)
				if err := h.RenderInvoice(templatePath, outputPath, ctx); err != nil {
					t.Fatalf("RenderInvoice(%s): %v", templatePath, err)
				}
				rendered, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatal(err)
				}
				checkGolden(t, filepath.Join(dir, want), string(rendered))
			}

			pdfPath := filepath.Join(tmp, "invoice.pdf")
			if err := os.WriteFile(pdfPath, []byte("%PDF"), 0o644); err != nil {
				t.Fatal(err)
			}
			var email string
			message, err := h.PrepareInvoiceEmail(EmailParams{CustomersPath: customersPath, IssuerPath: issuerPath, InvoicePath: invoicePath, PDFPath: pdfPath})
			if err != nil {
				email = "error: " + strings.ReplaceAll(err.Error(), dir+string(filepath.Separator), "") + "\n"
			} else {
				email = fmt.Sprintf("to=%s\nfrom=%s <%s>\nsubject=%s\n\n%s", message.Recipient, message.SenderName, message.SenderAddress, message.Subject, message.Body)
			}
			checkGolden(t, filepath.Join(dir, "want-email.txt"), email)
		})
	}
}

func checkGolden(t *testing.T, path, got string) {
	t.Helper()
	if *updateGolden {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("%s differs:\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}
