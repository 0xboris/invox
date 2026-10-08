// Package email renders the subject and body of an invoice email from
// their templates and builds the .eml draft that carries the PDF. It knows
// the placeholders and the MIME layout, not invoices: the caller fills in
// Fields.
package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"path/filepath"
	"time"
)

// Draft is an email with one PDF attachment, ready to write as .eml.
type Draft struct {
	Recipient     string
	Subject       string
	Body          string
	SenderName    string
	SenderAddress string
	// AttachmentPath names the PDF; only its base name goes into the draft.
	AttachmentPath string
}

// Build encodes d as a multipart/mixed message: the body quoted-printable,
// pdf base64 under the attachment's base name, dated now, marked unsent so
// mail clients open it for editing. boundary separates the parts.
func Build(d Draft, pdf []byte, now time.Time, boundary string) ([]byte, error) {
	fromAddress := mail.Address{
		Name:    d.SenderName,
		Address: d.SenderAddress,
	}
	toAddress := mail.Address{
		Address: d.Recipient,
	}

	var buffer bytes.Buffer

	fmt.Fprintf(&buffer, "From: %s\r\n", fromAddress.String())
	fmt.Fprintf(&buffer, "To: %s\r\n", toAddress.String())
	fmt.Fprintf(&buffer, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", d.Subject))
	fmt.Fprintf(&buffer, "Date: %s\r\n", now.Format(time.RFC1123Z))
	fmt.Fprintf(&buffer, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buffer, "X-Unsent: 1\r\n")
	fmt.Fprintf(&buffer, "Content-Type: multipart/mixed; boundary=%q\r\n", boundary)
	fmt.Fprintf(&buffer, "\r\n")

	writer := multipart.NewWriter(&buffer)
	if err := writer.SetBoundary(boundary); err != nil {
		return nil, err
	}

	textPart, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {`text/plain; charset="utf-8"`},
		"Content-Transfer-Encoding": {"quoted-printable"},
	})
	if err != nil {
		return nil, err
	}
	bodyWriter := quotedprintable.NewWriter(textPart)
	if _, err := bodyWriter.Write([]byte(d.Body)); err != nil {
		return nil, err
	}
	if err := bodyWriter.Close(); err != nil {
		return nil, err
	}

	filename := filepath.Base(d.AttachmentPath)
	attachmentPart, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {formatMIMEParameter("application/pdf", "name", filename)},
		"Content-Transfer-Encoding": {"base64"},
		"Content-Disposition":       {formatMIMEParameter("attachment", "filename", filename)},
	})
	if err != nil {
		return nil, err
	}
	if err := writeBase64MIME(attachmentPart, pdf); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// formatMIMEParameter keeps the plain quoted form for printable ASCII values
// without quotes or backslashes, and otherwise lets mime.FormatMediaType escape
// the value or RFC 2231-encode it.
func formatMIMEParameter(mediaType, key, value string) string {
	plain := true
	for i := 0; i < len(value); i++ {
		if c := value[i]; c < 0x20 || c > 0x7e || c == '"' || c == '\\' {
			plain = false
			break
		}
	}
	if plain {
		return fmt.Sprintf(`%s; %s="%s"`, mediaType, key, value)
	}
	return mime.FormatMediaType(mediaType, map[string]string{key: value})
}

func writeBase64MIME(buffer io.Writer, data []byte) error {
	encoded := base64.StdEncoding.EncodeToString(data)
	for len(encoded) > 76 {
		if _, err := buffer.Write([]byte(encoded[:76] + "\r\n")); err != nil {
			return err
		}
		encoded = encoded[76:]
	}
	if encoded != "" {
		if _, err := buffer.Write([]byte(encoded + "\r\n")); err != nil {
			return err
		}
	}
	return nil
}
