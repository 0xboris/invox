// Package applemail opens an editable compose window in Apple Mail.
package applemail

import (
	"context"
	"fmt"
	"os"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/iostreams"
)

// Message is the email to compose.
type Message struct {
	To         string
	Subject    string
	Body       string
	Attachment string
	// Sender is the From address, or empty for Mail's default account.
	Sender string
}

// Composer runs an AppleScript through osascript. Its output goes to stderr,
// so stdout carries only the data a command prints.
type Composer struct {
	runner run.Runner
	ios    *iostreams.IOStreams
}

// New returns a Composer that runs osascript with runner.
func New(runner run.Runner, ios *iostreams.IOStreams) *Composer {
	return &Composer{runner: runner, ios: ios}
}

var script = []string{
	`on run argv`,
	`set recipientAddress to item 1 of argv`,
	`set messageSubject to item 2 of argv`,
	`set messageBody to item 3 of argv`,
	`set attachmentPath to item 4 of argv`,
	`set senderAddress to item 5 of argv`,
	`tell application "Mail"`,
	`activate`,
	`set draftMessage to make new outgoing message with properties {visible:true, subject:messageSubject, content:messageBody}`,
	`tell draftMessage`,
	`make new to recipient at end of to recipients with properties {address:recipientAddress}`,
	`if senderAddress is not "" then`,
	`try`,
	`set sender to senderAddress`,
	`end try`,
	`end if`,
	`delay 0.2`,
	`make new attachment with properties {file name:(POSIX file attachmentPath)} at after the last paragraph`,
	`set visible to true`,
	`end tell`,
	`end tell`,
	`end run`,
}

// Compose opens msg in a new Mail compose window with its attachment.
func (c *Composer) Compose(ctx context.Context, msg Message) error {
	args := make([]string, 0, 2*len(script)+6)
	for _, line := range script {
		args = append(args, "-e", line)
	}
	args = append(args, "--", msg.To, msg.Subject, msg.Body, msg.Attachment, msg.Sender)
	return c.runner.Run(ctx, run.Cmd{
		Name:   "osascript",
		Args:   args,
		Stdout: c.ios.ErrOut,
		Stderr: c.ios.ErrOut,
	})
}

// Draft opens m in an Apple Mail compose window with the PDF attached. It
// implements billing.Mailer; the draft has no file.
func (c *Composer) Draft(ctx context.Context, m billing.Message, dryRun bool) (string, error) {
	if _, err := os.Stat(m.Attachment); err != nil {
		return "", fmt.Errorf("read %s: %w", m.Attachment, err)
	}
	if dryRun {
		return "", nil
	}
	err := c.Compose(ctx, Message{
		To:         m.To,
		Subject:    m.Subject,
		Body:       m.Body,
		Attachment: m.Attachment,
		Sender:     m.FromAddress,
	})
	if err != nil {
		return "", fmt.Errorf("failed to open editable email draft: %w", err)
	}
	return "", nil
}
