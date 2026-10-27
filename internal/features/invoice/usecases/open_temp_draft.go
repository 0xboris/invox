package usecases

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/domain/ports"
)

const (
	draftDirPrefix = "invox-email-"
	// draftMaxAge is how long a temporary draft stays for the mail app
	// before a later run of invox email removes it.
	draftMaxAge = 24 * time.Hour
)

// OpenTempDraft writes an email draft named name into a new temporary
// directory and opens it. The draft stays there, because the mail app can
// read it after the opener returns, so invox cannot know when to delete it.
// Each run first removes the drafts that are older than draftMaxAge.
type OpenTempDraft struct {
	hosts  *invoice.HostSource
	opener ports.Opener
}

func NewOpenTempDraft(hosts *invoice.HostSource, opener ports.Opener) OpenTempDraft {
	return OpenTempDraft{hosts: hosts, opener: opener}
}

// Run returns the draft's path. When writing or opening the draft fails, it
// removes the directory again; an open that fails returns an *OpenError.
func (u OpenTempDraft) Run(ctx context.Context, now time.Time, message invoice.EmailMessage, name string) (string, error) {
	h := u.hosts.Host()
	files := h.Files()
	pruneEmailDrafts(files, files.TempDir(), now.Add(-draftMaxAge))
	dir, err := files.MkdirTemp("", draftDirPrefix+"*")
	if err != nil {
		return "", fmt.Errorf("create temporary draft directory: %w", err)
	}
	path := filepath.Join(dir, name)
	if _, err := h.CreateInvoiceEmailDraft(now, message, path, false); err != nil {
		_ = files.RemoveAll(dir)
		return path, err
	}
	if err := u.opener.Open(ctx, path); err != nil {
		_ = files.RemoveAll(dir)
		return path, &OpenError{Err: err}
	}
	return path, nil
}

// pruneEmailDrafts removes the temporary draft directories in dir that earlier
// runs left behind and that were last modified before cutoff. It skips
// anything that is not a directory, so a symlink is never followed.
func pruneEmailDrafts(files ports.Files, dir string, cutoff time.Time) {
	entries, err := files.ReadDir(dir)
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
		_ = files.RemoveAll(filepath.Join(dir, entry.Name()))
	}
}
