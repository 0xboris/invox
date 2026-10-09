package cli_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestCustomerListOutput(t *testing.T) {
	x := clitest.New(t)

	dir := t.TempDir()
	x.Chdir(dir)
	testfixture.WriteFile(t, filepath.Join(dir, "customers.yaml"), `ACME:
  name: "Acme\tTools\nLtd \e[31mred\e[0m C:\\x"
  status: active
CTRL:
  name: "a\x01b\bc\x7fd\Ne\rf\u202eg\u2066h\u200bi\u00adj"
EMOJI:
  name: "🚀⭐✅👍🏽"
  status: active
LONG:
  name: Very Long Company Name Gesellschaft mit beschraenkter Haftung
  status: active
WIDE:
  name: 株式会社テスト
  status: inactive
`)
	testfixture.WriteFile(t, filepath.Join(dir, "empty.yaml"), "{}\n")

	x.RunOutputCases(t, []clitest.OutputCase{
		{
			Name: "pipe",
			Args: []string{"customer", "list", "-c", "customers.yaml"},
			WantStdout: "ACME\tAcme\\tTools\\nLtd red C:\\\\x\tactive\n" +
				"CTRL\tabcde\\rfghij\t\n" +
				"EMOJI\t🚀⭐✅👍🏽\tactive\n" +
				"LONG\tVery Long Company Name Gesellschaft mit beschraenkter Haftung\tactive\n" +
				"WIDE\t株式会社テスト\tinactive\n",
		},
		{
			Name: "terminal",
			TTY:  true,
			Args: []string{"customer", "list", "-c", "customers.yaml"},
			WantStdout: "ID     NAME" + strings.Repeat(" ", 38) + "STATUS\n" +
				"ACME   Acme Tools Ltd red C:\\x" + strings.Repeat(" ", 19) + "active\n" +
				"CTRL   abcde fghij\n" +
				"EMOJI  🚀⭐✅👍🏽" + strings.Repeat(" ", 34) + "active\n" +
				"LONG   Very Long Company Name Gesellschaft mit…  active\n" +
				"WIDE   株式会社テスト" + strings.Repeat(" ", 28) + "inactive\n",
		},
		{
			Name: "pipe empty",
			Args: []string{"customer", "list", "-c", "empty.yaml"},
		},
		{
			Name:       "terminal empty",
			TTY:        true,
			Args:       []string{"customer", "list", "-c", "empty.yaml"},
			WantStderr: "No customers found in empty.yaml\n",
		},
	})
}

func TestArchiveListOutput(t *testing.T) {
	x := clitest.New(t)

	archiveDir := t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	testfixture.WriteFile(t, filepath.Join(archiveDir, "a.yaml"), `customer_id: "CUST\t1\n\e[31mX"
invoice:
  issue_date: 2026-03-06
  status: archived
`)

	x.RunOutputCases(t, []clitest.OutputCase{
		{
			Name:       "pipe",
			Args:       []string{"archive", "list"},
			WantStdout: "a.yaml\tCUST\\t1\\nX\t2026-03-06\tarchived\n",
		},
		{
			Name: "terminal",
			TTY:  true,
			Args: []string{"archive", "list"},
			WantStdout: "FILE    CUSTOMER  ISSUE DATE  STATUS\n" +
				"a.yaml  CUST 1 X  2026-03-06  archived\n",
		},
	})

	if err := os.Remove(filepath.Join(archiveDir, "a.yaml")); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}
	x.RunOutputCases(t, []clitest.OutputCase{
		{
			Name: "pipe empty",
			Args: []string{"archive", "list"},
		},
		{
			Name:       "terminal empty",
			TTY:        true,
			Args:       []string{"archive", "list"},
			WantStderr: "No archived invoices found in " + archiveDir + "\n",
		},
	})
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
