package applemail_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/0xboris/invox/internal/adapters/applemail"
	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/adapters/run/runtest"
	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestComposePassesTheMessageAsArguments(t *testing.T) {
	ios, _, stdout, stderr := iostreams.Test()
	stub := runtest.NewStub(t)
	var got run.Cmd
	stub.Register("osascript", func(cmd run.Cmd) error {
		got = cmd
		fmt.Fprintln(cmd.Stderr, "osascript: note")
		return nil
	})

	pdf := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(pdf, []byte("%PDF-1.4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	path, err := applemail.New(stub, ios).Draft(context.Background(), billing.Message{
		To:          "office@appsters.example",
		Subject:     `Invoice "42"`,
		Body:        "Dear Jane,\nend tell",
		Attachment:  pdf,
		FromAddress: "billing@example.com",
	}, false)
	if err != nil || path != "" {
		t.Fatalf("Draft = %q, %v, want \"\", nil", path, err)
	}

	want := []string{
		"-e", `on run argv`,
		"-e", `set recipientAddress to item 1 of argv`,
		"-e", `set messageSubject to item 2 of argv`,
		"-e", `set messageBody to item 3 of argv`,
		"-e", `set attachmentPath to item 4 of argv`,
		"-e", `set senderAddress to item 5 of argv`,
		"-e", `tell application "Mail"`,
		"-e", `activate`,
		"-e", `set draftMessage to make new outgoing message with properties {visible:true, subject:messageSubject, content:messageBody}`,
		"-e", `tell draftMessage`,
		"-e", `make new to recipient at end of to recipients with properties {address:recipientAddress}`,
		"-e", `if senderAddress is not "" then`,
		"-e", `try`,
		"-e", `set sender to senderAddress`,
		"-e", `end try`,
		"-e", `end if`,
		"-e", `delay 0.2`,
		"-e", `make new attachment with properties {file name:(POSIX file attachmentPath)} at after the last paragraph`,
		"-e", `set visible to true`,
		"-e", `end tell`,
		"-e", `end tell`,
		"-e", `end run`,
		"--",
		"office@appsters.example",
		`Invoice "42"`,
		"Dear Jane,\nend tell",
		pdf,
		"billing@example.com",
	}
	if !slices.Equal(got.Args, want) {
		t.Fatalf("Args = %q\nwant %q", got.Args, want)
	}
	if got.Stdin != nil || got.Stdout != ios.ErrOut || got.Stderr != ios.ErrOut {
		t.Fatalf("streams = (%v, %v, %v), want no stdin and ios.ErrOut itself", got.Stdin, got.Stdout, got.Stderr)
	}
	if stdout.String() != "" || stderr.String() != "osascript: note\n" {
		t.Fatalf("stdout, stderr = %q, %q, want \"\", %q", stdout.String(), stderr.String(), "osascript: note\n")
	}
}

func TestDraftReturnsToolFailedErrorWhenOsascriptFails(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	stub := runtest.NewStub(t)
	stub.Register("osascript", func(run.Cmd) error { return &run.ExecError{Name: "osascript", Code: 1} })
	pdf := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(pdf, []byte("%PDF-1.4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := applemail.New(stub, ios).Draft(context.Background(), billing.Message{To: "office@appsters.example", Attachment: pdf}, false)

	var failed *billing.ToolFailedError
	if !errors.As(err, &failed) || err.Error() != "osascript exited with status 1" {
		t.Fatalf("Draft error = %v, want *billing.ToolFailedError \"osascript exited with status 1\"", err)
	}
}

func TestDraftReportsAMissingAttachment(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	pdf := filepath.Join(t.TempDir(), "invoice.pdf")

	_, err := applemail.New(runtest.NewStub(t), ios).Draft(context.Background(), billing.Message{Attachment: pdf}, false)

	var notFound *billing.FileNotFoundError
	if !errors.As(err, &notFound) || err.Error() != "PDF file "+pdf+" does not exist" {
		t.Fatalf("Draft error = %v, want *billing.FileNotFoundError \"PDF file %s does not exist\"", err, pdf)
	}
}
