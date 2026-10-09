// Package billing holds invox's use cases. Each is a method on Service,
// which reaches files, programs and the mail app only through the
// interfaces in ports.go.
package billing

import (
	"time"

	"github.com/0xboris/invox/internal/numbering"
)

// Settings are the parts of config.yaml the use cases read.
type Settings struct {
	// File is the config file, named in errors about its settings; "" when
	// there is none.
	File string
	// Numbering has the defaults filled in. It is checked when an invoice
	// is numbered, not before.
	Numbering    numbering.Settings
	EmailSubject string
	EmailBody    string
}

// Service runs the use cases.
type Service struct {
	Invoices  Invoices
	Directory Directory
	Archives  Archive // not "Archive": Service has an Archive method
	Renderer  Renderer
	Compiler  Compiler
	Mailer    Mailer
	// Settings reads config.yaml; it is called only by the use cases that
	// need a setting.
	Settings func() (Settings, error)
	Now      func() time.Time
}
