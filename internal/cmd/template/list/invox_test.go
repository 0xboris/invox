package list_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
)

func TestTemplateListPrintsAvailableTemplates(t *testing.T) {
	x := clitest.New(t)

	configPath := x.WriteConfig("")
	configDir := filepath.Dir(configPath)
	workDir := t.TempDir()
	x.Chdir(workDir)

	for _, path := range []string{
		filepath.Join(configDir, "multi_vat.tex"),
		filepath.Join(configDir, "template.tex"),
		filepath.Join(workDir, "project.tex"),
	} {
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", path, err)
		}
	}

	exitCode, stdout, stderr := x.Run([]string{"template", "list"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	// Piped output escapes the backslashes in Windows paths.
	tsvPath := func(path string) string { return strings.ReplaceAll(path, `\`, `\\`) }
	for _, want := range []string{
		"multi_vat.tex\t" + tsvPath(filepath.Join(configDir, "multi_vat.tex")),
		"template.tex\t" + tsvPath(filepath.Join(configDir, "template.tex")),
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
	if strings.Contains(stdout, "project.tex\t"+tsvPath(filepath.Join(workDir, "project.tex"))) {
		t.Fatalf("stdout %q should not contain project template outside default template dir", stdout)
	}
}
