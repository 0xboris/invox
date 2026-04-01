package cli

import (
	"encoding/json"
	"io"

	"invox/internal/invoice"
)

type customerListJSONItem struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type templateListJSONItem struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type archiveListJSONItem struct {
	Filename   string `json:"filename"`
	CustomerID string `json:"customer_id"`
	IssueDate  string `json:"issue_date"`
	Status     string `json:"status"`
}

type validateJSONResult struct {
	OK               bool   `json:"ok"`
	InvoiceNumber    string `json:"invoice_number"`
	CustomerID       string `json:"customer_id"`
	LineItemCount    int    `json:"line_item_count"`
	Currency         string `json:"currency"`
	SubtotalCents    int64  `json:"subtotal_cents"`
	VATAmountCents   int64  `json:"vat_amount_cents"`
	TotalCents       int64  `json:"total_cents"`
	PaidAmountCents  int64  `json:"paid_amount_cents"`
	OutstandingCents int64  `json:"outstanding_cents"`
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func customerListJSON(customers []invoice.CustomerSummary) []customerListJSONItem {
	items := make([]customerListJSONItem, 0, len(customers))
	for _, customer := range customers {
		items = append(items, customerListJSONItem{
			ID:     customer.ID,
			Name:   customer.LegalCompanyName,
			Status: customer.Status,
		})
	}
	return items
}

func templateListJSON(templates []invoice.TemplateSummary) []templateListJSONItem {
	items := make([]templateListJSONItem, 0, len(templates))
	for _, template := range templates {
		items = append(items, templateListJSONItem{
			Name: template.Name,
			Path: template.Path,
		})
	}
	return items
}

func archiveListJSON(archivedInvoices []invoice.ArchivedInvoiceSummary) []archiveListJSONItem {
	items := make([]archiveListJSONItem, 0, len(archivedInvoices))
	for _, archivedInvoice := range archivedInvoices {
		items = append(items, archiveListJSONItem{
			Filename:   archivedInvoice.Filename,
			CustomerID: archivedInvoice.CustomerID,
			IssueDate:  archivedInvoice.IssueDate,
			Status:     archivedInvoice.Status,
		})
	}
	return items
}

func validateJSON(ctx *invoice.Context) validateJSONResult {
	return validateJSONResult{
		OK:               true,
		InvoiceNumber:    ctx.InvoiceNumber,
		CustomerID:       ctx.CustomerID,
		LineItemCount:    len(ctx.LineItems),
		Currency:         ctx.Currency,
		SubtotalCents:    ctx.SubtotalCents,
		VATAmountCents:   ctx.VATAmountCents,
		TotalCents:       ctx.TotalCents,
		PaidAmountCents:  ctx.PaidAmountCents,
		OutstandingCents: ctx.OutstandingCents,
	}
}
