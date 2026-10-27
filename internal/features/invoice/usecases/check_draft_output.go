package usecases

import (
	"github.com/0xboris/invox/internal/domain/invoice"
)

// CheckDraftOutput checks that an email draft may be written to path: it
// returns an *invoice.OutputIsDirError for a directory, and an error matching
// fs.ErrExist for an existing file when overwrite is not set.
type CheckDraftOutput struct {
	hosts *invoice.HostSource
}

func NewCheckDraftOutput(hosts *invoice.HostSource) CheckDraftOutput {
	return CheckDraftOutput{hosts: hosts}
}

func (u CheckDraftOutput) Run(path string, overwrite bool) error {
	return invoice.CheckEmailDraftOutput(u.hosts.Host().Files(), path, overwrite)
}
