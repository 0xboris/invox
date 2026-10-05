# invox correctness and robustness bug hunt

## (a) Verdict
The core money math (big.Rat, half-up rounding, VAT per rate) and the EPC payload are correct. Most of the serious bugs come from the edges around that core. The default starter template silently drops or mangles non-ASCII characters in the PDF. yaml.v3 reads numbers with a leading zero as octal, which silently changes quantities, prices and postal codes. Nothing makes invoice numbers unique outside the archive. Several smaller problems trace back to the `map[string]any` model: `paid_amount` has no sign check, YAML merge keys (`<<`) are ignored, values of the wrong type are turned into strings, and placeholder replacement depends on Go's random map order.

Proof tests were written in a scratch copy of the repo and are not kept; each finding names its test and observed output. TeX proofs were compiled with XeLaTeX from TeX Live, which is the same engine tectonic wraps; the tectonic bundle host is blocked by the proxy. The QR code was decoded with `zbarimg`.

## (b) Findings

### CRITICAL

**1. The starter template silently corrupts names and addresses** — `internal/invoice/starter/template.tex:2-3` (`\usepackage[T1]{fontenc}` + `inputenc` on XeTeX)
- Failure: the customer `Čapek s.r.o. – Łódź Kőbánya` and street `Hauptstraße 1 § 5` render as **"apek s.r.o. ód Kbánya"** and **"HauptstraSSe 1 ğ 5"**. The item name `Straße` renders as "StraSSe". XeLaTeX exits 0 and the log only says `Missing character: There is no Č…`.
- **PROVED**: `TestBH_RenderStarterUnicode` plus `xelatex`, then `pdftotext`.
- Fix: remove fontenc and inputenc and use `\usepackage{fontspec}`, which uses TU encoding. Add `\tracinglostchars=3` so a missing glyph becomes an error.

**2. A leading zero makes YAML read a number as octal** — `yaml.go:62-71` (`normalizeYAMLScalar` → `node.Decode(&any)`)
- Failure: `postal_code: 01067` becomes **"567"**. `unit_price: 0100, quantity: 010` gives **64 × 8 = 512.00** instead of 1000.00, with no error. `number: 0042` becomes "34". `08001` becomes float 8001, so the leading zero is lost.
- Related: `parseDecimal` (`service.go:1728`) uses `big.Rat.SetString`, which also accepts `"010/1"` (= 8), `"0x10"` (= 16) and `"1/3"`.
- **PROVED**: `TestBH_YAMLScalarCoercion`, `TestBH_OctalQuantityAndPostalCode`, and the `rat/main.go` program.
- Fix: for `!!int`/`!!float` nodes keep `node.Value` (the source text) and parse money with a strict decimal regex `^-?\d+(\.\d+)?$`. Reject ints with a leading zero, or keep them as strings for text fields. Better still, decode into typed structs that use string and decimal fields.

### MAJOR

**3. Two invoices can get the same number** — `numbering.go:72` (`highestArchivedCounter` only scans the archive), `drafts.go:96`, `drafts.go:300-314` (`ArchiveInvoice` only checks the filename)
- Failure: running `invox new CUST-001 -o a.yaml` and then `-o b.yaml` before archiving gives **both** the number CUST-001-001. Both then archive successfully, so the archive holds two invoices with the same number. The default `-o` only collides by accident, and that check is racy because `writeFileAtomic` overwrites.
- **PROVED**: `TestBH_DuplicateNumbersWithoutArchive` and `TestBH_DuplicateNumbersArchivedTwice`.
- Fix: refuse to archive a number that already exists in the archive. Reserve numbers with a locked counter file, or also scan open drafts.

**4. A negative `paid_amount` inflates the amount due and the QR amount** — `service.go:793` (`coerceDecimal` has no sign check)
- Failure: an invoice with a total of 120.00 and `paid_amount: -500` gives **outstanding 620.00**, and the EPC QR line reads `EUR620.00`.
- **PROVED**: `TestBH_NegativePaidAmount`.
- Fix: reject `paid_amount < 0`.

**5. YAML merge keys and duplicate keys are silently wrong** — `yaml.go:39-46`
- Failure: a position written as `<<: *reduced` (where `reduced` sets `vat_percent: 10`) is billed at the **invoice's 20%**: total 120.00 instead of 110.00. The decoder treats `<<` as a literal key. Duplicate keys (`unit_price: 1` followed by `unit_price: 2`) give last-wins with no error, while yaml.v3's normal decode would reject them.
- **PROVED**: `TestBH_MergeKeyIgnored` and `TestBH_YAMLScalarCoercion` (`dup` → 2).
- Fix: decode into typed structs with `yaml.v3` `Decode`, which handles merges and rejects duplicates. At minimum, implement `<<` and error on duplicate keys.

### MINOR

**6. Placeholder replacement is nondeterministic and rescans substituted values** — `service.go:917-923`
- Failure: the item `name: "Ref @@IBAN@@"`, `description: "see @@TOTAL@@"` produced **4 different outputs in 200 renders**, because the result depends on map iteration order. Values pass through `latexEscape`, which leaves `@@` alone, so user text can pick up the IBAN or total.
- **PROVED**: `TestBH_PlaceholderReinjection`. This is not a TeX injection, because every value is already escaped.
- Fix: do one pass with a single `strings.NewReplacer` that covers all placeholders, or escape `@@` in values. Also reject unknown `@@X@@` tokens, which currently pass through as literal text.

**7. A value that starts with `*` or `[` after `\\` breaks the output** — `latexEscape` (`service.go:1826`) together with the template's `…@@\\` lines
- Failure: street `*Hinterhaus* Hauptstr. 1` renders as "Hinterhaus* …" (the first `*` is silently eaten by `\\*`). `period: "[Q1] 2026"` stops the build with `! Missing number… Illegal unit of measure`.
- **PROVED**: `TestBH_RenderStarterEdge` plus `xelatex`.
- Fix: in `latexEscape`, escape `[`/`]` as `{[}`/`{]}` and prefix the value with `{}` (or `\relax`).

**8. Cents overflow int64 silently** — `service.go:1775` (`quotient.Int64()`) and the `+=` at `:839`
- Failure: `unit_price: "100000000000000000"` gives subtotal **-8446744073709551616** and a printed total of "83.106.511.852.580.896,77".
- **PROVED**: `TestBH_Int64Overflow`.
- Fix: check `IsInt64()` and add a sane upper bound per line and per total.

**9. The displayed unit price does not match the line total** — `service.go:1555` and `:1268`
- Failure: `unit_price: 0.125, quantity: 8` prints `0,13 € × 8 = 1,00 €`, while the visible arithmetic gives 1,04.
- **PROVED**: `TestBH_UnitPriceDisplayMismatch`.
- Fix: reject prices with more than 2 decimals, or show the full precision.

**10. MIME problems in the .eml draft** — `email.go:264-283`
- Failure: the text part declares `Content-Transfer-Encoding: 7bit` but carries UTF-8 ("Grüße"). The attachment filename is not RFC 2231/2047 encoded and quotes are not escaped. For `Rechnung "Müller" 2026.pdf`, Go's parser returns `FileName() == ""`.
- **PROVED**: `TestBH_EmailMIME`.
- Fix: use `8bit` or quoted-printable for the body, and `mime.FormatMediaType` for `name`/`filename`.

**11. Alias expansion has no limit (billion laughs)** — `yaml.go:53`
- Failure: a 330-byte file took **16 s** of CPU in `normalizeYAMLNode`. One such file in the archive dir stalls every `new` and `archive list`.
- **PROVED**: `TestBH_BillionLaughs`.
- Fix: cap expansion, or use `Decode` (which has yaml.v3's alias limits).

**12. A numeric `customer_id` is rejected with a misleading error** — `service.go:704` (`.(string)`)
- Failure: `customer_id: 1001` gives "missing `customer_id`" plus a cascade of `customer.*` errors, while `invoiceIdentity` (`drafts.go:345`) accepts the same value.
- **PROVED**: `TestBH_NumericCustomerID`.
- Fix: use `asString`, or a typed field.

**13. Gaps in numbering pattern validation** — `numbering.go:121-168` and `:300-352`
- Failure: `{customer_id}-{counter:03}/{counter}` passes validation, but `parseInvoiceCounter` can never match it (it builds 2 groups and expects `len(matches) != 2`). The counter therefore restarts every time and produces duplicates. With `{customer_code}{counter:03}`, customer `A` reads `A1007` (which belongs to customer `A1`) as counter 1007.
- **PROVED**: `TestBH_DuplicateCounterToken` and `TestBH_PatternPrefixCollision`.
- Fix: allow exactly one counter token, and require a non-digit literal between the customer token and the counter.

**14. The IBAN check accepts check digits 00, 01 and 99** — `service.go:1431`
- Failure: `DE01370400440000000042` is accepted when the correct check digits are 98.
- **PROVED**: `TestBH_IBANCheckDigits01`.
- Fix: require check digits in the range 02–98.

**15. Values of the wrong type are silently turned into strings** — `asString` (`service.go:1855`)
- Failure: `name: {first: Appsters}` prints as "map[first:Appsters]", and `street: [a, b]` prints as "[a b]".
- **PROVED**: `TestBH_WrongTypesStringified`.
- Fix: use typed fields so a wrong type becomes a validation error.

**16. The email draft file is deleted after 5 s even with an explicit `-o`** — `cli/commands_invoice.go:193-218`
- Failure: this is documented behaviour, but the file passed to `-o OUTPUT.eml` disappears. Any existing `<input>.eml` is overwritten and then removed.
- TRACED, not tested.

**17. `build` overwrites the status of archived invoices** — `cli/commands_invoice.go:256`
- Failure: rebuilding an archived invoice changes its status from `archived` to `built`.
- TRACED.

## Checked, OK
- **Float to decimal:** `19.99` and `0.1` parse exactly, because Go's shortest float formatting round-trips. Rounding is half-up away from zero. VAT is computed per rate on the summed net (§14 UStG style), and total = subtotal + Σ VAT rows, so totals are consistent. `paid > total` is rejected.
- **TeX escaping:** `latexEscape` covers all 10 special characters in a single pass. `\input{/etc/passwd}` and `\write18` in user values are neutralised. Tectonic runs without `-Z shell-escape`.
- **EPC payload:** field order, BCD/002/1/SCT, `EUR` amount format, the 0.01–999999999.99 range, rune limits (name 70, text 140, info 70) and the 331-byte limit are all right. Structured and unstructured remittance are never emitted together.
- **EPC QR decode:** decoding the PDF's QR with zbar returns correct UTF-8 bytes (`M\303\274ller`), and `% ~ { } &` survive. The mod-97 algorithm and the SEPA country list are current, including AL, MD, ME, MK and RS.
- **Archive:** `archive edit ../../x` and absolute paths are rejected. Archive writes go through temp+rename in the same dir, so there is no EXDEV problem.
- **Email headers:** osascript receives values as argv, so there is no AppleScript injection. A CRLF in the recipient cannot inject headers (net/mail quotes it). Subject and From use RFC 2047. Base64 is wrapped at 76 columns with CRLF.
- **Dates:** YAML timestamps are normalised to `YYYY-MM-DD`, invalid dates are rejected, and `new` uses the local date.
- **Status rewrite:** comments and values are kept. Only indentation and blank lines are normalised, which is cosmetic.
- **Panics:** fuzzing `LoadContext` + `RenderInvoice` for 60 s (12k execs) found none.
- **Errors:** no ignored errors that matter.
