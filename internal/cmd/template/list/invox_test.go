package list_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
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

func TestTemplateListOutput(t *testing.T) {
	x := clitest.New(t)

	if runtime.GOOS == "windows" {
		t.Skip("Windows file names cannot hold tabs, newlines or escape characters")
	}

	dir := t.TempDir()
	x.WriteConfig("paths:\n  template: " + testfixture.QuoteYAML(filepath.Join(dir, "template.tex")) + "\n")
	testfixture.WriteFile(t, filepath.Join(dir, "template.tex"), "")
	testfixture.WriteFile(t, filepath.Join(dir, "a\tb\nc\x1b[31m.tex"), "")

	x.RunOutputCases(t, []clitest.OutputCase{
		{
			Name: "pipe",
			Args: []string{"template", "list"},
			WantStdout: "a\\tb\\nc.tex\t" + dir + "/a\\tb\\nc.tex\n" +
				"template.tex\t" + dir + "/template.tex\n",
		},
		{
			Name: "terminal",
			TTY:  true,
			Args: []string{"template", "list"},
			WantStdout: "NAME          PATH\n" +
				"a b c.tex     " + dir + "/a b c.tex\n" +
				"template.tex  " + dir + "/template.tex\n",
		},
		{
			Name:       "pipe names",
			Args:       []string{"template", "list", "--names"},
			WantStdout: "a\\tb\\nc.tex\ntemplate.tex\n",
		},
		{
			Name:       "terminal names",
			TTY:        true,
			Args:       []string{"template", "list", "--names"},
			WantStdout: "NAME\na b c.tex\ntemplate.tex\n",
		},
	})
}

func TestTemplateListOutputEmpty(t *testing.T) {
	x := clitest.New(t)

	dir := t.TempDir()
	x.WriteConfig("paths:\n  template: " + testfixture.QuoteYAML(filepath.Join(dir, "template.tex")) + "\n")

	x.RunOutputCases(t, []clitest.OutputCase{
		{
			Name: "pipe",
			Args: []string{"template", "list"},
		},
		{
			Name:       "terminal",
			TTY:        true,
			Args:       []string{"template", "list"},
			WantStderr: "No templates found in " + dir + "\n",
		},
	})
}
