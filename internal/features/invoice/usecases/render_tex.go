package usecases

import (
	"github.com/0xboris/invox/internal/domain/invoice"
)

// RenderTeX fills the template at templatePath with inv and returns the TeX
// source. It writes nothing.
type RenderTeX struct {
	hosts *invoice.HostSource
}

func NewRenderTeX(hosts *invoice.HostSource) RenderTeX {
	return RenderTeX{hosts: hosts}
}

func (u RenderTeX) Run(templatePath string, inv *invoice.Context) (string, error) {
	return invoice.RenderTeX(u.hosts.Host().Files(), templatePath, inv)
}
