package cli_test

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/testfixture"
)

// snapshotDirs records every file and directory below dirs with its content
// and modification time, so a test can show that a command wrote nothing.
func snapshotDirs(t *testing.T, dirs ...string) map[string]string {
	t.Helper()

	snapshot := make(map[string]string)
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if entry.IsDir() {
				snapshot[path] = "dir"
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			snapshot[path] = fmt.Sprintf("%d %o %q", info.ModTime().UnixNano(), info.Mode(), content)
			return nil
		})
		if err != nil {
			t.Fatalf("WalkDir(%s) returned error: %v", dir, err)
		}
	}
	return snapshot
}

func assertSnapshotUnchanged(t *testing.T, before map[string]string, dirs ...string) {
	t.Helper()

	after := snapshotDirs(t, dirs...)
	for path, state := range after {
		if before[path] != state {
			t.Errorf("%s was written or created", path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			t.Errorf("%s was removed", path)
		}
	}
}

// builtInvoice copies the context fixture invoice to dir/name with
// invoice.status built, and writes the matching PDF next to it.
func builtInvoice(t *testing.T, invoicePath, dir, name string) string {
	t.Helper()

	source := strings.Replace(testfixture.ReadFile(t, invoicePath), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", path, err)
	}
	pdfPath := strings.TrimSuffix(path, ".yaml") + ".pdf"
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", pdfPath, err)
	}
	return path
}

func jsonString(s string) string {
	quoted, _ := json.Marshal(s)
	return string(quoted)
}

type dryRunCase struct {
	args       []string
	dirs       []string
	wantStdout string
	wantStderr string
}

func TestDryRunWritesNothing(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, x *clitest.Invox) dryRunCase
	}{
		{name: "new", setup: func(t *testing.T, x *clitest.Invox) dryRunCase {
			draft := testfixture.WriteDraft(t)
			workDir := t.TempDir()
			configPath := x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\n")
			x.Chdir(workDir)
			return dryRunCase{
				args:       []string{"new", "CUST-001", "-c", draft.Customers, "-u", draft.Issuer, "--defaults", draft.Defaults, "-n", "-e"},
				dirs:       []string{workDir, filepath.Dir(draft.Customers), filepath.Dir(configPath)},
				wantStdout: "CUST-001-002.yaml\n",
				wantStderr: "Would create CUST-001-002.yaml for CUST-001 (CUST-001-002)\n" +
					"warning: -e, --edit could not open the editor: stdin is not a terminal\n",
			}
		}},
		{name: "increment", setup: func(t *testing.T, x *clitest.Invox) dryRunCase {
			draft := testfixture.WriteDraft(t)
			workDir := t.TempDir()
			x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\n")
			invoicePath := testfixture.WriteNumberedInvoice(t, workDir, "invoice.yaml", "CUST-001-009", "draft")
			x.Chdir(workDir)
			return dryRunCase{
				args:       []string{"increment", "-i", "invoice.yaml", "-c", draft.Customers, "--dry-run"},
				dirs:       []string{workDir, filepath.Dir(invoicePath)},
				wantStdout: "invoice.yaml\n",
				wantStderr: "Would increment invoice.yaml for CUST-001: CUST-001-009 -> CUST-001-010\n",
			}
		}},
		{name: "render", setup: func(t *testing.T, x *clitest.Invox) dryRunCase {
			fx := testfixture.WriteContext(t)
			workDir := t.TempDir()
			x.Chdir(workDir)
			return dryRunCase{
				args:       []string{"render", "-i", fx.Invoice, "-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template, "-o", "out/x.tex", "-n"},
				dirs:       []string{workDir, filepath.Dir(fx.Invoice)},
				wantStdout: filepath.Join("out", "x.tex") + "\n",
				wantStderr: "Would render " + filepath.Join("out", "x.tex") + " for CUST-001 (CUST-001-001)\n",
			}
		}},
		{name: "build", setup: func(t *testing.T, x *clitest.Invox) dryRunCase {
			fx := testfixture.WriteContext(t)
			// The stub panics if tectonic runs, so a dry run that ran it fails.
			tempDir := isolateTempDir(t)
			workDir := t.TempDir()
			x.Chdir(workDir)
			return dryRunCase{
				args:       []string{"build", fx.Invoice, "-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template, "-n"},
				dirs:       []string{workDir, filepath.Dir(fx.Invoice), tempDir},
				wantStdout: strings.TrimSuffix(fx.Invoice, ".yaml") + ".pdf\n",
				wantStderr: "Would build " + strings.TrimSuffix(fx.Invoice, ".yaml") + ".pdf for CUST-001 (CUST-001-001)\n" +
					"Would set invoice.status to built in " + fx.Invoice + "\n",
			}
		}},
		{name: "build --archive", setup: func(t *testing.T, x *clitest.Invox) dryRunCase {
			fx := testfixture.WriteContext(t)
			archiveDir := t.TempDir()
			configPath := x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
			workDir := t.TempDir()
			x.Chdir(workDir)
			return dryRunCase{
				args:       []string{"build", fx.Invoice, "-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template, "--archive", "--dry-run"},
				dirs:       []string{workDir, filepath.Dir(fx.Invoice), archiveDir, filepath.Dir(configPath)},
				wantStdout: strings.TrimSuffix(fx.Invoice, ".yaml") + ".pdf\n",
				wantStderr: "Would build " + strings.TrimSuffix(fx.Invoice, ".yaml") + ".pdf for CUST-001 (CUST-001-001)\n" +
					"Would set invoice.status to built in " + fx.Invoice + "\n" +
					"Would archive " + fx.Invoice + " -> " + filepath.Join(archiveDir, "invoice.yaml") + "\n",
			}
		}},
		{name: "archive", setup: func(t *testing.T, x *clitest.Invox) dryRunCase {
			archiveDir := t.TempDir()
			x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
			workDir := t.TempDir()
			testfixture.WriteNumberedInvoice(t, workDir, "first.yaml", "CUST-001-001", "built")
			x.Chdir(workDir)
			archived := filepath.Join(archiveDir, "first.yaml")
			return dryRunCase{
				args:       []string{"archive", "add", "first.yaml", "-n"},
				dirs:       []string{workDir, archiveDir},
				wantStdout: archived + "\n",
				wantStderr: "Would archive first.yaml -> " + archived + "\n",
			}
		}},
		{name: "archive replacing an archived invoice without a terminal", setup: func(t *testing.T, x *clitest.Invox) dryRunCase {
			e := x.EditArchive()
			return dryRunCase{
				args:       []string{"archive", "add", "first.yaml", "--dry-run"},
				dirs:       []string{filepath.Dir(e.WorkingCopy), e.ArchiveDir},
				wantStdout: e.ArchivedPath + "\n",
				wantStderr: "Would replace archived invoice " + e.ArchivedPath + "; the previous version would be kept in " + filepath.Join(e.ArchiveDir, ".history") + "\n" +
					"Would archive first.yaml -> " + e.ArchivedPath + "\n",
			}
		}},
		{name: "archive --json replacing an archived invoice", setup: func(t *testing.T, x *clitest.Invox) dryRunCase {
			e := x.EditArchive()
			return dryRunCase{
				args:       []string{"archive", "add", "first.yaml", "--dry-run", "--json", "path,replaced"},
				dirs:       []string{filepath.Dir(e.WorkingCopy), e.ArchiveDir},
				wantStdout: "{\"path\":" + jsonString(e.ArchivedPath) + ",\"replaced\":[{\"path\":" + jsonString(e.ArchivedPath) + ",\"backupPath\":\"\"}]}\n",
				wantStderr: "Would replace archived invoice " + e.ArchivedPath + "; the previous version would be kept in " + filepath.Join(e.ArchiveDir, ".history") + "\n" +
					"Would archive first.yaml -> " + e.ArchivedPath + "\n",
			}
		}},
		{name: "archive edit", setup: func(t *testing.T, x *clitest.Invox) dryRunCase {
			archiveDir := t.TempDir()
			x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
			archivedPath := testfixture.WriteNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")
			workDir := t.TempDir()
			x.Chdir(workDir)
			return dryRunCase{
				args:       []string{"archive", "edit", "first.yaml", "-n"},
				dirs:       []string{workDir, archiveDir},
				wantStdout: "first.yaml\n",
				wantStderr: "Would copy " + archivedPath + " -> first.yaml\n",
			}
		}},
		{name: "email -o", setup: func(t *testing.T, x *clitest.Invox) dryRunCase {
			fx := testfixture.WriteContext(t)
			workDir := t.TempDir()
			builtInvoice(t, fx.Invoice, workDir, "inv.yaml")
			x.Chdir(workDir)
			return dryRunCase{
				args:       []string{"email", "inv.yaml", "-c", fx.Customers, "-u", fx.Issuer, "-o", "draft.eml", "--dry-run"},
				dirs:       []string{workDir},
				wantStdout: "draft.eml\n",
				wantStderr: "Would open email draft for CUST-001 (CUST-001-001) to office@appsters.example\n" +
					"Subject: Invoice CUST-001-001\n" +
					"Attachment: inv.pdf\n" +
					"Would write the draft to draft.eml\n",
			}
		}},
		{name: "email to a temporary draft", setup: func(t *testing.T, x *clitest.Invox) dryRunCase {
			fx := testfixture.WriteContext(t)
			tempDir := isolateTempDir(t)
			// A draft old enough for a real run to remove.
			stale := filepath.Join(tempDir, "invox-email-stale")
			if err := os.Mkdir(stale, 0o755); err != nil {
				t.Fatalf("Mkdir returned error: %v", err)
			}
			if err := os.Chtimes(stale, factorytest.Now.Add(-48*time.Hour), factorytest.Now.Add(-48*time.Hour)); err != nil {
				t.Fatalf("Chtimes returned error: %v", err)
			}
			workDir := t.TempDir()
			builtInvoice(t, fx.Invoice, workDir, "inv.yaml")
			x.Chdir(workDir)
			return dryRunCase{
				args:       []string{"email", "inv.yaml", "-c", fx.Customers, "-u", fx.Issuer, "-n", "--to", "billing@example.com"},
				dirs:       []string{workDir, tempDir},
				wantStdout: "",
				wantStderr: "Would open email draft for CUST-001 (CUST-001-001) to billing@example.com\n" +
					"Subject: Invoice CUST-001-001\n" +
					"Attachment: inv.pdf\n",
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := clitest.New(t)

			c := tc.setup(t, x)
			before := snapshotDirs(t, c.dirs...)

			// The stub fails the test if the dry run opens the editor or a
			// draft.
			exitCode, stdout, stderr := x.Run(c.args)
			if exitCode != 0 {
				t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
			}
			if stdout != c.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout, c.wantStdout)
			}
			if stderr != c.wantStderr {
				t.Errorf("stderr = %q, want %q", stderr, c.wantStderr)
			}
			assertSnapshotUnchanged(t, before, c.dirs...)
		})
	}
}

// TestDryRunFailsWhereTheRunFails runs each command with and without
// --dry-run on input the real run rejects: both fail the same way, and
// neither writes anything.
func TestDryRunFailsWhereTheRunFails(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, x *clitest.Invox) ([]string, []string)
		// realRunWrites is set where the real run writes before it fails,
		// as build does before its archive step.
		realRunWrites bool
		wantExit      int
		wantStderr    string
	}{
		{name: "new over an existing output", wantExit: 1, setup: func(t *testing.T, x *clitest.Invox) ([]string, []string) {
			draft := testfixture.WriteDraft(t)
			workDir := t.TempDir()
			x.WriteConfig("")
			if err := os.WriteFile(filepath.Join(workDir, "mine.yaml"), []byte("keep\n"), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			x.Chdir(workDir)
			return []string{"new", "CUST-001", "-c", draft.Customers, "-u", draft.Issuer, "--defaults", draft.Defaults, "-o", "mine.yaml"}, []string{workDir}
		}, wantStderr: "error: mine.yaml already exists; pass --force to replace it or choose a different -o/--output path\n"},
		{name: "new --force over an archived invoice", wantExit: 1, setup: func(t *testing.T, x *clitest.Invox) ([]string, []string) {
			draft := testfixture.WriteDraft(t)
			archiveDir := t.TempDir()
			x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
			archivedPath := testfixture.WriteNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")
			workDir := t.TempDir()
			x.Chdir(workDir)
			return []string{"new", "CUST-001", "-c", draft.Customers, "-u", draft.Issuer, "--defaults", draft.Defaults, "-o", archivedPath, "--force"}, []string{workDir, archiveDir}
		}, wantStderr: "first.yaml is in the archive directory and is never overwritten"},
		{name: "archive edit --force inside the archive", wantExit: 1, setup: func(t *testing.T, x *clitest.Invox) ([]string, []string) {
			archiveDir := t.TempDir()
			x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
			testfixture.WriteNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")
			x.Chdir(archiveDir)
			return []string{"archive", "edit", "first.yaml", "--force"}, []string{archiveDir}
		}, wantStderr: "first.yaml is in the archive directory and is never overwritten"},
		{name: "render with a broken template", wantExit: 1, setup: func(t *testing.T, x *clitest.Invox) ([]string, []string) {
			fx := testfixture.WriteContext(t)
			if err := os.WriteFile(fx.Template, []byte("@@ISSUER_CITY_AND_POSTAL_CODE@@\n"), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			workDir := t.TempDir()
			x.Chdir(workDir)
			return []string{"render", "-i", fx.Invoice, "-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template}, []string{workDir, filepath.Dir(fx.Invoice)}
		}, wantStderr: "@@ISSUER_CITY_AND_POSTAL_CODE@@: unknown placeholder"},
		{name: "build with a broken template", wantExit: 1, setup: func(t *testing.T, x *clitest.Invox) ([]string, []string) {
			fx := testfixture.WriteContext(t)
			if err := os.WriteFile(fx.Template, []byte("@@ISSUER_CITY_AND_POSTAL_CODE@@\n"), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			workDir := t.TempDir()
			x.Chdir(workDir)
			return []string{"build", fx.Invoice, "-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template}, []string{workDir, filepath.Dir(fx.Invoice)}
		}, wantStderr: "@@ISSUER_CITY_AND_POSTAL_CODE@@: unknown placeholder"},
		{name: "build --archive with a duplicate number", wantExit: 1, realRunWrites: true, setup: func(t *testing.T, x *clitest.Invox) ([]string, []string) {
			fx := testfixture.WriteContext(t)
			archiveDir := t.TempDir()
			x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
			testfixture.WriteNumberedInvoice(t, archiveDir, "old.yaml", "CUST-001-001", "archived")
			workDir := t.TempDir()
			x.Chdir(workDir)
			return []string{"build", fx.Invoice, "-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template, "--archive"}, []string{workDir, filepath.Dir(fx.Invoice), archiveDir}
		}, wantStderr: "is already used by archived invoice"},
		{name: "build --archive of an archived invoice", wantExit: 1, realRunWrites: true, setup: func(t *testing.T, x *clitest.Invox) ([]string, []string) {
			fx := testfixture.WriteContext(t)
			source := strings.Replace(testfixture.ReadFile(t, fx.Invoice), "  paid_amount: 0", "  paid_amount: 0\n  status: archived", 1)
			if err := os.WriteFile(fx.Invoice, []byte(source), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			archiveDir := t.TempDir()
			x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
			workDir := t.TempDir()
			x.Chdir(workDir)
			return []string{"build", fx.Invoice, "-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template, "--archive"}, []string{workDir, filepath.Dir(fx.Invoice), archiveDir}
		}, wantStderr: "invoice.status must be `built` before archiving, got `archived`"},
		{name: "archive a duplicate number", wantExit: 1, setup: func(t *testing.T, x *clitest.Invox) ([]string, []string) {
			archiveDir := t.TempDir()
			x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
			testfixture.WriteNumberedInvoice(t, archiveDir, "old.yaml", "CUST-001-001", "archived")
			workDir := t.TempDir()
			testfixture.WriteNumberedInvoice(t, workDir, "first.yaml", "CUST-001-001", "built")
			x.Chdir(workDir)
			return []string{"archive", "add", "first.yaml"}, []string{workDir, archiveDir}
		}, wantStderr: "is already used by archived invoice"},
		{name: "archive edit over a working copy", wantExit: 1, setup: func(t *testing.T, x *clitest.Invox) ([]string, []string) {
			archiveDir := t.TempDir()
			x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
			testfixture.WriteNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")
			workDir := t.TempDir()
			testfixture.WriteNumberedInvoice(t, workDir, "first.yaml", "CUST-001-001", "editing")
			x.Chdir(workDir)
			return []string{"archive", "edit", "first.yaml"}, []string{workDir, archiveDir}
		}, wantStderr: "error: first.yaml already exists; pass --force to replace it or choose a different working directory\n"},
		{name: "email over an existing -o", wantExit: 1, setup: func(t *testing.T, x *clitest.Invox) ([]string, []string) {
			fx := testfixture.WriteContext(t)
			workDir := t.TempDir()
			builtInvoice(t, fx.Invoice, workDir, "inv.yaml")
			if err := os.WriteFile(filepath.Join(workDir, "draft.eml"), []byte("keep\n"), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			x.Chdir(workDir)
			return []string{"email", "inv.yaml", "-c", fx.Customers, "-u", fx.Issuer, "-o", "draft.eml"}, []string{workDir}
		}, wantStderr: "error: draft.eml already exists; pass --force to replace it or choose a different -o/--output path\n"},
	} {
		for _, dryRun := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s dry-run=%t", tc.name, dryRun), func(t *testing.T) {
				x := clitest.New(t)

				args, dirs := tc.setup(t, x)
				if dryRun {
					args = append(args, "--dry-run")
				}
				before := snapshotDirs(t, dirs...)

				if tc.realRunWrites && !dryRun {
					// The real run runs tectonic; elsewhere the stub panics if
					// it runs.
					x.ExpectTectonic()
				}
				exitCode, stdout, stderr := x.Run(args)
				if exitCode != tc.wantExit {
					t.Fatalf("exitCode = %d, want %d, stderr=%q", exitCode, tc.wantExit, stderr)
				}
				if stdout != "" {
					t.Errorf("stdout = %q, want empty", stdout)
				}
				want := tc.wantStderr
				if strings.HasPrefix(want, "error: ") {
					if stderr != want {
						t.Errorf("stderr = %q, want %q", stderr, want)
					}
				} else if !strings.Contains(stderr, want) {
					t.Errorf("stderr = %q, want it to contain %q", stderr, want)
				}
				if dryRun || !tc.realRunWrites {
					assertSnapshotUnchanged(t, before, dirs...)
				}
			})
		}
	}
}

// TestForceNeverReplacesArchivedFile points new -o --force at an archived
// invoice by paths that differ from archive.dir in their name but not in the
// file they reach.
func TestForceNeverReplacesArchivedFile(t *testing.T) {
	for _, tc := range []struct {
		name string
		// setup returns archive.dir and the -o path for the archived
		// invoice archiveDir/first.yaml.
		setup func(t *testing.T, root string) (archiveDir, output string)
	}{
		{name: "differently cased path", setup: func(t *testing.T, root string) (string, string) {
			archiveDir := filepath.Join(root, "archive")
			if err := os.Mkdir(archiveDir, 0o755); err != nil {
				t.Fatalf("Mkdir returned error: %v", err)
			}
			upper := filepath.Join(root, "ARCHIVE")
			if _, err := os.Stat(upper); err != nil {
				t.Skip("the file system is case-sensitive")
			}
			return archiveDir, filepath.Join(upper, "first.yaml")
		}},
		{name: "symlink to the archived invoice", setup: func(t *testing.T, root string) (string, string) {
			archiveDir := filepath.Join(root, "archive")
			if err := os.Mkdir(archiveDir, 0o755); err != nil {
				t.Fatalf("Mkdir returned error: %v", err)
			}
			link := filepath.Join(root, "link.yaml")
			if err := os.Symlink(filepath.Join(archiveDir, "first.yaml"), link); err != nil {
				t.Skipf("cannot create symlinks: %v", err)
			}
			return archiveDir, link
		}},
		{name: "archive.dir is a symlink", setup: func(t *testing.T, root string) (string, string) {
			realDir := filepath.Join(root, "real")
			if err := os.Mkdir(realDir, 0o755); err != nil {
				t.Fatalf("Mkdir returned error: %v", err)
			}
			archiveDir := filepath.Join(root, "archive")
			if err := os.Symlink(realDir, archiveDir); err != nil {
				t.Skipf("cannot create symlinks: %v", err)
			}
			return archiveDir, filepath.Join(realDir, "first.yaml")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := clitest.New(t)

			draft := testfixture.WriteDraft(t)
			archiveDir, output := tc.setup(t, t.TempDir())
			x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
			archivedPath := testfixture.WriteNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")
			before := snapshotDirs(t, archiveDir+string(filepath.Separator))
			workDir := t.TempDir()
			x.Chdir(workDir)

			for _, args := range [][]string{{"--force"}, {"--force", "--dry-run"}} {
				exitCode, stdout, stderr := x.Run(append([]string{"new", "CUST-001", "-c", draft.Customers, "-u", draft.Issuer, "--defaults", draft.Defaults, "-o", output}, args...))
				if exitCode != 1 {
					t.Fatalf("%v: exitCode = %d, want 1, stderr=%q", args, exitCode, stderr)
				}
				if stdout != "" {
					t.Errorf("%v: stdout = %q, want empty", args, stdout)
				}
				if !strings.Contains(stderr, "is in the archive directory and is never overwritten") {
					t.Errorf("%v: stderr = %q, want the archive refusal", args, stderr)
				}
			}
			assertSnapshotUnchanged(t, before, archiveDir+string(filepath.Separator))
			if got := testfixture.ReadFile(t, archivedPath); !strings.Contains(got, "status: archived") {
				t.Fatalf("archived invoice changed:\n%s", got)
			}
		})
	}
}
