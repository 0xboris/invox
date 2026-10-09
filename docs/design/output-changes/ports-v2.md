# Output changes: ports v2

`docs/design/ports-v2.md` narrows billing's ports to domain data. Its legacy-directory and
Markdown commits are units U2 and U3 and have their own files. The changes below come from
the other commits. None of them changes a testscript golden, `docs/cli` or `share/man`; each
is pinned by the Go test named with it.

## Reading invoices with `Load`

`archive add`, `archive edit` and `increment` read the invoice with the strict decoder that
`validate` uses, keeping only the decode errors of the fields they need, instead of a separate
lenient identity reader. `new` checks its output file when it writes it.

| Command | Before | Now |
| --- | --- | --- |
| `archive add` and `archive edit` with `invoice:` or `invoice: ~` | ``error: inv.yaml: `invoice` must be a mapping`` | ``error: inv.yaml: missing `invoice` mapping`` |
| `archive add` with a scalar such as `invoice: 5` | ``error: inv.yaml: `invoice` must be a mapping`` | `error: inv.yaml:2: invoice must be a mapping, got an integer`, the decoder's message with its line |
| `increment` with a malformed `invoice.issue_date` | `error: invoice.yaml: invoice.issue_date: expected YYYY-MM-DD, got ...` | the same with the line: `error: invoice.yaml:4: invoice.issue_date: ...` |
| `new` when the output file (`-o`, or the default `<number>.yaml`) exists or is a directory, and something else is wrong too | the output error won | an unknown customer, a missing `payment` mapping, a bad `due_days` or a numbering error wins; the output error comes last, with the same message |

Pinned by `TestArchiveRefusesInvoiceKeyThatIsNotAMapping`, `TestIncrementNeedsOnlyTheFieldsNumberingReads`
and the `new-default-exists-and-bad-due` case of `TestPortOrder`. A bad value in a field
`increment` does not read, such as `paid_amount: lots`, still does not stop it.

## Archive placement inside `Add`

`archive add` and `build --archive` check the invoice's status and number first, and where
the file goes when it is added, so the archive directory is read later.

| Command | Before | Now |
| --- | --- | --- |
| `archive add` with a broken `config.yaml` or unusable `archive.dir` and an invoice that is not built | the config or directory error | ``error: invoice.yaml: invoice.status must be `built` before archiving, got `draft` `` |
| `archive add` when a file of that name is already in the archive and an archived invoice has the same number | `error: <archive>/invoice.yaml already exists` | the duplicate-number error and its `invox increment` hint |
| `archive add` of an invoice whose `invoice:` is an alias, when an archived invoice has the same number | ``error: invoice.yaml: `invoice` must be a mapping`` | the duplicate-number error |
| `archive add --dry-run` of an invoice whose `invoice:` is an alias | ``error: invoice.yaml: `invoice` must be a mapping`` | unchanged: the dry run rewrites the invoice in memory, as the real run does |

Pinned by the `exists-vs-duplicate`, `dir-error-before-status` and
`archive-aliased-header-and-duplicate` cases of `TestPortOrder` and the `add dry-run alias`
case of `TestArchiveRefusesInvoiceKeyThatIsNotAMapping`.

## One `Mailer.Draft`

The mailer checks the PDF, writes the draft and opens it in one call; the `email` command no
longer opens the `.eml` file itself.

| Command | Before | Now |
| --- | --- | --- |
| `email` with a PDF that cannot be read and a recipient or subject error too | the PDF error (`read X: stat X: ...`) | the recipient or subject error; the PDF error comes after them, with the same message. Inferred from the code: validation catches a missing customer email first, so no run reached this order |

The messages for a draft that was written but could not be opened (`created X but failed to
open it: ...`, and `failed to open email draft: ...` for a temporary one, which is removed)
are unchanged; `email.txtar` and the email tests of `internal/cli` pin them.
