package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/iostreams"
)

type listCase struct {
	name       string
	tty        bool
	args       []string
	wantStdout string
	wantStderr string
}

func runListCases(t *testing.T, cases []listCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			ios.SetStdoutTTY(tc.tty)
			exitCode, stdout, stderr := captureRunStreams(t, ios, tc.args)
			if exitCode != 0 || stdout != tc.wantStdout || stderr != tc.wantStderr {
				t.Fatalf("exit=%d\nstdout=%q\nstderr=%q\nwant exit=0\nstdout=%q\nstderr=%q", exitCode, stdout, stderr, tc.wantStdout, tc.wantStderr)
			}
		})
	}
}

func writeListFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", path, err)
	}
}

func TestCustomerListOutput(t *testing.T) {
	dir := t.TempDir()
	chdirForTest(t, dir)
	writeListFile(t, filepath.Join(dir, "customers.yaml"), `ACME:
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
	writeListFile(t, filepath.Join(dir, "empty.yaml"), "{}\n")

	runListCases(t, []listCase{
		{
			name: "pipe",
			args: []string{"customer", "list", "-c", "customers.yaml"},
			wantStdout: "ACME\tAcme\\tTools\\nLtd red C:\\\\x\tactive\n" +
				"CTRL\tabcde\\rfghij\t\n" +
				"EMOJI\t🚀⭐✅👍🏽\tactive\n" +
				"LONG\tVery Long Company Name Gesellschaft mit beschraenkter Haftung\tactive\n" +
				"WIDE\t株式会社テスト\tinactive\n",
		},
		{
			name: "terminal",
			tty:  true,
			args: []string{"customer", "list", "-c", "customers.yaml"},
			wantStdout: "ID     NAME" + strings.Repeat(" ", 38) + "STATUS\n" +
				"ACME   Acme Tools Ltd red C:\\x" + strings.Repeat(" ", 19) + "active\n" +
				"CTRL   abcde fghij\n" +
				"EMOJI  🚀⭐✅👍🏽" + strings.Repeat(" ", 34) + "active\n" +
				"LONG   Very Long Company Name Gesellschaft mit…  active\n" +
				"WIDE   株式会社テスト" + strings.Repeat(" ", 28) + "inactive\n",
		},
		{
			name: "pipe empty",
			args: []string{"customer", "list", "-c", "empty.yaml"},
		},
		{
			name:       "terminal empty",
			tty:        true,
			args:       []string{"customer", "list", "-c", "empty.yaml"},
			wantStderr: "No customers found in empty.yaml\n",
		},
	})
}

func TestArchiveListOutput(t *testing.T) {
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	writeListFile(t, filepath.Join(archiveDir, "a.yaml"), `customer_id: "CUST\t1\n\e[31mX"
invoice:
  issue_date: 2026-03-06
  status: archived
`)

	runListCases(t, []listCase{
		{
			name:       "pipe",
			args:       []string{"archive", "list"},
			wantStdout: "a.yaml\tCUST\\t1\\nX\t2026-03-06\tarchived\n",
		},
		{
			name: "terminal",
			tty:  true,
			args: []string{"archive", "list"},
			wantStdout: "FILE    CUSTOMER  ISSUE DATE  STATUS\n" +
				"a.yaml  CUST 1 X  2026-03-06  archived\n",
		},
	})

	if err := os.Remove(filepath.Join(archiveDir, "a.yaml")); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}
	runListCases(t, []listCase{
		{
			name: "pipe empty",
			args: []string{"archive", "list"},
		},
		{
			name:       "terminal empty",
			tty:        true,
			args:       []string{"archive", "list"},
			wantStderr: "No archived invoices found in " + archiveDir + "\n",
		},
	})
}

func TestTemplateListOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows file names cannot hold tabs, newlines or escape characters")
	}

	dir := t.TempDir()
	writeConfigFile(t, "paths:\n  template: "+quoteYAMLString(filepath.Join(dir, "template.tex"))+"\n")
	writeListFile(t, filepath.Join(dir, "template.tex"), "")
	writeListFile(t, filepath.Join(dir, "a\tb\nc\x1b[31m.tex"), "")

	runListCases(t, []listCase{
		{
			name: "pipe",
			args: []string{"template", "list"},
			wantStdout: "a\\tb\\nc.tex\t" + dir + "/a\\tb\\nc.tex\n" +
				"template.tex\t" + dir + "/template.tex\n",
		},
		{
			name: "terminal",
			tty:  true,
			args: []string{"template", "list"},
			wantStdout: "NAME          PATH\n" +
				"a b c.tex     " + dir + "/a b c.tex\n" +
				"template.tex  " + dir + "/template.tex\n",
		},
		{
			name:       "pipe names",
			args:       []string{"template", "list", "--names"},
			wantStdout: "a\\tb\\nc.tex\ntemplate.tex\n",
		},
		{
			name:       "terminal names",
			tty:        true,
			args:       []string{"template", "list", "--names"},
			wantStdout: "NAME\na b c.tex\ntemplate.tex\n",
		},
	})
}

func TestTemplateListOutputEmpty(t *testing.T) {
	dir := t.TempDir()
	writeConfigFile(t, "paths:\n  template: "+quoteYAMLString(filepath.Join(dir, "template.tex"))+"\n")

	runListCases(t, []listCase{
		{
			name: "pipe",
			args: []string{"template", "list"},
		},
		{
			name:       "terminal",
			tty:        true,
			args:       []string{"template", "list"},
			wantStderr: "No templates found in " + dir + "\n",
		},
	})
}
