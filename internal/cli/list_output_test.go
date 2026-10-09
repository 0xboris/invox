package cli_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

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
