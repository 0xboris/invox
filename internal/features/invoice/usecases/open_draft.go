package usecases

import (
	"context"
	"time"

	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/domain/ports"
)

// OpenDraft writes an email draft to path as an .eml file and opens it.
type OpenDraft struct {
	hosts  *invoice.HostSource
	opener ports.Opener
}

func NewOpenDraft(hosts *invoice.HostSource, opener ports.Opener) OpenDraft {
	return OpenDraft{hosts: hosts, opener: opener}
}

// OpenError reports that the draft was written but could not be opened.
type OpenError struct {
	Err error
}

func (e *OpenError) Error() string { return e.Err.Error() }
func (e *OpenError) Unwrap() error { return e.Err }

// Run replaces an existing file at path only when overwrite is set. An
// open that fails returns an *OpenError and keeps the draft.
func (u OpenDraft) Run(ctx context.Context, now time.Time, message invoice.EmailMessage, path string, overwrite bool) error {
	if _, err := u.hosts.Host().CreateInvoiceEmailDraft(now, message, path, overwrite); err != nil {
		return err
	}
	if err := u.opener.Open(ctx, path); err != nil {
		return &OpenError{Err: err}
	}
	return nil
}
