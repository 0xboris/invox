package email

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"path/filepath"
	"strings"
	"testing"
)

// parseDraftParts parses an .eml produced by build and returns
// the text body's transfer encoding, the decoded text body and the attachment's filename as Go's parsers see them.
func parseDraftParts(t *testing.T, eml []byte) (encoding, body, contentTypeName, dispositionFilename string) {
	t.Helper()
	message, err := mail.ReadMessage(bytes.NewReader(eml))
	if err != nil {
		t.Fatalf("mail.ReadMessage returned error: %v", err)
	}
	mediaType, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/mixed" {
		t.Fatalf("Content-Type = %q, %v; want multipart/mixed", mediaType, err)
	}
	reader := multipart.NewReader(message.Body, params["boundary"])

	textPart, err := reader.NextRawPart()
	if err != nil {
		t.Fatalf("reading text part returned error: %v", err)
	}
	encoding = textPart.Header.Get("Content-Transfer-Encoding")
	var bodyReader io.Reader = textPart
	if encoding == "quoted-printable" {
		bodyReader = quotedprintable.NewReader(textPart)
	}
	decoded, err := io.ReadAll(bodyReader)
	if err != nil {
		t.Fatalf("reading text body returned error: %v", err)
	}

	attachmentPart, err := reader.NextRawPart()
	if err != nil {
		t.Fatalf("reading attachment part returned error: %v", err)
	}
	_, typeParams, err := mime.ParseMediaType(attachmentPart.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parsing attachment Content-Type %q returned error: %v", attachmentPart.Header.Get("Content-Type"), err)
	}
	_, dispositionParams, err := mime.ParseMediaType(attachmentPart.Header.Get("Content-Disposition"))
	if err != nil {
		t.Fatalf("parsing Content-Disposition %q returned error: %v", attachmentPart.Header.Get("Content-Disposition"), err)
	}
	return encoding, string(decoded), typeParams["name"], dispositionParams["filename"]
}

func TestBuildInvoiceEmailDraftBodyRoundTripsUTF8(t *testing.T) {
	body := "Sehr geehrte Frau Müller,\n\nanbei die Rechnung über 1.234,56 € (Größe: ½).\n" +
		"A long line that goes on and on and on, well past seventy-six characters, to check soft line breaks = fine.\n\nGrüße,\nJürgen\n"
	eml, err := build(draft{
		Recipient:      "office@example.com",
		Subject:        "Rechnung",
		Body:           body,
		SenderName:     "Jürgen",
		SenderAddress:  "hello@example.com",
		AttachmentPath: "invoice.pdf",
	}, []byte("%PDF-1.4\nfake"), draftTime, draftBoundary)
	if err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	encoding, got, _, _ := parseDraftParts(t, eml)
	if encoding != "quoted-printable" {
		t.Fatalf("text part Content-Transfer-Encoding = %q, want quoted-printable for a UTF-8 body", encoding)
	}
	want := strings.ReplaceAll(body, "\n", "\r\n")
	if got != want {
		t.Fatalf("decoded body = %q, want %q", got, want)
	}
}

func TestBuildInvoiceEmailDraftAttachmentFilenameRoundTrips(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		header   string
	}{
		{name: "ASCII", filename: "BL00210001.pdf", header: `filename="BL00210001.pdf"`},
		{name: "spaces and quotes", filename: `Invoice "May" 2026.pdf`, header: `filename="Invoice \"May\" 2026.pdf"`},
		{name: "non-ASCII", filename: "Rechnung Müller €.pdf", header: "filename*=utf-8''Rechnung%20M%C3%BCller%20%E2%82%AC.pdf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eml, err := build(draft{
				Recipient:      "office@example.com",
				Subject:        "Invoice",
				Body:           "Hello\n",
				SenderAddress:  "hello@example.com",
				AttachmentPath: filepath.Join(t.TempDir(), tt.filename),
			}, []byte("%PDF-1.4\nfake"), draftTime, draftBoundary)
			if err != nil {
				t.Fatalf("build returned error: %v", err)
			}
			_, _, name, filename := parseDraftParts(t, eml)
			if name != tt.filename {
				t.Fatalf("Content-Type name = %q, want %q", name, tt.filename)
			}
			if filename != tt.filename {
				t.Fatalf("Content-Disposition filename = %q, want %q", filename, tt.filename)
			}
			if tt.header != "" && !bytes.Contains(eml, []byte(tt.header)) {
				t.Fatalf("draft does not contain %q:\n%s", tt.header, eml)
			}
		})
	}
}
