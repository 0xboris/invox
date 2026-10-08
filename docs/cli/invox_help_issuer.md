# invox help issuer

```text
issuer.yaml reference.

Usage:
  invox help issuer

Behavior:
  Shows the supported issuer.yaml shape used by new, validate, render, build, and email.
  `invox init` writes a starter issuer.yaml with this structure.

Formatting:
  Top-level keys must start at column 1 with no leading spaces.

Issuer fields:
  issuer.yaml contains your own company and payment details.
  Required fields are validated by new, validate, render, build, and email.

Required company fields:
  company.legal_company_name          Company name used on invoices, in email placeholders, and as the default EPC QR recipient name
  company.company_registration_number Company registration number shown on the invoice
  company.vat_tax_id                  VAT/tax number shown on the invoice
  company.website                     Website shown on the invoice
  company.email                       Sender/contact email shown on the invoice
  company.address.street              Business address street
  company.address.postal_code         Business address postal code
  company.address.city                Business address city
  company.address.country             Business address country

Required payment fields:
  payment.bank_name                   Bank name rendered into the invoice template
  payment.iban                        Bank account IBAN used on the invoice and for EPC QR generation
  payment.bic                         Bank identifier code shown on the invoice
  payment.due_days                    Non-negative integer day count used by `new` to prefill invoice.due_date
  payment.payment_terms_text          Payment terms text used by templates and email placeholders

Optional payment fields:
  payment.vat_label                   Overrides the VAT label used by @@VAT_SUMMARY_ROWS@@, defaults to VAT
  payment.epc_qr.label                Overrides the EPC QR label, defaults to Pay via EPC-QR
  payment.epc_qr.name                 Overrides the EPC QR recipient name, defaults to company.legal_company_name
  payment.epc_qr.purpose              Optional EPC QR purpose code, must be 1-4 letters or digits
  payment.epc_qr.text                 Optional EPC QR text line, defaults to invoice.number
  payment.epc_qr.information          Optional EPC QR unstructured remittance information


Rules:
  payment.due_days must be a non-negative integer.
  payment.vat_label defaults to VAT when omitted.
  payment.epc_qr.name defaults to company.legal_company_name.
  payment.epc_qr.text defaults to invoice.number.
  payment.epc_qr.label defaults to Pay via EPC-QR.
  EPC QR generation requires a valid SEPA-scope payment.iban.

Lookup:
  issuer.yaml: upward project search, then $HOME/.config/invox/issuer.yaml

Examples:
  invox help issuer
  invox new CUST-001 -u issuer.yaml

issuer.yaml example:
company:
  legal_company_name: Boris Consulting
  company_registration_number: FN 123456a
  vat_tax_id: ATU87654321
  website: https://example.com
  email: hello@example.com
  address:
    street: Ring 1
    postal_code: "1010"
    city: Vienna
    country: Austria
payment:
  bank_name: Test Bank
  iban: AT611904300234573201
  bic: BKAUATWW
  due_days: 30
  payment_terms_text: Pay within 30 days
  vat_label: VAT
  epc_qr:
    label: Pay via EPC-QR
    purpose: SUPP
    information: Scan to pay this invoice
    # name: Boris Consulting
    # text: 2026-0001
```

## See also

- [invox](invox.md): Generate LaTeX and PDF invoices from YAML data
