package email

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/fsutil"
)

const (
	draftDirPrefix = "invox-email-"
	// draftMaxAge is how long a temporary draft stays for the mail app
	// before a later draft removes it.
	draftMaxAge = 24 * time.Hour
)

// Mailer writes drafts as .eml files. It implements billing.Mailer.
type Mailer struct{}

var _ billing.Mailer = Mailer{}

// Check returns an *billing.OutputIsDirError when m.Output is a directory,
// and an error matching fs.ErrExist when the draft would replace an
// existing file and m.Overwrite is not set.
func (Mailer) Check(m billing.Message) error {
	return checkOutput(m.Output, m.Overwrite)
}

// Draft writes m as an .eml file: to m.Output, or, for a temporary draft,
// to a new temporary directory that a draft a day later removes. Unless
// m.Overwrite is set, an existing file is left untouched and the error
// matches fs.ErrExist.
func (Mailer) Draft(_ context.Context, m billing.Message) (billing.Draft, error) {
	if !m.Temporary {
		return billing.Draft{Path: m.Output}, write(m, m.Output, m.Overwrite)
	}
	pruneDrafts(os.TempDir(), m.Date.Add(-draftMaxAge))
	dir, err := os.MkdirTemp("", draftDirPrefix+"*")
	if err != nil {
		return billing.Draft{}, fmt.Errorf("create temporary draft directory: %w", err)
	}
	discard := func() { _ = os.RemoveAll(dir) }
	path := filepath.Join(dir, filepath.Base(m.Output))
	if err := write(m, path, false); err != nil {
		discard()
		return billing.Draft{}, err
	}
	// The draft stays in its temporary directory: the mail app can read it
	// after the opener returns, so invox cannot know when to delete it.
	return billing.Draft{Path: path, Discard: discard}, nil
}

func write(m billing.Message, path string, overwrite bool) error {
	if err := checkOutput(path, overwrite); err != nil {
		return err
	}
	pdf, err := os.ReadFile(m.Attachment)
	if err != nil {
		return fmt.Errorf("read %s: %w", m.Attachment, err)
	}
	message, err := Build(Draft{
		Recipient:      m.To,
		Subject:        m.Subject,
		Body:           m.Body,
		SenderName:     m.FromName,
		SenderAddress:  m.FromAddress,
		AttachmentPath: m.Attachment,
	}, pdf, m.Date, fmt.Sprintf("invox-boundary-%d", m.Date.UnixNano()))
	if err != nil {
		return err
	}
	writeFile := fsutil.WriteNewFile
	if overwrite {
		writeFile = fsutil.WriteFile
	}
	return writeFile(path, message, fsutil.Public)
}

func checkOutput(path string, overwrite bool) error {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return &billing.OutputIsDirError{Path: path}
	}
	if overwrite {
		return nil
	}
	if _, err := os.Lstat(path); err == nil {
		return &fs.PathError{Op: "write", Path: path, Err: fs.ErrExist}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// pruneDrafts removes the temporary draft directories in dir that earlier
// runs left behind and that were last modified before cutoff. It skips
// anything that is not a directory, so a symlink is never followed.
func pruneDrafts(dir string, cutoff time.Time) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), draftDirPrefix) {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.ModTime().Before(cutoff) {
			continue
		}
		_ = os.RemoveAll(filepath.Join(dir, entry.Name()))
	}
}
