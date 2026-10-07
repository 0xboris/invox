package invoice

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var (
	emlDateHeader = regexp.MustCompile(`(?m)^Date: [^\r]*\r$`)
	emlBoundary   = regexp.MustCompile(`invox-boundary-[0-9]+`)
)

func TestBuildInvoiceEmailDraftMatchesGolden(t *testing.T) {
	eml, err := buildInvoiceEmailDraft(EmailMessage{
		Recipient:      "office@example.com",
		Subject:        "Rechnung 2026-0001 für März",
		Body:           "Sehr geehrte Frau Müller,\n\nanbei die Rechnung.\n\nGrüße,\nJürgen\n",
		SenderName:     "Jürgen Beispiel",
		SenderAddress:  "hello@example.com",
		AttachmentPath: filepath.Join("out", "2026-0001.pdf"),
	}, []byte("%PDF-1.4\nfake pdf bytes\n"))
	if err != nil {
		t.Fatalf("buildInvoiceEmailDraft returned error: %v", err)
	}

	got := emlBoundary.ReplaceAll(emlDateHeader.ReplaceAll(eml, []byte("Date: DATE\r")), []byte("invox-boundary-BOUNDARY"))
	want, err := os.ReadFile(filepath.Join("testdata", "email", "draft.eml"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("eml = %q\nwant %q", got, want)
	}
}
