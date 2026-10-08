package latex

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// AssetDirs lists the relative directories a rendered template names in
// Path= options.
func AssetDirs(rendered string) []string {
	re := regexp.MustCompile(`Path=([^,\]\n]+)`)
	matches := re.FindAllStringSubmatch(rendered, -1)
	return uniqueRelativePaths(matches, 1)
}

// AssetFiles lists the relative files a rendered template includes with
// \includegraphics.
func AssetFiles(rendered string) []string {
	re := regexp.MustCompile(`\\includegraphics(?:\[[^\]]*\])?\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(rendered, -1)
	return uniqueRelativePaths(matches, 1)
}

func uniqueRelativePaths(matches [][]string, index int) []string {
	seen := make(map[string]struct{})
	paths := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) <= index {
			continue
		}
		path := strings.TrimSpace(match[index])
		if path == "" || filepath.IsAbs(path) || strings.HasPrefix(path, "..") {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}
