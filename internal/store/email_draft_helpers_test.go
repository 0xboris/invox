package store

import "time"

// createEmailDraft prepares the email for params and writes its draft, as
// invox email does.
func createEmailDraft(h Host, now time.Time, params EmailParams, outputPath string, overwrite bool) (EmailDraftResult, error) {
	message, err := h.PrepareInvoiceEmail(params)
	if err != nil {
		return EmailDraftResult{}, err
	}
	return h.CreateInvoiceEmailDraft(now, message, outputPath, overwrite)
}
