package invoice

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/0xboris/invox/internal/fsutil"
)

func (h Host) copyTemplateAssets(templatePath, outputPath, rendered string) error {
	templateDir := filepath.Dir(templatePath)
	outputDir := filepath.Dir(outputPath)
	if templateDir == outputDir {
		return nil
	}

	for _, relDir := range referencedAssetDirs(rendered) {
		sourceDir := h.findAssetDir(templatePath, relDir)
		if sourceDir == "" {
			continue
		}
		destDir := filepath.Join(outputDir, relDir)
		if err := copyDir(sourceDir, destDir); err != nil {
			return err
		}
	}

	for _, relFile := range referencedAssetFiles(rendered) {
		sourceFile := h.findAssetFile(templatePath, relFile)
		if sourceFile == "" {
			continue
		}
		destFile := filepath.Join(outputDir, relFile)
		if fileExists(destFile) {
			continue
		}
		if err := copyFile(sourceFile, destFile); err != nil {
			return err
		}
	}

	return nil
}

func (h Host) findAssetDir(templatePath, relPath string) string {
	return h.findAsset(templatePath, relPath, true)
}

func (h Host) findAssetFile(templatePath, relPath string) string {
	return h.findAsset(templatePath, relPath, false)
}

// findAsset looks for relPath next to the template, then in the config
// directories.
func (h Host) findAsset(templatePath, relPath string, isDir bool) string {
	if candidate := filepath.Join(filepath.Dir(templatePath), relPath); pathExists(candidate, isDir) {
		return candidate
	}
	found, _ := h.findInConfigDir(isDir, relPath)
	return found.Path
}

func referencedAssetDirs(rendered string) []string {
	re := regexp.MustCompile(`Path=([^,\]\n]+)`)
	matches := re.FindAllStringSubmatch(rendered, -1)
	return uniqueRelativePaths(matches, 1)
}

func referencedAssetFiles(rendered string) []string {
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

func copyDir(sourceDir, destDir string) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(destDir, relPath)
		if info.IsDir() {
			return fsutil.MkdirAll(targetPath, fsutil.Perm{Dir: info.Mode().Perm()})
		}
		return copyFile(path, targetPath)
	})
}

func copyFile(sourcePath, destPath string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}
	return fsutil.WriteFile(destPath, data, fsutil.Perm{File: info.Mode().Perm(), Dir: 0o755})
}
