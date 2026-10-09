package store

import (
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

const (
	internalMetadataKey    = "_invox"
	internalArchivePathKey = "archive_path"
)

func archiveMetadata(root *yaml.Node) string {
	internalNode := findMappingValue(root, internalMetadataKey)
	if internalNode == nil || internalNode.Kind != yaml.MappingNode {
		return ""
	}
	return filepath.FromSlash(strings.TrimSpace(nodeText(findMappingValue(internalNode, internalArchivePathKey))))
}

// setArchiveMetadata records the archive-relative path with forward slashes
// so a working copy stays portable between operating systems.
func setArchiveMetadata(root *yaml.Node, archivePath string) {
	internalNode := getOrCreateMappingNode(root, internalMetadataKey)
	setMappingString(internalNode, internalArchivePathKey, filepath.ToSlash(archivePath))
}
