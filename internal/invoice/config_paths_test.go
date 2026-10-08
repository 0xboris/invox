package invoice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveDefaultPathsPreferLocalProjectFilesOverGlobalConfig(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	configHome := filepath.Join(rootDir, "config-home")
	configDir := filepath.Join(configHome, "invox")
	customDir := filepath.Join(rootDir, "custom")
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

	for _, name := range []string{"customers.yaml", "issuer.yaml", "invoice_defaults.yaml", "template.tex"} {
		path := filepath.Join(configDir, name)
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", path, err)
		}
	}
	for _, name := range []string{"customers.yaml", "issuer.yaml", "invoice_defaults.yaml", "template.tex"} {
		path := filepath.Join(workDir, name)
		if err := os.WriteFile(path, []byte("local"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", path, err)
		}
		path = filepath.Join(customDir, name)
		if err := os.WriteFile(path, []byte("custom"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", path, err)
		}
	}

	configSource := strings.TrimSpace(`
paths:
  customers: ../../custom/customers.yaml
  issuer: ../../custom/issuer.yaml
  defaults: ../../custom/invoice_defaults.yaml
  template: ../../custom/template.tex
`) + "\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configSource), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	h := testHost(configHome, filepath.Join(t.TempDir(), "home"))

	opts := resolveDefaultOptions(t, h, workDir)

	if opts.CustomersPath != filepath.Join(workDir, "customers.yaml") {
		t.Fatalf("CustomersPath = %q, want local project path", opts.CustomersPath)
	}
	if opts.IssuerPath != filepath.Join(workDir, "issuer.yaml") {
		t.Fatalf("IssuerPath = %q, want local project path", opts.IssuerPath)
	}
	if opts.DefaultsPath != filepath.Join(workDir, "invoice_defaults.yaml") {
		t.Fatalf("DefaultsPath = %q, want local project path", opts.DefaultsPath)
	}
	if opts.TemplatePath != filepath.Join(workDir, "template.tex") {
		t.Fatalf("TemplatePath = %q, want local project path", opts.TemplatePath)
	}
}

func TestResolveDefaultPathsFallbackToGlobalConfigFiles(t *testing.T) {
	t.Parallel()

	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	workDir := filepath.Join(t.TempDir(), "work")
	h := testHost(configHome, filepath.Join(t.TempDir(), "home"))
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workDir) returned error: %v", err)
	}

	for _, name := range []string{"customers.yaml", "issuer.yaml", "invoice_defaults.yaml", "template.tex"} {
		path := filepath.Join(configDir, name)
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", path, err)
		}
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(h.defaultConfigTemplate()), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	opts := resolveDefaultOptions(t, h, workDir)

	if opts.CustomersPath != filepath.Join(configDir, "customers.yaml") {
		t.Fatalf("CustomersPath = %q, want global config path", opts.CustomersPath)
	}
	if opts.IssuerPath != filepath.Join(configDir, "issuer.yaml") {
		t.Fatalf("IssuerPath = %q, want global config path", opts.IssuerPath)
	}
	if opts.DefaultsPath != filepath.Join(configDir, "invoice_defaults.yaml") {
		t.Fatalf("DefaultsPath = %q, want global config path", opts.DefaultsPath)
	}
	if opts.TemplatePath != filepath.Join(configDir, "template.tex") {
		t.Fatalf("TemplatePath = %q, want global config path", opts.TemplatePath)
	}
}

func TestResolveDefaultPathsUseConfiguredPathsBeforeGlobalDefaults(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	configHome := filepath.Join(rootDir, "config-home")
	configDir := filepath.Join(configHome, "invox")
	customDir := filepath.Join(rootDir, "custom")
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

	for _, name := range []string{"customers.yaml", "issuer.yaml", "invoice_defaults.yaml", "template.tex"} {
		if err := os.WriteFile(filepath.Join(customDir, name), []byte("custom"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(configDir, name), []byte("global"), 0o644); err != nil {
			t.Fatalf("WriteFile(global %s) returned error: %v", name, err)
		}
	}

	configSource := strings.TrimSpace(`
paths:
  customers: ../../custom/customers.yaml
  issuer: ../../custom/issuer.yaml
  defaults: ../../custom/invoice_defaults.yaml
  template: ../../custom/template.tex
`) + "\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configSource), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	h := testHost(configHome, filepath.Join(t.TempDir(), "home"))

	opts := resolveDefaultOptions(t, h, workDir)

	if opts.CustomersPath != filepath.Join(customDir, "customers.yaml") {
		t.Fatalf("CustomersPath = %q, want configured path", opts.CustomersPath)
	}
	if opts.IssuerPath != filepath.Join(customDir, "issuer.yaml") {
		t.Fatalf("IssuerPath = %q, want configured path", opts.IssuerPath)
	}
	if opts.DefaultsPath != filepath.Join(customDir, "invoice_defaults.yaml") {
		t.Fatalf("DefaultsPath = %q, want configured path", opts.DefaultsPath)
	}
	if opts.TemplatePath != filepath.Join(customDir, "template.tex") {
		t.Fatalf("TemplatePath = %q, want configured path", opts.TemplatePath)
	}
}

func TestResolveDefaultPathsFallbackToLegacyConfigFiles(t *testing.T) {
	t.Parallel()

	configHome := filepath.Join(t.TempDir(), "config-home")
	legacyDir := filepath.Join(configHome, "invoice-tool")
	workDir := filepath.Join(t.TempDir(), "work")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(legacyDir) returned error: %v", err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workDir) returned error: %v", err)
	}

	for _, name := range []string{"customers.yaml", "issuer.yaml", "invoice_defaults.yaml", "template.tex"} {
		path := filepath.Join(legacyDir, name)
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", path, err)
		}
	}

	h := testHost(configHome, filepath.Join(t.TempDir(), "home"))

	opts := resolveDefaultOptions(t, h, workDir)

	if opts.CustomersPath != filepath.Join(legacyDir, "customers.yaml") {
		t.Fatalf("CustomersPath = %q, want legacy config path", opts.CustomersPath)
	}
	if opts.IssuerPath != filepath.Join(legacyDir, "issuer.yaml") {
		t.Fatalf("IssuerPath = %q, want legacy config path", opts.IssuerPath)
	}
	if opts.DefaultsPath != filepath.Join(legacyDir, "invoice_defaults.yaml") {
		t.Fatalf("DefaultsPath = %q, want legacy config path", opts.DefaultsPath)
	}
	if opts.TemplatePath != filepath.Join(legacyDir, "template.tex") {
		t.Fatalf("TemplatePath = %q, want legacy config path", opts.TemplatePath)
	}
}

func TestResolveArchiveDirDefaultsToPlatformDataDir(t *testing.T) {
	t.Parallel()

	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	homeDir := filepath.Join(t.TempDir(), "home")
	h := testHost(configHome, homeDir)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(homeDir) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(h.defaultConfigTemplate()), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}
	expected := filepath.Join(homeDir, ".local", "share", "invox", "invoices")

	got, err := h.ResolveArchiveDir()
	if err != nil {
		t.Fatalf("ResolveArchiveDir returned error: %v", err)
	}
	if got != expected {
		t.Fatalf("ResolveArchiveDir() = %q, want %q", got, expected)
	}
}

func TestResolveArchiveDirUsesConfigOverride(t *testing.T) {
	t.Parallel()

	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	homeDir := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(homeDir) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("archive:\n  dir: ~/Documents/Invox/Invoices\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	h := testHost(configHome, homeDir)

	got, err := h.ResolveArchiveDir()
	if err != nil {
		t.Fatalf("ResolveArchiveDir returned error: %v", err)
	}

	want := filepath.Join(homeDir, "Documents", "Invox", "Invoices")
	if got != want {
		t.Fatalf("ResolveArchiveDir() = %q, want %q", got, want)
	}
}

func TestResolveArchiveDirUsesRelativeConfigPath(t *testing.T) {
	t.Parallel()

	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("archive:\n  dir: archive\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	h := testHost(configHome, filepath.Join(t.TempDir(), "home"))

	got, err := h.ResolveArchiveDir()
	if err != nil {
		t.Fatalf("ResolveArchiveDir returned error: %v", err)
	}

	want := filepath.Join(configDir, "archive")
	if got != want {
		t.Fatalf("ResolveArchiveDir() = %q, want %q", got, want)
	}
}

func TestResolveArchiveDirUsesLegacyConfigOverride(t *testing.T) {
	t.Parallel()

	configHome := filepath.Join(t.TempDir(), "config-home")
	legacyDir := filepath.Join(configHome, "invoice-tool")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(legacyDir) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "config.yaml"), []byte("archive:\n  dir: archived-invoices\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	h := testHost(configHome, filepath.Join(t.TempDir(), "home"))

	got, err := h.ResolveArchiveDir()
	if err != nil {
		t.Fatalf("ResolveArchiveDir returned error: %v", err)
	}

	want := filepath.Join(legacyDir, "archived-invoices")
	if got != want {
		t.Fatalf("ResolveArchiveDir() = %q, want %q", got, want)
	}
}

func TestEditableConfigPathCreatesCommentedTemplate(t *testing.T) {
	t.Parallel()

	configHome := filepath.Join(t.TempDir(), "config-home")
	homeDir := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(homeDir) returned error: %v", err)
	}

	h := testHost(configHome, homeDir)
	archiveDir := filepath.Join(homeDir, ".local", "share", "invox", "invoices")

	path, err := h.EditableConfigPath()
	if err != nil {
		t.Fatalf("EditableConfigPath returned error: %v", err)
	}

	wantPath := filepath.Join(configHome, "invox", "config.yaml")
	if path != wantPath {
		t.Fatalf("EditableConfigPath() = %q, want %q", path, wantPath)
	}

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(config.yaml) returned error: %v", err)
	}

	text := string(source)
	for _, want := range []string{
		"# Invox user configuration.",
		"# Supported settings:",
		"# - Top-level keys must not be indented.",
		"#   paths.customers",
		"#   paths.issuer",
		"#   paths.defaults",
		"#   paths.template",
		"#   numbering.pattern",
		"#   numbering.start",
		"#   archive.dir",
		"#   email.subject",
		"#   email.body",
		"#       {email_greeting}",
		"#       {contact_person}",
		"# - Per-customer numbering overrides live in customers.yaml at:",
		"#   <customer>.numbering.start",
		"# paths:",
		"#   customers: 'customers.yaml'",
		"#   issuer: 'issuer.yaml'",
		"#   defaults: 'invoice_defaults.yaml'",
		"#   template: 'template.tex'",
		"# numbering:",
		"#   pattern: '{customer_code}-{counter:03}'",
		"#   start: 1",
		"# archive:",
		"#   dir: '" + h.configTemplatePath(archiveDir) + "'",
		"# email:",
		"#   subject: 'Invoice {invoice_number}'",
		"#   body: |",
		"#     Please find attached invoice {invoice_number}.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("config template %q does not contain %q", text, want)
		}
	}
}

func TestEditableConfigPathPreservesExistingConfig(t *testing.T) {
	t.Parallel()

	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}

	h := testHost(configHome, filepath.Join(t.TempDir(), "home"))

	want := "archive:\n  dir: ~/Documents/Invox/Invoices\n"
	path := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}

	gotPath, err := h.EditableConfigPath()
	if err != nil {
		t.Fatalf("EditableConfigPath returned error: %v", err)
	}
	if gotPath != path {
		t.Fatalf("EditableConfigPath() = %q, want %q", gotPath, path)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(config.yaml) returned error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("config.yaml = %q, want %q", string(got), want)
	}
}

func TestResolveDefaultCustomersPathRejectsIndentedTopLevelConfig(t *testing.T) {
	t.Parallel()

	h := writeConfigFile(t, " numbering:\n  pattern: '{customer_id}-{counter:03}'\npaths:\n  customers: '~/customers.yaml'\n")

	_, err := h.ResolveSupportFile(Customers, t.TempDir())
	if err == nil {
		t.Fatalf("ResolveSupportFile returned nil error for indented top-level config")
	}
	if !strings.Contains(err.Error(), "top-level keys must not be indented") {
		t.Fatalf("error %q does not contain top-level indentation message", err.Error())
	}
}

// resolveDefaultOptions resolves the four support files from start the way
// the CLI does before flags override them.
func resolveDefaultOptions(t *testing.T, h Host, start string) Options {
	t.Helper()
	var opts Options
	for _, resolve := range []struct {
		path *string
		kind SupportFile
	}{
		{&opts.CustomersPath, Customers},
		{&opts.IssuerPath, Issuer},
		{&opts.DefaultsPath, Defaults},
		{&opts.TemplatePath, Template},
	} {
		found, err := h.ResolveSupportFile(resolve.kind, start)
		if err != nil {
			t.Fatalf("ResolveSupportFile returned error: %v", err)
		}
		*resolve.path = found.Path
	}
	return opts
}
