package cli

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

	source := strings.Replace(readFileForTest(t, invoicePath), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
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
		setup func(t *testing.T) dryRunCase
	}{
		{name: "new", setup: func(t *testing.T) dryRunCase {
			customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
			workDir := t.TempDir()
			configPath := writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\n")
			chdirForTest(t, workDir)
			return dryRunCase{
				args:       []string{"new", "CUST-001", "-c", customersPath, "-u", issuerPath, "--defaults", defaultsPath, "-n", "-e"},
				dirs:       []string{workDir, filepath.Dir(customersPath), filepath.Dir(configPath)},
				wantStdout: "CUST-001-002.yaml\n",
				wantStderr: "Would create CUST-001-002.yaml for CUST-001 (CUST-001-002)\n" +
					"warning: -e, --edit could not open the editor: stdin is not a terminal\n",
			}
		}},
		{name: "increment", setup: func(t *testing.T) dryRunCase {
			customersPath, _, _ := writeDraftFixtures(t)
			workDir := t.TempDir()
			writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\n")
			invoicePath := writeNumberedInvoice(t, workDir, "invoice.yaml", "CUST-001-009", "draft")
			chdirForTest(t, workDir)
			return dryRunCase{
				args:       []string{"increment", "-i", "invoice.yaml", "-c", customersPath, "--dry-run"},
				dirs:       []string{workDir, filepath.Dir(invoicePath)},
				wantStdout: "invoice.yaml\n",
				wantStderr: "Would increment invoice.yaml for CUST-001: CUST-001-009 -> CUST-001-010\n",
			}
		}},
		{name: "render", setup: func(t *testing.T) dryRunCase {
			customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
			workDir := t.TempDir()
			chdirForTest(t, workDir)
			return dryRunCase{
				args:       []string{"render", "-i", invoicePath, "-c", customersPath, "-u", issuerPath, "-t", templatePath, "-o", "out/x.tex", "-n"},
				dirs:       []string{workDir, filepath.Dir(invoicePath)},
				wantStdout: filepath.Join("out", "x.tex") + "\n",
				wantStderr: "Would render " + filepath.Join("out", "x.tex") + " for CUST-001 (CUST-001-001)\n",
			}
		}},
		{name: "build", setup: func(t *testing.T) dryRunCase {
			customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
			// Tectonic fails if it runs, so a dry run that ran it would fail.
			installFakeTectonic(t, fakeTectonicFail)
			tempDir := isolateTempDir(t)
			workDir := t.TempDir()
			chdirForTest(t, workDir)
			return dryRunCase{
				args:       []string{"build", invoicePath, "-c", customersPath, "-u", issuerPath, "-t", templatePath, "-n"},
				dirs:       []string{workDir, filepath.Dir(invoicePath), tempDir},
				wantStdout: strings.TrimSuffix(invoicePath, ".yaml") + ".pdf\n",
				wantStderr: "Would build " + strings.TrimSuffix(invoicePath, ".yaml") + ".pdf for CUST-001 (CUST-001-001)\n" +
					"Would set invoice.status to built in " + invoicePath + "\n",
			}
		}},
		{name: "build --archive", setup: func(t *testing.T) dryRunCase {
			customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
			installFakeTectonic(t, fakeTectonicFail)
			archiveDir := t.TempDir()
			configPath := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
			workDir := t.TempDir()
			chdirForTest(t, workDir)
			return dryRunCase{
				args:       []string{"build", invoicePath, "-c", customersPath, "-u", issuerPath, "-t", templatePath, "--archive", "--dry-run"},
				dirs:       []string{workDir, filepath.Dir(invoicePath), archiveDir, filepath.Dir(configPath)},
				wantStdout: strings.TrimSuffix(invoicePath, ".yaml") + ".pdf\n",
				wantStderr: "Would build " + strings.TrimSuffix(invoicePath, ".yaml") + ".pdf for CUST-001 (CUST-001-001)\n" +
					"Would set invoice.status to built in " + invoicePath + "\n" +
					"Would archive " + invoicePath + " -> " + filepath.Join(archiveDir, "invoice.yaml") + "\n",
			}
		}},
		{name: "archive", setup: func(t *testing.T) dryRunCase {
			archiveDir := t.TempDir()
			writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
			workDir := t.TempDir()
			writeNumberedInvoice(t, workDir, "first.yaml", "CUST-001-001", "built")
			chdirForTest(t, workDir)
			archived := filepath.Join(archiveDir, "first.yaml")
			return dryRunCase{
				args:       []string{"archive", "add", "first.yaml", "-n"},
				dirs:       []string{workDir, archiveDir},
				wantStdout: archived + "\n",
				wantStderr: "Would archive first.yaml -> " + archived + "\n",
			}
		}},
		{name: "archive replacing an archived invoice without a terminal", setup: func(t *testing.T) dryRunCase {
			e := setupEditedArchive(t)
			return dryRunCase{
				args:       []string{"archive", "add", "first.yaml", "--dry-run"},
				dirs:       []string{filepath.Dir(e.workingCopy), e.archiveDir},
				wantStdout: e.archivedPath + "\n",
				wantStderr: "Would replace archived invoice " + e.archivedPath + "; the previous version would be kept in " + filepath.Join(e.archiveDir, ".history") + "\n" +
					"Would archive first.yaml -> " + e.archivedPath + "\n",
			}
		}},
		{name: "archive --json replacing an archived invoice", setup: func(t *testing.T) dryRunCase {
			e := setupEditedArchive(t)
			return dryRunCase{
				args:       []string{"archive", "add", "first.yaml", "--dry-run", "--json", "path,replaced"},
				dirs:       []string{filepath.Dir(e.workingCopy), e.archiveDir},
				wantStdout: "{\"path\":" + jsonString(e.archivedPath) + ",\"replaced\":[{\"path\":" + jsonString(e.archivedPath) + ",\"backupPath\":\"\"}]}\n",
				wantStderr: "Would replace archived invoice " + e.archivedPath + "; the previous version would be kept in " + filepath.Join(e.archiveDir, ".history") + "\n" +
					"Would archive first.yaml -> " + e.archivedPath + "\n",
			}
		}},
		{name: "archive edit", setup: func(t *testing.T) dryRunCase {
			archiveDir := t.TempDir()
			writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
			archivedPath := writeNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")
			workDir := t.TempDir()
			chdirForTest(t, workDir)
			return dryRunCase{
				args:       []string{"archive", "edit", "first.yaml", "-n"},
				dirs:       []string{workDir, archiveDir},
				wantStdout: "first.yaml\n",
				wantStderr: "Would copy " + archivedPath + " -> first.yaml\n",
			}
		}},
		{name: "email -o", setup: func(t *testing.T) dryRunCase {
			customersPath, issuerPath, invoicePath, _ := writeContextFixtures(t)
			workDir := t.TempDir()
			builtInvoice(t, invoicePath, workDir, "inv.yaml")
			chdirForTest(t, workDir)
			return dryRunCase{
				args:       []string{"email", "inv.yaml", "-c", customersPath, "-u", issuerPath, "-o", "draft.eml", "--dry-run"},
				dirs:       []string{workDir},
				wantStdout: "draft.eml\n",
				wantStderr: "Would open email draft for CUST-001 (CUST-001-001) to office@appsters.example\n" +
					"Subject: Invoice CUST-001-001\n" +
					"Attachment: inv.pdf\n" +
					"Would write the draft to draft.eml\n",
			}
		}},
		{name: "email to a temporary draft", setup: func(t *testing.T) dryRunCase {
			customersPath, issuerPath, invoicePath, _ := writeContextFixtures(t)
			tempDir := isolateTempDir(t)
			// A draft old enough for a real run to remove.
			stale := filepath.Join(tempDir, "invox-email-stale")
			if err := os.Mkdir(stale, 0o755); err != nil {
				t.Fatalf("Mkdir returned error: %v", err)
			}
			if err := os.Chtimes(stale, time.Now().Add(-48*time.Hour), time.Now().Add(-48*time.Hour)); err != nil {
				t.Fatalf("Chtimes returned error: %v", err)
			}
			workDir := t.TempDir()
			builtInvoice(t, invoicePath, workDir, "inv.yaml")
			chdirForTest(t, workDir)
			return dryRunCase{
				args:       []string{"email", "inv.yaml", "-c", customersPath, "-u", issuerPath, "-n", "--to", "billing@example.com"},
				dirs:       []string{workDir, tempDir},
				wantStdout: "",
				wantStderr: "Would open email draft for CUST-001 (CUST-001-001) to billing@example.com\n" +
					"Subject: Invoice CUST-001-001\n" +
					"Attachment: inv.pdf\n",
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.setup(t)
			before := snapshotDirs(t, c.dirs...)

			// The stub fails the test if the dry run opens the editor or a
			// draft.
			f, _ := testFactory(t)
			exitCode, stdout, stderr := captureRunFactory(t, f, c.args)
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
		setup func(t *testing.T) ([]string, []string)
		// realRunWrites is set where the real run writes before it fails,
		// as build does before its archive step.
		realRunWrites bool
		wantExit      int
		wantStderr    string
	}{
		{name: "new over an existing output", wantExit: 1, setup: func(t *testing.T) ([]string, []string) {
			customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
			workDir := t.TempDir()
			writeConfigFile(t, "")
			if err := os.WriteFile(filepath.Join(workDir, "mine.yaml"), []byte("keep\n"), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			chdirForTest(t, workDir)
			return []string{"new", "CUST-001", "-c", customersPath, "-u", issuerPath, "--defaults", defaultsPath, "-o", "mine.yaml"}, []string{workDir}
		}, wantStderr: "error: mine.yaml already exists; pass --force to replace it or choose a different -o/--output path\n"},
		{name: "new --force over an archived invoice", wantExit: 1, setup: func(t *testing.T) ([]string, []string) {
			customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
			archiveDir := t.TempDir()
			writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
			archivedPath := writeNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")
			workDir := t.TempDir()
			chdirForTest(t, workDir)
			return []string{"new", "CUST-001", "-c", customersPath, "-u", issuerPath, "--defaults", defaultsPath, "-o", archivedPath, "--force"}, []string{workDir, archiveDir}
		}, wantStderr: "first.yaml is in the archive directory and is never overwritten"},
		{name: "archive edit --force inside the archive", wantExit: 1, setup: func(t *testing.T) ([]string, []string) {
			archiveDir := t.TempDir()
			writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
			writeNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")
			chdirForTest(t, archiveDir)
			return []string{"archive", "edit", "first.yaml", "--force"}, []string{archiveDir}
		}, wantStderr: "first.yaml is in the archive directory and is never overwritten"},
		{name: "render with a broken template", wantExit: 1, setup: func(t *testing.T) ([]string, []string) {
			customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
			if err := os.WriteFile(templatePath, []byte("@@ISSUER_CITY_AND_POSTAL_CODE@@\n"), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			workDir := t.TempDir()
			chdirForTest(t, workDir)
			return []string{"render", "-i", invoicePath, "-c", customersPath, "-u", issuerPath, "-t", templatePath}, []string{workDir, filepath.Dir(invoicePath)}
		}, wantStderr: "@@ISSUER_CITY_AND_POSTAL_CODE@@: unknown placeholder"},
		{name: "build with a broken template", wantExit: 1, setup: func(t *testing.T) ([]string, []string) {
			customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
			if err := os.WriteFile(templatePath, []byte("@@ISSUER_CITY_AND_POSTAL_CODE@@\n"), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			installFakeTectonic(t, fakeTectonicWritePDF)
			workDir := t.TempDir()
			chdirForTest(t, workDir)
			return []string{"build", invoicePath, "-c", customersPath, "-u", issuerPath, "-t", templatePath}, []string{workDir, filepath.Dir(invoicePath)}
		}, wantStderr: "@@ISSUER_CITY_AND_POSTAL_CODE@@: unknown placeholder"},
		{name: "build --archive with a duplicate number", wantExit: 1, realRunWrites: true, setup: func(t *testing.T) ([]string, []string) {
			customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
			installFakeTectonic(t, fakeTectonicWritePDF)
			archiveDir := t.TempDir()
			writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
			writeNumberedInvoice(t, archiveDir, "old.yaml", "CUST-001-001", "archived")
			workDir := t.TempDir()
			chdirForTest(t, workDir)
			return []string{"build", invoicePath, "-c", customersPath, "-u", issuerPath, "-t", templatePath, "--archive"}, []string{workDir, filepath.Dir(invoicePath), archiveDir}
		}, wantStderr: "is already used by archived invoice"},
		{name: "build --archive of an archived invoice", wantExit: 1, realRunWrites: true, setup: func(t *testing.T) ([]string, []string) {
			customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
			source := strings.Replace(readFileForTest(t, invoicePath), "  paid_amount: 0", "  paid_amount: 0\n  status: archived", 1)
			if err := os.WriteFile(invoicePath, []byte(source), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			installFakeTectonic(t, fakeTectonicWritePDF)
			archiveDir := t.TempDir()
			writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
			workDir := t.TempDir()
			chdirForTest(t, workDir)
			return []string{"build", invoicePath, "-c", customersPath, "-u", issuerPath, "-t", templatePath, "--archive"}, []string{workDir, filepath.Dir(invoicePath), archiveDir}
		}, wantStderr: "invoice.status must be `built` before archiving, got `archived`"},
		{name: "archive a duplicate number", wantExit: 1, setup: func(t *testing.T) ([]string, []string) {
			archiveDir := t.TempDir()
			writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
			writeNumberedInvoice(t, archiveDir, "old.yaml", "CUST-001-001", "archived")
			workDir := t.TempDir()
			writeNumberedInvoice(t, workDir, "first.yaml", "CUST-001-001", "built")
			chdirForTest(t, workDir)
			return []string{"archive", "add", "first.yaml"}, []string{workDir, archiveDir}
		}, wantStderr: "is already used by archived invoice"},
		{name: "archive edit over a working copy", wantExit: 1, setup: func(t *testing.T) ([]string, []string) {
			archiveDir := t.TempDir()
			writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
			writeNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")
			workDir := t.TempDir()
			writeNumberedInvoice(t, workDir, "first.yaml", "CUST-001-001", "editing")
			chdirForTest(t, workDir)
			return []string{"archive", "edit", "first.yaml"}, []string{workDir, archiveDir}
		}, wantStderr: "error: first.yaml already exists; pass --force to replace it or choose a different working directory\n"},
		{name: "email over an existing -o", wantExit: 1, setup: func(t *testing.T) ([]string, []string) {
			customersPath, issuerPath, invoicePath, _ := writeContextFixtures(t)
			workDir := t.TempDir()
			builtInvoice(t, invoicePath, workDir, "inv.yaml")
			if err := os.WriteFile(filepath.Join(workDir, "draft.eml"), []byte("keep\n"), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			chdirForTest(t, workDir)
			return []string{"email", "inv.yaml", "-c", customersPath, "-u", issuerPath, "-o", "draft.eml"}, []string{workDir}
		}, wantStderr: "error: draft.eml already exists; pass --force to replace it or choose a different -o/--output path\n"},
	} {
		for _, dryRun := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s dry-run=%t", tc.name, dryRun), func(t *testing.T) {
				args, dirs := tc.setup(t)
				if dryRun {
					args = append(args, "--dry-run")
				}
				before := snapshotDirs(t, dirs...)

				var exitCode int
				var stdout, stderr string
				if tc.realRunWrites {
					// The real run runs the fake tectonic.
					exitCode, stdout, stderr = captureRun(t, args)
				} else {
					f, _ := testFactory(t)
					exitCode, stdout, stderr = captureRunFactory(t, f, args)
				}
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
			customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
			archiveDir, output := tc.setup(t, t.TempDir())
			writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
			archivedPath := writeNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")
			before := snapshotDirs(t, archiveDir+string(filepath.Separator))
			workDir := t.TempDir()
			chdirForTest(t, workDir)

			for _, args := range [][]string{{"--force"}, {"--force", "--dry-run"}} {
				exitCode, stdout, stderr := captureRun(t, append([]string{"new", "CUST-001", "-c", customersPath, "-u", issuerPath, "--defaults", defaultsPath, "-o", output}, args...))
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
			if got := readFileForTest(t, archivedPath); !strings.Contains(got, "status: archived") {
				t.Fatalf("archived invoice changed:\n%s", got)
			}
		})
	}
}
