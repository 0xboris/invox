package store

// archivedInvoiceIdentity reads what the archive lists of the invoice at
// path. ok is false for a file that is not an invoice: Markdown without
// front matter, or a document whose invoice fields cannot be read.
func archivedInvoiceIdentity(path string) (invoiceIdentity, bool, error) {
	document, ok, err := loadArchivedInvoiceDocument(path)
	if err != nil || !ok {
		return invoiceIdentity{}, false, err
	}
	root, err := documentRootMapping(document, path)
	if err != nil {
		return invoiceIdentity{}, false, nil //nolint:nilerr // not an invoice: the caller skips it
	}
	var identity invoiceIdentity
	if err := decodeYAMLNode(root, path, &identity, false); err != nil || identity.Invoice == nil {
		return invoiceIdentity{}, false, nil //nolint:nilerr // not an invoice: the caller skips it
	}
	return identity, true, nil
}
