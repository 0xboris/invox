package email

import (
	"bytes"
	"encoding/base64"
	"mime"
	"mime/multipart"
	"net/mail"
	"runtime"
	"strings"
	"testing"
)

func TestBuildWrapsTheAttachmentAt76Characters(t *testing.T) {
	pdf := make([]byte, 200)
	for i := range pdf {
		pdf[i] = byte(i)
	}
	eml, err := build(draft{
		Recipient:      "office@example.com",
		Subject:        "Invoice",
		Body:           "Hello\n",
		SenderAddress:  "hello@example.com",
		AttachmentPath: "invoice.pdf",
	}, pdf, draftTime, draftBoundary)
	if err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	message, err := mail.ReadMessage(bytes.NewReader(eml))
	if err != nil {
		t.Fatalf("mail.ReadMessage returned error: %v", err)
	}
	_, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parsing Content-Type returned error: %v", err)
	}
	reader := multipart.NewReader(message.Body, params["boundary"])
	if _, err := reader.NextRawPart(); err != nil {
		t.Fatalf("reading text part returned error: %v", err)
	}
	attachment, err := reader.NextRawPart()
	if err != nil {
		t.Fatalf("reading attachment part returned error: %v", err)
	}
	var raw bytes.Buffer
	if _, err := raw.ReadFrom(attachment); err != nil {
		t.Fatalf("reading attachment returned error: %v", err)
	}

	lines := strings.Split(strings.TrimRight(raw.String(), "\r\n"), "\r\n")
	wantLengths := []int{76, 76, 76, 40}
	if len(lines) != len(wantLengths) {
		t.Fatalf("attachment has %d lines, want %d: %q", len(lines), len(wantLengths), lines)
	}
	for i, line := range lines {
		if len(line) != wantLengths[i] {
			t.Fatalf("attachment line %d is %d characters, want %d", i+1, len(line), wantLengths[i])
		}
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.Join(lines, ""))
	if err != nil || !bytes.Equal(decoded, pdf) {
		t.Fatalf("attachment decodes to %d bytes (%v), want the 200-byte PDF", len(decoded), err)
	}
}

func TestBuildQuotesABackslashInTheAttachmentName(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a backslash separates path elements on Windows")
	}
	const filename = `Invoice\(2026).pdf`
	eml, err := build(draft{
		Recipient:      "office@example.com",
		Subject:        "Invoice",
		Body:           "Hello\n",
		SenderAddress:  "hello@example.com",
		AttachmentPath: "/tmp/" + filename,
	}, []byte("%PDF-1.4\nfake"), draftTime, draftBoundary)
	if err != nil {
		t.Fatalf("build returned error: %v", err)
	}
	_, _, name, dispositionFilename := parseDraftParts(t, eml)
	if name != filename || dispositionFilename != filename {
		t.Fatalf("attachment name = %q, filename = %q; want %q for both", name, dispositionFilename, filename)
	}
}
