package billing_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/billing"
)

func TestLoadContextValidatesPaidAmount(t *testing.T) {
	// The context fixture totals 252.00.
	tests := []struct {
		name            string
		paidAmount      string
		wantErr         string
		wantOutstanding int64
		wantEPCAmount   string
	}{
		{name: "negative", paidAmount: "-500", wantErr: "invoice.paid_amount: must not be negative"},
		{name: "negative cent", paidAmount: "-0.01", wantErr: "invoice.paid_amount: must not be negative"},
		{name: "zero", paidAmount: "0", wantOutstanding: 25200, wantEPCAmount: "EUR252.00"},
		{name: "partial", paidAmount: "52", wantOutstanding: 20000, wantEPCAmount: "EUR200.00"},
		{name: "equal to total", paidAmount: "252", wantOutstanding: 0},
		{name: "greater than total", paidAmount: "252.01", wantErr: "exceeds total"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
			source, err := os.ReadFile(invoicePath)
			if err != nil {
				t.Fatalf("ReadFile(invoice.yaml) returned error: %v", err)
			}
			mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: "+tt.paidAmount, 1)
			mutatedPath := filepath.Join(t.TempDir(), "invoice.yaml")
			if err := os.WriteFile(mutatedPath, []byte(mutated), 0o644); err != nil {
				t.Fatalf("WriteFile(invoice.yaml) returned error: %v", err)
			}

			ctx, err := loadContext(t, customersPath, issuerPath, mutatedPath)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("LoadContext returned nil error, want %q (outstanding %d)", tt.wantErr, ctx.OutstandingCents)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadContext returned error: %v", err)
			}
			if ctx.OutstandingCents != tt.wantOutstanding {
				t.Fatalf("OutstandingCents = %d, want %d", ctx.OutstandingCents, tt.wantOutstanding)
			}
			if ctx.OutstandingCents > ctx.TotalCents {
				t.Fatalf("OutstandingCents %d exceeds TotalCents %d", ctx.OutstandingCents, ctx.TotalCents)
			}

			if code := billing.EPCFor(ctx); code.Payload == nil && code.Err == nil {
				if tt.wantEPCAmount != "" {
					t.Fatalf("epcQRCodeEligible = false, want an EPC payload with %s", tt.wantEPCAmount)
				}
				return
			}
			payload, err := billing.EPCPayload(ctx)
			if err != nil {
				t.Fatalf("buildEPCPayload returned error: %v", err)
			}
			lines := strings.Split(string(payload), "\n")
			if len(lines) < 8 || lines[7] != tt.wantEPCAmount {
				t.Fatalf("EPC payload amount line = %q, want %q", lines, tt.wantEPCAmount)
			}
		})
	}
}

func TestBuildEPCPayloadRejectsOutstandingAboveTotal(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	ctx, err := loadContext(t, customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}
	ctx.PaidAmountCents = -50000
	ctx.OutstandingCents = ctx.TotalCents - ctx.PaidAmountCents

	payload, err := billing.EPCPayload(ctx)
	if err == nil {
		t.Fatalf("buildEPCPayload returned nil error, payload %q", payload)
	}
	if !strings.Contains(err.Error(), "exceeds total") {
		t.Fatalf("error %q does not contain %q", err.Error(), "exceeds total")
	}
}
