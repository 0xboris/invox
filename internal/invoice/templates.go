package invoice

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type TemplateSummary struct {
	Name string
	Path string
}

func ListTemplates(start string) ([]TemplateSummary, error) {
	templateDir, err := catalogTemplateDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(templateDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	templates := make([]TemplateSummary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".tex" {
			continue
		}
		path, err := filepath.Abs(filepath.Join(templateDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		templates = append(templates, TemplateSummary{
			Name: entry.Name(),
			Path: path,
		})
	}

	sort.Slice(templates, func(left, right int) bool {
		if templates[left].Name == templates[right].Name {
			return templates[left].Path < templates[right].Path
		}
		return templates[left].Name < templates[right].Name
	})
	return templates, nil
}

func TemplateNames(start string) ([]string, error) {
	templates, err := ListTemplates(start)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(templates))
	for _, template := range templates {
		names = append(names, template.Name)
	}
	return names, nil
}

func ResolveTemplateReference(start, reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return "", nil
	}

	startDir, err := templateBaseDir(start)
	if err != nil {
		return "", err
	}

	if looksLikeTemplatePath(reference) {
		return resolveTemplatePath(startDir, reference)
	}

	templateDir, err := catalogTemplateDir()
	if err != nil {
		return "", err
	}
	templates, err := ListTemplates(templateDir)
	if err != nil {
		return "", err
	}

	matches := make([]TemplateSummary, 0, 1)
	for _, template := range templates {
		if template.Name == reference {
			matches = append(matches, template)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("template %q not found; run `invox template list`", reference)
	case 1:
		return matches[0].Path, nil
	default:
		return "", fmt.Errorf("template %q is ambiguous", reference)
	}
}

func catalogTemplateDir() (string, error) {
	configuredTemplatePath, err := resolveConfiguredPath("paths", "template")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(configuredTemplatePath) != "" {
		return filepath.Dir(configuredTemplatePath), nil
	}

	defaultTemplatePath := firstExistingPath(configSearchPaths("template.tex", "invoice_template.tex")...)
	if strings.TrimSpace(defaultTemplatePath) != "" {
		return filepath.Dir(defaultTemplatePath), nil
	}

	globalTemplatePath := GlobalTemplatePath()
	if strings.TrimSpace(globalTemplatePath) == "" {
		return "", fmt.Errorf("template file not found; pass -t/--template with a path, set paths.template in config.yaml, or place template.tex at %s", GlobalTemplatePath())
	}
	return filepath.Dir(globalTemplatePath), nil
}

func templateBaseDir(start string) (string, error) {
	if strings.TrimSpace(start) == "" {
		return os.Getwd()
	}

	resolved, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	switch {
	case err == nil && info.IsDir():
		return resolved, nil
	case err == nil:
		return filepath.Dir(resolved), nil
	case errors.Is(err, os.ErrNotExist):
		if strings.EqualFold(filepath.Ext(resolved), ".tex") {
			return filepath.Dir(resolved), nil
		}
		return resolved, nil
	default:
		return "", err
	}
}

func looksLikeTemplatePath(reference string) bool {
	return filepath.IsAbs(reference) ||
		strings.HasPrefix(reference, ".") ||
		strings.HasPrefix(reference, "~") ||
		strings.Contains(reference, "/") ||
		strings.Contains(reference, `\`)
}

func resolveTemplatePath(startDir, reference string) (string, error) {
	resolved := expandHomePath(reference)
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(startDir, resolved)
	}
	return filepath.Abs(resolved)
}
