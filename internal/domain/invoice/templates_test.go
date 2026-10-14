package invoice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListTemplatesUsesDefaultTemplateDirectoryOnly(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	configHome := filepath.Join(rootDir, "config-home")
	configDir := filepath.Join(configHome, "invox")
	customDir := filepath.Join(rootDir, "custom-templates")
	workDir := filepath.Join(rootDir, "work")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(customDir) returned error: %v", err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workDir) returned error: %v", err)
	}

	for _, file := range []string{
		filepath.Join(workDir, "project.tex"),
		filepath.Join(configDir, "template.tex"),
		filepath.Join(customDir, "custom.tex"),
		filepath.Join(customDir, "multi_vat.tex"),
	} {
		if err := os.WriteFile(file, []byte("test"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", file, err)
		}
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(strings.TrimSpace(`
paths:
  template: ../../custom-templates/custom.tex
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	h := testHost(configHome, filepath.Join(t.TempDir(), "home"))

	templates, err := h.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates returned error: %v", err)
	}

	got := make([]string, 0, len(templates))
	for _, template := range templates {
		got = append(got, template.Name+"\t"+template.Path)
	}
	for _, want := range []string{
		"custom.tex\t" + filepath.Join(customDir, "custom.tex"),
		"multi_vat.tex\t" + filepath.Join(customDir, "multi_vat.tex"),
	} {
		if !containsString(got, want) {
			t.Fatalf("template list %q does not contain %q", got, want)
		}
	}
	for _, forbidden := range []string{
		"project.tex\t" + filepath.Join(workDir, "project.tex"),
		"template.tex\t" + filepath.Join(configDir, "template.tex"),
	} {
		if containsString(got, forbidden) {
			t.Fatalf("template list %q should not contain %q", got, forbidden)
		}
	}
}

func TestResolveTemplateReferenceFindsNamedConfigTemplate(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	configHome := filepath.Join(rootDir, "config-home")
	configDir := filepath.Join(configHome, "invox")
	customDir := filepath.Join(rootDir, "custom-templates")
	workDir := filepath.Join(rootDir, "work")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(customDir) returned error: %v", err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workDir) returned error: %v", err)
	}

	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(strings.TrimSpace(`
paths:
  template: ../../custom-templates/custom.tex
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	templatePath := filepath.Join(customDir, "multi_vat.tex")
	if err := os.WriteFile(templatePath, []byte("test"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(customDir, "custom.tex"), []byte("test"), 0o644); err != nil {
		t.Fatalf("WriteFile(custom.tex) returned error: %v", err)
	}

	h := testHost(configHome, filepath.Join(t.TempDir(), "home"))

	resolvedPath, err := h.ResolveTemplateReference(workDir, "multi_vat.tex")
	if err != nil {
		t.Fatalf("ResolveTemplateReference returned error: %v", err)
	}
	if resolvedPath != templatePath {
		t.Fatalf("resolvedPath = %q, want %q", resolvedPath, templatePath)
	}
}

func TestResolveTemplateReferenceDoesNotSearchOutsideDefaultTemplateDirectory(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	configHome := filepath.Join(rootDir, "config-home")
	configDir := filepath.Join(configHome, "invox")
	customDir := filepath.Join(rootDir, "custom-templates")
	workDir := filepath.Join(rootDir, "work")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(customDir) returned error: %v", err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workDir) returned error: %v", err)
	}

	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(strings.TrimSpace(`
paths:
  template: ../../custom-templates/custom.tex
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(customDir, "custom.tex"), []byte("test"), 0o644); err != nil {
		t.Fatalf("WriteFile(custom.tex) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "outside.tex"), []byte("test"), 0o644); err != nil {
		t.Fatalf("WriteFile(outside.tex) returned error: %v", err)
	}

	h := testHost(configHome, filepath.Join(t.TempDir(), "home"))

	_, err := h.ResolveTemplateReference(workDir, "outside.tex")
	if err == nil {
		t.Fatal("ResolveTemplateReference returned nil error for template outside default template directory")
	}
	if !strings.Contains(err.Error(), "template \"outside.tex\" not found") {
		t.Fatalf("error %q does not contain not-found message", err.Error())
	}
}
