package store

import (
	"path/filepath"

	yaml "gopkg.in/yaml.v3"
)

const (
	internalMetadataKey    = "_invox"
	internalArchivePathKey = "archive_path"
)

// setArchiveMetadata records the archive-relative path with forward slashes
// so a working copy stays portable between operating systems.
func setArchiveMetadata(root *yaml.Node, archivePath string) {
	internalNode := getOrCreateMappingNode(root, internalMetadataKey)
	setMappingString(internalNode, internalArchivePathKey, filepath.ToSlash(archivePath))
}
