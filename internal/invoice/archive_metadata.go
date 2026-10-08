package invoice

import (
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

const (
	internalMetadataKey       = "_invox"
	internalArchivePathKey    = "archive_path"
	internalArchiveReplaceKey = "archive_replace_path"
)

func archiveMetadata(root *yaml.Node) (string, string) {
	internalNode := findMappingValue(root, internalMetadataKey)
	if internalNode == nil || internalNode.Kind != yaml.MappingNode {
		return "", ""
	}
	return filepath.FromSlash(strings.TrimSpace(nodeText(findMappingValue(internalNode, internalArchivePathKey)))),
		filepath.FromSlash(strings.TrimSpace(nodeText(findMappingValue(internalNode, internalArchiveReplaceKey))))
}

// setArchiveMetadata records archive-relative paths with forward slashes so
// a working copy stays portable between operating systems.
func setArchiveMetadata(root *yaml.Node, archivePath, archiveReplacePath string) {
	internalNode := getOrCreateMappingNode(root, internalMetadataKey)
	setMappingString(internalNode, internalArchivePathKey, filepath.ToSlash(archivePath))
	if strings.TrimSpace(archiveReplacePath) == "" || archiveReplacePath == archivePath {
		deleteMappingKey(internalNode, internalArchiveReplaceKey)
	} else {
		setMappingString(internalNode, internalArchiveReplaceKey, filepath.ToSlash(archiveReplacePath))
	}
}

func clearArchiveMetadata(root *yaml.Node) {
	deleteMappingKey(root, internalMetadataKey)
}
