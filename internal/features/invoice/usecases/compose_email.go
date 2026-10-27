package usecases

import (
	"context"

	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/domain/ports"
)

// ComposeEmail opens an editable compose window in the mail app, with the
// invoice PDF attached.
type ComposeEmail struct {
	// mailer is nil where no mail app can compose.
	mailer ports.MailComposer
}

func NewComposeEmail(mailer ports.MailComposer) ComposeEmail {
	return ComposeEmail{mailer: mailer}
}

// Available reports whether a mail app can compose here. Without one, the
// email command writes an .eml draft instead.
func (u ComposeEmail) Available() bool {
	return u.mailer != nil
}

func (u ComposeEmail) Run(ctx context.Context, message invoice.EmailMessage) error {
	return u.mailer.Compose(ctx, ports.MailMessage{
		To:         message.Recipient,
		Subject:    message.Subject,
		Body:       message.Body,
		Attachment: message.AttachmentPath,
		Sender:     message.SenderAddress,
	})
}
