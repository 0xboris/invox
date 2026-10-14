package ports

import (
	"context"
)

// PDFCompiler turns a .tex file into a PDF next to it.
type PDFCompiler interface {
	Build(ctx context.Context, texPath string) error
}

// Editor opens a file in the user's editor and returns when it exits.
type Editor interface {
	Edit(ctx context.Context, path string) error
}

// Opener opens a file with the program the OS associates with it.
type Opener interface {
	Open(ctx context.Context, path string) error
}

// MailComposer opens a new email in the user's mail program.
type MailComposer interface {
	Compose(ctx context.Context, msg MailMessage) error
}

// MailMessage is the email to compose.
type MailMessage struct {
	To         string
	Subject    string
	Body       string
	Attachment string
	// Sender is the From address, or empty for the mail program's default
	// account.
	Sender string
}

// CompilerNotInstalledError means the PDF compiler, tectonic, is not on
// PATH.
type CompilerNotInstalledError struct{}

func (e *CompilerNotInstalledError) Error() string { return "tectonic not found in PATH" }
