package invoice

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/0xboris/invox/internal/fsutil"
)

// EditArchiveOptions controls EditArchivedInvoice.
type EditArchiveOptions struct {
	// Overwrite replaces an existing working copy, unless it is in the
	// archive directory.
	Overwrite bool
	// DryRun runs every check and returns the paths without writing.
	DryRun bool
}

// EditArchivedInvoice copies the archived invoice archiveName into workDir as
// a working copy and returns its path and the archived invoice's path.
func (h Host) EditArchivedInvoice(archiveName, workDir string, opts EditArchiveOptions) (string, string, error) {
	target, err := h.resolveArchiveInputPath(archiveName)
	if err != nil {
		return "", "", err
	}
	archivePath := target.Path

	document, ok, err := loadArchivedInvoiceDocument(archivePath)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", fmt.Errorf("%s: archived invoice could not be loaded", archivePath)
	}
	// The working copy keeps keys invox does not know, so opening an
	// archived invoice never fails over them; validate reports them.
	if err := decodeYAMLDocument(document, archivePath, &InvoiceFile{}, false); err != nil {
		return "", "", err
	}

	root, err := documentRootMapping(document, archivePath)
	if err != nil {
		return "", "", err
	}
	invoiceNode, err := invoiceMapping(root, archivePath)
	if err != nil {
		return "", "", err
	}

	edit := target.Edit()
	outputPath := filepath.Join(workDir, edit.Filename)
	if err := refuseDirOutput(outputPath); err != nil {
		return "", "", err
	}
	if !opts.Overwrite && fileExists(outputPath) {
		return "", "", &OutputExistsError{Path: outputPath}
	}

	setMappingString(invoiceNode, "status", string(Editing))
	setArchiveMetadata(root, edit.Target, edit.Replace)

	data, err := encodeYAMLDocument(document)
	if err != nil {
		return "", "", err
	}
	if err := h.refuseArchivedOverwrite(outputPath, opts.Overwrite); err != nil {
		return "", "", err
	}
	if opts.DryRun {
		return outputPath, archivePath, nil
	}
	write := fsutil.WriteNewFile
	if opts.Overwrite {
		write = fsutil.WriteFile
	}
	if err := write(outputPath, data, fsutil.Public); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", "", &OutputExistsError{Path: outputPath}
		}
		return "", "", err
	}
	return outputPath, archivePath, nil
}
