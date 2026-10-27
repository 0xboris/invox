# Glossary

One name for each business concept. Code, tests, and conversations use these names, and nothing else, for these concepts.

Each concept is a level-2 heading. The architecture check reads the headings: utilities and shared UI may not use these words, because those modules have no business meaning.

## Invoice

A YAML file that bills one customer for one or more line items. invox validates it, renders it to LaTeX, builds the PDF with tectonic, emails it, and archives it.

## Invoice number

The number that identifies an invoice. It is unique among the issuer's archived invoices, and the customer's numbering settings decide the next one.

## Line item

One billed position on an invoice: a description, a quantity, a unit price, and a VAT rate.

## Customer

Someone the issuer bills. A customer has an address, tax details, billing settings, and numbering settings, and lives in the customers file.

## Issuer

The company that sends the invoices, with its address and payment details.

## VAT

The value added tax on a line item. An invoice shows a VAT breakdown with one row per rate.

## EPC QR code

The payment QR code that the European Payments Council defines. invox adds it to an invoice in euros when the payment details allow it.

## Archived invoice

An invoice that was sent and moved into the archive directory. Its invoice number counts as used.
