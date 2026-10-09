package email

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

var (
	draftTime     = time.Date(2026, 3, 6, 9, 30, 0, 0, time.FixedZone("CET", 60*60))
	draftBoundary = "invox-boundary-1772785800000000000"
)

func TestBuildInvoiceEmailDraftMatchesGolden(t *testing.T) {
	t.Parallel()

	eml, err := build(draft{
		Recipient:      "office@example.com",
		Subject:        "Rechnung 2026-0001 für März",
		Body:           "Sehr geehrte Frau Müller,\n\nanbei die Rechnung.\n\nGrüße,\nJürgen\n",
		SenderName:     "Jürgen Beispiel",
		SenderAddress:  "hello@example.com",
		AttachmentPath: filepath.Join("out", "2026-0001.pdf"),
	}, []byte("%PDF-1.4\nfake pdf bytes\n"), draftTime, draftBoundary)
	if err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "draft.eml"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(eml) != string(want) {
		t.Fatalf("eml = %q\nwant %q", eml, want)
	}
}
