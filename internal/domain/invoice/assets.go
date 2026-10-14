package invoice

import (
	"io/fs"
	"path/filepath"

	"github.com/0xboris/invox/internal/domain/latex"
	"github.com/0xboris/invox/internal/domain/ports"
)

func (h Host) copyTemplateAssets(templatePath, outputPath, rendered string) error {
	files := h.files
	templateDir := filepath.Dir(templatePath)
	outputDir := filepath.Dir(outputPath)
	if templateDir == outputDir {
		return nil
	}

	for _, relDir := range latex.AssetDirs(rendered) {
		sourceDir := h.findAssetDir(templatePath, relDir)
		if sourceDir == "" {
			continue
		}
		destDir := filepath.Join(outputDir, relDir)
		if err := copyDir(files, sourceDir, destDir); err != nil {
			return err
		}
	}

	for _, relFile := range latex.AssetFiles(rendered) {
		sourceFile := h.findAssetFile(templatePath, relFile)
		if sourceFile == "" {
			continue
		}
		destFile := filepath.Join(outputDir, relFile)
		if fileExists(files, destFile) {
			continue
		}
		if err := copyFile(files, sourceFile, destFile); err != nil {
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
	files := h.files
	if candidate := filepath.Join(filepath.Dir(templatePath), relPath); pathExists(files, candidate, isDir) {
		return candidate
	}
	found, _ := h.findInConfigDir(isDir, relPath)
	return found.Path
}

func copyDir(files ports.Files, sourceDir, destDir string) error {
	return files.Walk(sourceDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(destDir, relPath)
		if info.IsDir() {
			return files.MkdirAll(targetPath, ports.Perm{Dir: info.Mode().Perm()})
		}
		return copyFile(files, path, targetPath)
	})
}

func copyFile(files ports.Files, sourcePath, destPath string) error {
	data, err := files.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	info, err := files.Stat(sourcePath)
	if err != nil {
		return err
	}
	return files.WriteFile(destPath, data, ports.Perm{File: info.Mode().Perm(), Dir: 0o755})
}
