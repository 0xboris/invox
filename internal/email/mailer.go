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

// Mailer writes drafts as .eml files and opens them. It implements
// billing.Mailer.
type Mailer struct {
	// Open opens a draft in the user's mail app.
	Open func(ctx context.Context, path string) error
}

var _ billing.Mailer = Mailer{}

// Draft writes m as an .eml file and opens it: to m.Output, or, for a
// temporary draft, to a new temporary directory that a draft a day later
// removes. It returns an *billing.OutputIsDirError when m.Output is a
// directory and, unless m.Overwrite is set, leaves an existing file
// untouched with an error matching fs.ErrExist. With dryRun it runs the
// same checks and writes nothing.
func (mailer Mailer) Draft(ctx context.Context, m billing.Message, dryRun bool) (string, error) {
	if _, err := os.Stat(m.Attachment); err != nil {
		return "", fmt.Errorf("read %s: %w", m.Attachment, err)
	}
	if dryRun {
		if m.Temporary {
			return "", nil
		}
		return "", checkOutput(m.Output, m.Overwrite)
	}
	if !m.Temporary {
		if err := write(m, m.Output, m.Overwrite); err != nil {
			return "", err
		}
		if err := mailer.Open(ctx, m.Output); err != nil {
			return "", fmt.Errorf("created %s but failed to open it: %w", m.Output, err)
		}
		return m.Output, nil
	}
	pruneDrafts(os.TempDir(), m.Date.Add(-draftMaxAge))
	dir, err := os.MkdirTemp("", draftDirPrefix+"*")
	if err != nil {
		return "", fmt.Errorf("create temporary draft directory: %w", err)
	}
	path := filepath.Join(dir, filepath.Base(m.Output))
	if err := write(m, path, false); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	// The draft stays in its temporary directory: the mail app can read it
	// after the opener returns, so invox cannot know when to delete it.
	if err := mailer.Open(ctx, path); err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("failed to open email draft: %w", err)
	}
	return path, nil
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
