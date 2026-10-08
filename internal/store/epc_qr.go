package store

import (
	"errors"
	"fmt"
	"strings"

	"github.com/0xboris/invox/internal/epc"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/money"
)

func epcQRCodeEligible(ctx *invoice.Context) bool {
	return ctx != nil && ctx.OutstandingCents > 0 && strings.TrimSpace(ctx.Currency) == "EUR"
}

func buildEPCPayload(ctx *invoice.Context) ([]byte, error) {
	if strings.TrimSpace(ctx.Currency) != "EUR" {
		return nil, fmt.Errorf("EPC QR code requires billing.currency EUR, got `%s`", ctx.Currency)
	}
	// EPC amounts run from 0.01 to 999999999.99.
	if ctx.OutstandingCents <= 0 {
		return nil, errors.New("invoice.outstanding_amount: EPC QR code requires an amount above zero")
	}
	if ctx.OutstandingCents > ctx.TotalCents {
		return nil, fmt.Errorf("invoice.outstanding_amount: `%s` exceeds total `%s`", money.FormatCents(ctx.OutstandingCents), money.FormatCents(ctx.TotalCents))
	}
	if ctx.OutstandingCents > epc.MaxAmountCents {
		return nil, fmt.Errorf("invoice.outstanding_amount: `%s` exceeds EPC QR maximum `%s`", money.FormatCents(ctx.OutstandingCents), "999999999,99")
	}

	name := ctx.Payment.EPCQR.Name.Trim()
	if name == "" {
		name = ctx.Company.LegalCompanyName.Trim()
	}
	if name == "" {
		return nil, errors.New("issuer.payment.epc_qr.name: missing value")
	}

	iban := epc.CompactIdentifier(string(ctx.Payment.IBAN))
	if !epc.ValidIBAN(iban) {
		return nil, fmt.Errorf("issuer.payment.iban: invalid IBAN `%s`", ctx.Payment.IBAN)
	}
	if !epc.SEPASchemeIBAN(iban) {
		return nil, fmt.Errorf("issuer.payment.iban: IBAN `%s` is outside the current SEPA scheme scope", ctx.Payment.IBAN)
	}

	bic := epc.CompactIdentifier(string(ctx.Payment.BIC))
	if bic != "" && !epc.ValidBIC(bic) {
		return nil, fmt.Errorf("issuer.payment.bic: invalid BIC `%s`", ctx.Payment.BIC)
	}

	purpose := strings.ToUpper(ctx.Payment.EPCQR.Purpose.Trim())
	if purpose != "" && !epc.ValidPurpose(purpose) {
		return nil, fmt.Errorf("issuer.payment.epc_qr.purpose: expected 1-4 letters or digits, got `%s`", purpose)
	}

	text := ctx.Payment.EPCQR.Text.Trim()
	if text == "" {
		text = ctx.Header.Number.Trim()
	}
	information := ctx.Payment.EPCQR.Information.Trim()

	for _, field := range []struct {
		label    string
		value    string
		maxChars int
	}{
		{label: "issuer.payment.epc_qr.name", value: name, maxChars: epc.MaxNameChars},
		{label: "issuer.payment.epc_qr.text", value: text, maxChars: epc.MaxTextChars},
		{label: "issuer.payment.epc_qr.information", value: information, maxChars: epc.MaxInfoChars},
	} {
		if err := epc.CheckText(field.value, field.maxChars); err != nil {
			return nil, fmt.Errorf("%s: %w", field.label, err)
		}
	}

	return epc.Transfer{
		BIC:         bic,
		Name:        name,
		IBAN:        iban,
		AmountCents: ctx.OutstandingCents,
		Purpose:     purpose,
		Text:        text,
		Information: information,
	}.Encode()
}
