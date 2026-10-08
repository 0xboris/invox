# PRD: Per-Position VAT Override With Invoice Default

## Summary

Add support for defining VAT rates on individual invoice positions while keeping `invoice.vat_percent` as the default VAT rate for the whole invoice.

This feature standardizes on canonical YAML keys only:

- `invoice.period`
- `invoice.vat_percent`
- `positions`
- `positions[].vat_percent`

Legacy aliases are no longer supported.

The rule is:

- `invoice.vat_percent` remains the invoice-wide default VAT rate
- `positions[].vat_percent` is optional
- if a position defines `vat_percent`, that rate is used for that position
- otherwise the position inherits `invoice.vat_percent`
- if `invoice.vat_percent` is missing, the position falls back to `customer.tax.default_vat_rate`

This keeps the canonical invoice model simple while allowing mixed-rate invoices.

## Problem

Today the invoice model assumes a single VAT rate for the entire invoice.

That prevents common cases such as:

- one invoice containing services at `20%` VAT and goods at `10%`
- one default VAT rate for the invoice with only a few exception positions
- invoices where most positions follow the standard rate and only selected lines need a reduced or zero rate

## Goals

- Preserve `invoice.vat_percent` as the default VAT rate for the invoice
- Allow any position to override the default with `positions[].vat_percent`
- Keep existing canonical single-rate invoices working without migration
- Render correct totals for invoices with mixed VAT rates
- Show VAT totals grouped by VAT rate in the rendered invoice
- Allow custom LaTeX templates to show each position's effective VAT rate when desired

## Non-Goals

- Removing `invoice.vat_percent`
- Requiring every position to declare its own VAT rate
- Changing archive, email, numbering, or customer workflows beyond using the new totals
- Introducing VAT exemption reason codes or reverse-charge semantics in this feature
- Keeping support for removed YAML aliases such as `invoice.vat_rate_percent`, `invoice.period_label`, or `line_items`
- Changing the default starter template to show a VAT column in the line-item table

## User Stories

- As a user, I want to set `invoice.vat_percent: 20` once and let most positions inherit it.
- As a user, I want to mark one specific position with `vat_percent: 10` without affecting the rest of the invoice.
- As a user, I want existing invoices that only use `invoice.vat_percent` to continue working unchanged.
- As a user, I want to set `customer.tax.default_vat_rate` once and omit invoice VAT on invoices that always use that same rate.
- As a user, I want the rendered invoice totals to remain correct when multiple VAT rates are used.

## YAML Contract

### Canonical shape

```yaml
customer_id: CUST-001

invoice:
  number: BL00210001
  issue_date: 2026-03-10
  due_date: 2026-04-09
  status: draft
  period: March 2026
  vat_percent: 20

positions:
  - name: Consulting
    description: Strategy work
    unit_price: 100
    quantity: 2

  - name: Printed material
    description: Booklets
    unit_price: 30
    quantity: 1
    vat_percent: 10
```

### Resolution rules

- Effective VAT for each position is resolved in this order:
  1. `positions[i].vat_percent`
  2. `invoice.vat_percent`
  3. `customer.tax.default_vat_rate`
  4. validation error
- `invoice.vat_percent` is optional only if every position declares its own `vat_percent` or the customer defines `tax.default_vat_rate`
- `customer.tax.default_vat_rate` is part of runtime VAT resolution for `validate`, `render`, `build`, and `email`
- `new` may prefill `invoice.vat_percent` from `customer.tax.default_vat_rate`, but only when the selected draft source does not already define `invoice.vat_percent`
- The only supported invoice keys for this feature are the canonical keys shown above; removed aliases must fail validation

## Functional Requirements

### Validation

- `invoice.vat_percent` remains valid and supported as the invoice-wide default
- `positions[].vat_percent` is optional
- `invoice.vat_percent`, when present, must be a valid non-negative decimal or percent string
- `positions[].vat_percent`, when present, must be a valid non-negative decimal or percent string using the same parsing rules as `invoice.vat_percent`
- `customer.tax.default_vat_rate`, when present, must be a valid non-negative decimal or percent string using the same parsing rules as `invoice.vat_percent`
- `0` and `0%` are valid values for zero-rated positions
- Validation must fail when a position has no effective VAT rate after `positions[i].vat_percent -> invoice.vat_percent -> customer.tax.default_vat_rate` resolution
- Validation must reject removed aliases with migration-directed errors
- Validation errors should point to the concrete source field:
  - `positions[2].vat_percent: expected a number or percent string`
  - `customer.tax.default_vat_rate: expected a number or percent string`
  - `invoice.vat_percent: missing value` only when a position has no explicit rate and neither `invoice.vat_percent` nor `customer.tax.default_vat_rate` is available
  - `invoice.vat_rate_percent: unsupported key; use invoice.vat_percent`
  - `invoice.period_label: unsupported key; use invoice.period`
  - `line_items: unsupported key; use positions`

### Totals

- Net line total remains `unit_price * quantity`
- Each position gets an effective VAT rate
- Rate representations are normalized before grouping, so `10`, `10.0`, and `10%` map to the same VAT bucket
- VAT is calculated per VAT-rate bucket:
  - sum all net line totals for the same effective VAT rate
  - calculate VAT for that bucket
  - round that bucket VAT to cents
- Total VAT is the sum of all bucket VAT amounts
- Invoice total is `subtotal + total VAT`
- Outstanding amount continues to be `total - paid_amount`
- VAT buckets are rendered in ascending numeric VAT-rate order

### Rendering

- The default line-item table remains unchanged for this feature
- `@@LINE_ITEMS_ROWS@@` remains supported and keeps its current five-column shape:
  - `name`, `description`, `unit price`, `quantity`, `line total`
- Introduce a new structured placeholder `@@LINE_ITEMS_ROWS_WITH_VAT@@` for custom templates that want to show the effective VAT rate per position
- `@@LINE_ITEMS_ROWS_WITH_VAT@@` expands rows in this order:
  - `name`, `description`, `unit price`, `quantity`, `VAT rate`, `line total`
- The VAT value shown in `@@LINE_ITEMS_ROWS_WITH_VAT@@` is the resolved effective rate for that position after `positions[i].vat_percent -> invoice.vat_percent -> customer.tax.default_vat_rate` fallback
- Example row shape for `@@LINE_ITEMS_ROWS_WITH_VAT@@`:

```tex
Consulting & Strategy work & 100,00 EUR & 2 & 20\% & 200,00 EUR\\
```

- The totals section changes from one VAT row to one VAT row per VAT-rate bucket
- Introduce a new placeholder `@@VAT_SUMMARY_ROWS@@` that expands to one or more LaTeX rows inside the totals table
- Each generated row uses this shape:

```tex
VAT (10\%): & 3,00 EUR\\
```

- `@@VAT_SUMMARY_ROWS@@` must emit rows in ascending numeric VAT-rate order
- `@@VAT_RATE@@` and `@@VAT_AMOUNT@@` are removed from the supported template placeholder set because they cannot represent mixed-rate invoices correctly
- Example:

```text
Subtotal:      230,00 EUR
VAT (10%):      3,00 EUR
VAT (20%):     40,00 EUR
Total:        273,00 EUR
```

- If all positions resolve to the same VAT rate, `@@VAT_SUMMARY_ROWS@@` still emits exactly one VAT row and the rendered invoice should look effectively like today
- The starter template continues to use `@@LINE_ITEMS_ROWS@@`; custom templates may opt into `@@LINE_ITEMS_ROWS_WITH_VAT@@`

### Draft Creation

- `new` continues to support `invoice_defaults.yaml`
- `invoice_defaults.yaml` may keep `invoice.vat_percent` as the common default
- Positions in defaults may omit `vat_percent` when they should inherit the invoice default
- Positions in defaults may define explicit `vat_percent` overrides when needed
- If the selected draft source already defines `invoice.vat_percent`, `new` must preserve it and must not overwrite it from `customer.tax.default_vat_rate`
- If the selected draft source does not define `invoice.vat_percent`, `new` may prefill it from `customer.tax.default_vat_rate`
- `new --from-last` preserves the source invoice’s `invoice.vat_percent` and any explicit `positions[].vat_percent` overrides

### Build / Validate / Email / Archive

- `validate`, `render`, `build`, and `email` must use the new effective VAT calculation
- Runtime VAT resolution in those commands must honor `positions[i].vat_percent -> invoice.vat_percent -> customer.tax.default_vat_rate`
- `archive`, `archive edit`, and `archive list` need no feature-specific UX changes
- `email` placeholders based on total or outstanding amount continue to work, but their values must reflect the new mixed-rate totals

## Compatibility And Migration

- Existing invoices that already use canonical keys must continue to work without modification
- Existing invoice defaults that already use canonical keys must continue to work without modification
- Existing archived invoices that already use canonical keys must continue to render and validate without migration
- Files that still use removed aliases must be migrated before `validate`, `render`, `build`, `email`, `archive edit`, or `new --from-last`
- Existing custom templates must migrate from `@@VAT_RATE@@` and `@@VAT_AMOUNT@@` to `@@VAT_SUMMARY_ROWS@@`
- Position-level `vat_percent` is additive for canonical invoice data, but this release is intentionally breaking for alias-based YAML and single-rate VAT template placeholders

## Data Model Changes

Recommended internal model changes:

- `LineItem`
  - add `VATRatePercent`
- `Context`
  - keep `SubtotalCents`, `VATAmountCents`, `TotalCents`, `PaidAmountCents`, `OutstandingCents`
  - replace single-rate assumptions with a VAT breakdown collection:
    - `VATBreakdowns []VATBreakdown`
  - remove the single-rate field `VATRatePercent`

Recommended VAT breakdown shape:

```go
type VATBreakdown struct {
    RatePercent  *big.Rat
    NetCents     int64
    VATAmountCents int64
}
```

## Implementation Scope

Expected code areas:

- `internal/invoice/service.go`
  - reject removed aliases with clear migration errors
  - parse effective VAT rate per position using `positions[i].vat_percent -> invoice.vat_percent -> customer.tax.default_vat_rate`
  - compute normalized VAT buckets and sort them by rate
  - expose `@@LINE_ITEMS_ROWS_WITH_VAT@@`
  - expose `@@VAT_SUMMARY_ROWS@@`
- `internal/invoice/drafts.go`
  - write canonical keys only
  - preserve invoice default VAT and position overrides in generated drafts
  - only use `customer.tax.default_vat_rate` when the selected draft source leaves `invoice.vat_percent` unset
- `internal/invoice/starter/template.tex`
  - replace `@@VAT_RATE@@` and `@@VAT_AMOUNT@@` with `@@VAT_SUMMARY_ROWS@@`
- `README.md`
  - document removed aliases, `@@LINE_ITEMS_ROWS_WITH_VAT@@`, and the new VAT summary placeholder contract
- starter/sample YAML files
  - document inherited and explicit position-level VAT
- tests
  - validation
  - total calculation
  - mixed-rate rendering
  - customer-level VAT fallback
  - canonical backward compatibility
  - alias rejection and migration errors

## Acceptance Criteria

- An invoice with only `invoice.vat_percent` and canonical keys validates and renders the same visible totals as before
- An invoice with `invoice.vat_percent: 20` and one position override `vat_percent: 10` validates successfully
- An invoice with no `invoice.vat_percent` validates and renders successfully when `customer.tax.default_vat_rate` is present
- Mixed-rate invoices render separate VAT summary rows sorted by ascending VAT rate
- Totals are correct for mixed-rate invoices
- A custom template using `@@LINE_ITEMS_ROWS_WITH_VAT@@` renders each position's effective VAT rate correctly
- `new` can create invoices that rely on the invoice default VAT rate
- `new` only uses `customer.tax.default_vat_rate` when the selected draft source omits `invoice.vat_percent`
- `new --from-last` preserves explicit position overrides from the previous invoice and does not overwrite them from customer data
- Existing fixtures and archived invoices that use canonical invoice-level VAT continue to work
- Files that still use `invoice.vat_rate_percent`, `invoice.period_label`, or `line_items` fail with clear migration errors
- Templates using `@@VAT_SUMMARY_ROWS@@` render correctly for both single-rate and mixed-rate invoices

## Example Scenarios

### Scenario 1: Invoice-wide default only

```yaml
invoice:
  vat_percent: 20

positions:
  - name: Consulting
    description: Strategy work
    unit_price: 100
    quantity: 2
```

Expected:

- position VAT rate resolves to `20`
- one VAT row is rendered

### Scenario 2: One overridden position

```yaml
invoice:
  vat_percent: 20

positions:
  - name: Consulting
    description: Strategy work
    unit_price: 100
    quantity: 2

  - name: Printed material
    description: Booklets
    unit_price: 30
    quantity: 1
    vat_percent: 10
```

Expected:

- first position resolves to `20`
- second position resolves to `10`
- two VAT summary rows are rendered

### Scenario 3: Customer-level fallback

`customers.yaml`

```yaml
CUST-001:
  tax:
    default_vat_rate: 20
```

`invoice.yaml`

```yaml
customer_id: CUST-001
invoice: {}

positions:
  - name: Consulting
    description: Strategy work
    unit_price: 100
    quantity: 1
```

Expected:

- the position VAT rate resolves to the customer's `20`
- one VAT row is rendered

### Scenario 4: Missing effective VAT

```yaml
invoice: {}

positions:
  - name: Consulting
    description: Strategy work
    unit_price: 100
    quantity: 1
```

Expected:

- validation fails because the position has no effective VAT rate

### Scenario 5: Removed alias is rejected

```yaml
invoice:
  vat_rate_percent: 20

line_items:
  - name: Consulting
    description: Strategy work
    unit_price: 100
    quantity: 1
```

Expected:

- validation fails with migration-directed errors
- the user is told to replace `invoice.vat_rate_percent` with `invoice.vat_percent`
- the user is told to replace `line_items` with `positions`

## Template Decision

- The default starter template keeps the existing five-column line-item table
- Custom templates can opt into a VAT column by using `@@LINE_ITEMS_ROWS_WITH_VAT@@`
- The totals section still expands to `@@VAT_SUMMARY_ROWS@@` for mixed-rate VAT totals
