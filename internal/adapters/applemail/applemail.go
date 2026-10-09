// Package applemail opens an editable compose window in Apple Mail.
package applemail

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/iostreams"
)

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

// Draft opens m in an Apple Mail compose window with the PDF attached. It
// implements billing.Mailer; the draft has no file. An empty m.FromAddress
// leaves the sender to Mail's default account. It returns a
// *billing.ToolFailedError when osascript fails.
func (c *Composer) Draft(ctx context.Context, m billing.Message, dryRun bool) (string, error) {
	if _, err := os.Stat(m.Attachment); errors.Is(err, fs.ErrNotExist) {
		return "", &billing.FileNotFoundError{File: billing.PDFFile, Path: m.Attachment}
	} else if err != nil {
		return "", fmt.Errorf("read %s: %w", m.Attachment, err)
	}
	if dryRun {
		return "", nil
	}
	args := make([]string, 0, 2*len(script)+6)
	for _, line := range script {
		args = append(args, "-e", line)
	}
	args = append(args, "--", m.To, m.Subject, m.Body, m.Attachment, m.FromAddress)
	err := c.runner.Run(ctx, run.Cmd{
		Name:   "osascript",
		Args:   args,
		Stdout: c.ios.ErrOut,
		Stderr: c.ios.ErrOut,
	})
	var execErr *run.ExecError
	if errors.As(err, &execErr) {
		return "", &billing.ToolFailedError{Tool: "osascript", Code: execErr.Code, Err: err}
	}
	return "", err
}
