package usecases_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/adapters/fsutil"
	"github.com/0xboris/invox/internal/adapters/starter"
	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/features/invoice/usecases"
	"github.com/0xboris/invox/internal/testsupport"
	"github.com/0xboris/invox/internal/utilities/eml"
)

func newHosts(t *testing.T) *invoice.HostSource {
	host := invoice.NewHost(invoice.HostInputs{Files: fsutil.Files{}, Starter: starter.Files, GOOS: "linux", Home: t.TempDir(), ConfigDir: t.TempDir()})
	return invoice.NewHostSource(func(string) invoice.Host { return host })
}

func sampleMessage(t *testing.T) invoice.EmailMessage {
	pdf := filepath.Join(t.TempDir(), "2026-0001.pdf")
	if err := os.WriteFile(pdf, []byte("%PDF-1.4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return invoice.EmailMessage{
		Draft:         eml.Draft{Recipient: "office@example.com", Subject: "Invoice 2026-0001", Body: "Hello\n", SenderAddress: "hello@example.com", AttachmentPath: pdf},
		CustomerID:    "CUST-001",
		InvoiceNumber: "2026-0001",
	}
}

func TestOpenDraftKeepsTheDraftWhenTheOpenerFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "draft.eml")
	noApp := errors.New("no mail app")
	openDraft := usecases.NewOpenDraft(newHosts(t), testsupport.FailingOpener{Err: noApp})

	err := openDraft.Run(context.Background(), time.Unix(0, 0), sampleMessage(t), path, false)

	var openErr *usecases.OpenError
	if !errors.As(err, &openErr) || !errors.Is(err, noApp) {
		t.Fatalf("err = %v; want an *OpenError wrapping %v", err, noApp)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("draft not kept: %v", err)
	}
}

func TestOpenTempDraftRemovesItsDirectoryWhenTheOpenerFails(t *testing.T) {
	noApp := errors.New("no mail app")
	openTempDraft := usecases.NewOpenTempDraft(newHosts(t), testsupport.FailingOpener{Err: noApp})

	path, err := openTempDraft.Run(context.Background(), time.Now(), sampleMessage(t), "2026-0001.eml")

	var openErr *usecases.OpenError
	if !errors.As(err, &openErr) || !errors.Is(err, noApp) {
		t.Fatalf("err = %v; want an *OpenError wrapping %v", err, noApp)
	}
	if path == "" {
		t.Fatal("Run returned no draft path")
	}
	if _, err := os.Stat(filepath.Dir(path)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("draft directory %s still there: %v", filepath.Dir(path), err)
	}
}
