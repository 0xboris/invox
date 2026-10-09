package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/txtar"
)

// portOrderScenario runs invox in a directory laid out like the archive
// testscript: the support files under home/.config/invox, with archive.dir
// set to its archive subdirectory, plus files, where a name ending in "/" is
// a directory.
type portOrderScenario struct {
	name       string
	files      map[string]string
	cwd        string
	args       []string
	wantExit   int
	wantStdout string
	// wantStderr has backup timestamps scrubbed to <stamp> and {ENOENT}
	// for the OS's text of a missing file.
	wantStderr string
	check      func(t *testing.T, work string)
}

const portOrderConfig = "home/.config/invox/"
const portOrderArchive = portOrderConfig + "archive/"

var backupStamp = regexp.MustCompile(`\.[0-9]{8}T[0-9]{6}Z(-[0-9]+)?\.`)

// TestPortOrder pins which error a user sees first where a use case checks
// a domain rule and a file at once, and the bytes and mode of the files
// archiving writes. docs/design/ports-narrowing.md lists these scenarios.
func TestPortOrder(t *testing.T) {
	fixtures := archiveScriptFixtures(t)
	built := func(number, status string) string {
		source := strings.Replace(fixtures["invoice.yaml"], "status: draft", "status: "+status, 1)
		return strings.Replace(source, "CUST-001-001", number, 1)
	}
	oldMarkdown := "---\ncustomer_id: CUST-001\ninvoice:\n  number: CUST-001-001\n---\n"
	badDueIssuer := strings.Replace(fixtures[portOrderConfig+"issuer.yaml"], "due_days: 30", "due_days: -1", 1)

	scenarios := []portOrderScenario{
		{
			name:       "status-before-resolve",
			files:      map[string]string{"invoice.yaml": portOrderWorkingCopy("CUST-001-001", "draft", "/abs/x.yaml", "")},
			args:       []string{"archive", "add", "invoice.yaml"},
			wantExit:   1,
			wantStderr: "error: invoice.yaml: invoice.status must be `editing` or `built` before re-archiving, got `draft`\n",
		},
		// archive_replace_path was written only for Markdown working
		// copies; a stale one is ignored.
		{
			name:       "replace-resolve-no-number",
			files:      map[string]string{"invoice.yaml": portOrderWorkingCopy("", "editing", "invoice.yaml", "../escape.md")},
			args:       []string{"archive", "add", "invoice.yaml"},
			wantStdout: "home/.config/invox/archive/invoice.yaml\n",
			wantStderr: "Archived invoice.yaml -> home/.config/invox/archive/invoice.yaml\n",
		},
		{
			name:       "replace-resolve-with-number",
			files:      map[string]string{"invoice.yaml": portOrderWorkingCopy("CUST-001-001", "editing", "invoice.yaml", "../escape.md")},
			args:       []string{"archive", "add", "invoice.yaml"},
			wantStdout: "home/.config/invox/archive/invoice.yaml\n",
			wantStderr: "Archived invoice.yaml -> home/.config/invox/archive/invoice.yaml\n",
		},
		{
			name: "target-dir-vs-duplicate",
			files: map[string]string{
				"invoice.yaml":                     portOrderWorkingCopy("CUST-001-001", "editing", "invoice.yaml", ""),
				portOrderArchive + "other.yaml":    built("CUST-001-001", "built"),
				portOrderArchive + "invoice.yaml/": "",
			},
			args:     []string{"archive", "add", "invoice.yaml", "--yes"},
			wantExit: 1,
			wantStderr: "error: invoice.yaml: invoice number CUST-001-001 is already used by archived invoice home/.config/invox/archive/other.yaml\n" +
				"Run 'invox increment -i invoice.yaml' to give it the next free number, then archive it again.\n",
		},
		{
			name: "exists-vs-duplicate",
			files: map[string]string{
				"invoice.yaml":                    built("CUST-001-001", "built"),
				portOrderArchive + "invoice.yaml": built("X-1", "built"),
				portOrderArchive + "other.yaml":   built("CUST-001-001", "built"),
			},
			args:       []string{"archive", "add", "invoice.yaml"},
			wantExit:   1,
			wantStderr: "error: home/.config/invox/archive/invoice.yaml already exists\n",
		},
		{
			name:       "already-in-archive",
			files:      map[string]string{portOrderArchive + "invoice.yaml": portOrderWorkingCopy("CUST-001-001", "editing", "invoice.yaml", "")},
			cwd:        portOrderArchive,
			args:       []string{"archive", "add", "invoice.yaml", "--yes"},
			wantExit:   1,
			wantStderr: "error: invoice.yaml is already in the archive directory\n",
		},
		{
			name: "rearchive-yes",
			files: map[string]string{
				"invoice.yaml":                    portOrderWorkingCopy("CUST-001-001", "editing", "invoice.yaml", "old.md"),
				portOrderArchive + "invoice.yaml": built("CUST-001-001", "archived"),
				portOrderArchive + "old.md":       oldMarkdown,
			},
			args:       []string{"archive", "add", "invoice.yaml", "--yes"},
			wantStdout: "home/.config/invox/archive/invoice.yaml\n",
			wantStderr: "warning: 1 Markdown invoice in home/.config/invox/archive is no longer read; convert it to .yaml to include it:\n" +
				"  home/.config/invox/archive/old.md\n" +
				"Replaced archived invoice home/.config/invox/archive/invoice.yaml; previous version kept at home/.config/invox/archive/.history/invoice<stamp>yaml\n" +
				"Archived invoice.yaml -> home/.config/invox/archive/invoice.yaml\n",
			check: func(t *testing.T, work string) {
				archived := filepath.Join(work, portOrderArchive, "invoice.yaml")
				want := "customer_id: CUST-001\ninvoice:\n  number: CUST-001-001\n  issue_date: 2026-03-06\n  due_date: 2026-04-05\n" +
					"  status: archived\n  period: March\n  vat_percent: 20\n  paid_amount: 0\npositions:\n  - name: Dev\n" +
					"    description: Work\n    unit_price: 100\n    quantity: 1\n"
				if got := readFileForTest(t, archived); got != want {
					t.Errorf("archived invoice =\n%s\nwant\n%s", got, want)
				}
				assertPortOrderMode(t, archived, 0o640)
				assertPortOrderGone(t, filepath.Join(work, "invoice.yaml"))
				if got := readFileForTest(t, filepath.Join(work, portOrderArchive, "old.md")); got != oldMarkdown {
					t.Errorf("old.md = %q, want it untouched", got)
				}
				history, err := os.ReadDir(filepath.Join(work, portOrderArchive, ".history"))
				if err != nil || len(history) != 1 {
					t.Errorf(".history = %v, %v; want the replaced invoice.yaml", history, err)
				}
			},
		},
		{
			name:       "archive-comments-and-empty-link",
			files:      map[string]string{"invoice.yaml": "# keep me\n" + built("CUST-001-001", "built") + "_invox: {}\n"},
			args:       []string{"archive", "add", "invoice.yaml"},
			wantStdout: "home/.config/invox/archive/invoice.yaml\n",
			wantStderr: "Archived invoice.yaml -> home/.config/invox/archive/invoice.yaml\n",
			check: func(t *testing.T, work string) {
				archived := filepath.Join(work, portOrderArchive, "invoice.yaml")
				want := "# keep me\n" + built("CUST-001-001", "archived")
				if got := readFileForTest(t, archived); got != want {
					t.Errorf("archived invoice =\n%s\nwant\n%s", got, want)
				}
				assertPortOrderMode(t, archived, 0o600)
				assertPortOrderGone(t, filepath.Join(work, "invoice.yaml"))
			},
		},
		{
			name: "new-default-exists-and-bad-due",
			files: map[string]string{
				portOrderConfig + "issuer.yaml": badDueIssuer,
				"CUST-001-010.yaml":             "x: 1\n",
			},
			args:       []string{"new", "CUST-001"},
			wantExit:   1,
			wantStderr: "error: CUST-001-010.yaml already exists; pass --force to replace it or choose a different -o/--output path\n",
		},
		{
			name:       "new-output-in-other-dir-drafts",
			files:      map[string]string{"sub/d.yaml": built("CUST-001-011", "draft")},
			args:       []string{"new", "CUST-001", "-o", "sub/n.yaml"},
			wantStdout: "sub/n.yaml\n",
			wantStderr: "Created sub/n.yaml for CUST-001 (CUST-001-012)\n",
		},
		{
			name:       "email-upper-PDF",
			files:      map[string]string{"INV.yaml": built("CUST-001-001", "built")},
			args:       []string{"email", "INV.PDF", "-o", "d.eml", "-n"},
			wantExit:   1,
			wantStderr: "error: read INV.pdf: stat INV.pdf: {ENOENT}\n",
		},
		{
			name: "email-pdf-with-p",
			files: map[string]string{
				"invoice.pdf":  "%PDF\n",
				"other.pdf":    "%PDF\n",
				"invoice.yaml": built("CUST-001-001", "built"),
			},
			args:       []string{"email", "invoice.pdf", "-p", "other.pdf", "-o", "d.eml", "-n"},
			wantStdout: "d.eml\n",
			wantStderr: "Would open email draft for CUST-001 (CUST-001-001) to office@appsters.example\n" +
				"Subject: Invoice CUST-001-001\nAttachment: other.pdf\nWould write the draft to d.eml\n",
		},
		{
			name:       "email-missing-pdf-and-bad-subject",
			files:      map[string]string{"invoice.yaml": built("CUST-001-001", "built")},
			args:       []string{"email", "invoice.yaml", "-o", "d.eml", "--subject", "{nope}"},
			wantExit:   1,
			wantStderr: "error: read invoice.pdf: stat invoice.pdf: {ENOENT}\n",
		},
		{
			name: "email-dry-run-output-exists-bad-subject",
			files: map[string]string{
				"invoice.yaml": built("CUST-001-001", "built"),
				"invoice.pdf":  "%PDF\n",
				"d.eml":        "x",
			},
			args:       []string{"email", "invoice.yaml", "-o", "d.eml", "-n", "--subject", "{nope}"},
			wantExit:   1,
			wantStderr: "error: d.eml already exists; pass --force or choose another -o path\n",
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			work := portOrderLayout(t, fixtures, sc.files)
			chdirForTest(t, filepath.Join(work, filepath.FromSlash(sc.cwd)))
			exitCode, stdout, stderr := captureRun(t, sc.args)
			stdout = filepath.ToSlash(stdout)
			stderr = backupStamp.ReplaceAllString(filepath.ToSlash(stderr), "<stamp>")
			wantStderr := strings.ReplaceAll(sc.wantStderr, "{ENOENT}", missingFileText(t))
			if exitCode != sc.wantExit || stdout != sc.wantStdout || stderr != wantStderr {
				t.Fatalf("invox %s:\nexit %d, stdout %q, stderr %q\nwant exit %d, stdout %q, stderr %q",
					strings.Join(sc.args, " "), exitCode, stdout, stderr, sc.wantExit, sc.wantStdout, wantStderr)
			}
			if sc.check != nil {
				sc.check(t, work)
			}
		})
	}
}

// archiveScriptFixtures returns the input files of the archive testscript.
func archiveScriptFixtures(t *testing.T) map[string]string {
	t.Helper()

	archive, err := txtar.ParseFile(filepath.Join("..", "..", "cmd", "invox", "testdata", "script", "archive.txtar"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures := make(map[string]string)
	for _, file := range archive.Files {
		if !strings.HasPrefix(file.Name, "want") {
			fixtures[file.Name] = string(file.Data)
		}
	}
	return fixtures
}

// portOrderLayout writes fixtures and then files into a new directory, which
// it points HOME and XDG_CONFIG_HOME at, and returns it.
func portOrderLayout(t *testing.T, fixtures, files map[string]string) string {
	t.Helper()

	work, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, content string) {
		path := filepath.Join(work, filepath.FromSlash(name))
		if strings.HasSuffix(name, "/") {
			if err := os.MkdirAll(path, 0o755); err != nil {
				t.Fatal(err)
			}
			return
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0o640); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range fixtures {
		write(name, content)
	}
	for name, content := range files {
		write(name, content)
	}
	home := filepath.Join(work, "home")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	return work
}

// portOrderWorkingCopy is a working copy from `archive edit`, linked to
// archivePath and, when set, replacePath.
func portOrderWorkingCopy(number, status, archivePath, replacePath string) string {
	var b strings.Builder
	b.WriteString("customer_id: CUST-001\ninvoice:\n")
	if number != "" {
		b.WriteString("  number: " + number + "\n")
	}
	b.WriteString("  issue_date: 2026-03-06\n  due_date: 2026-04-05\n  status: " + status + "\n")
	b.WriteString("  period: March\n  vat_percent: 20\n  paid_amount: 0\npositions:\n  - name: Dev\n")
	b.WriteString("    description: Work\n    unit_price: 100\n    quantity: 1\n")
	b.WriteString("_invox:\n  archive_path: " + archivePath + "\n")
	if replacePath != "" {
		b.WriteString("  archive_replace_path: " + replacePath + "\n")
	}
	return b.String()
}

// missingFileText is the OS's text for a file that does not exist.
func missingFileText(t *testing.T) string {
	t.Helper()

	_, err := os.Stat(filepath.Join(t.TempDir(), "missing"))
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("Stat(missing) = %v, want a *fs.PathError", err)
	}
	return pathErr.Err.Error()
}

func assertPortOrderMode(t *testing.T, path string, want fs.FileMode) {
	t.Helper()

	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("%s mode = %o, want %o", path, got, want)
	}
}

func assertPortOrderGone(t *testing.T, paths ...string) {
	t.Helper()

	for _, path := range paths {
		if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s should be gone, Stat err = %v", path, err)
		}
	}
}
