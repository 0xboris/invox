package archive

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/invoice"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func walkPaths(t *testing.T, s Store) []string {
	t.Helper()
	paths, _ := walk(t, s)
	return paths
}

// walk returns the files Walk visits and the Markdown files it reports.
func walk(t *testing.T, s Store) (visited, markdown []string) {
	t.Helper()
	markdown, err := s.Walk(func(path string) error {
		visited = append(visited, path)
		return nil
	})
	if err != nil {
		t.Fatalf("Walk returned error: %v", err)
	}
	return visited, markdown
}

func TestWalkVisitsInvoiceFilesBelowTheRootInOrder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, name := range []string{"b.yaml", "a/z.YML", "notes.txt", "README.md", ".history/old.yaml", "a/.history/kept.yaml"} {
		writeFile(t, filepath.Join(dir, filepath.FromSlash(name)), "")
	}
	// Markdown with front matter is what invox used to read as an invoice.
	writeFile(t, filepath.Join(dir, "a", "x.md"), "---\ninvoice: {}\n---\n")
	writeFile(t, filepath.Join(dir, "a", "y.markdown"), "---\r\ninvoice: {}\r\n---\r\n")
	writeFile(t, filepath.Join(dir, ".history", "old.md"), "---\ninvoice: {}\n---\n")

	visited, markdown := walk(t, Store{Dir: dir})
	want := []string{
		filepath.Join(dir, "a", ".history", "kept.yaml"),
		filepath.Join(dir, "a", "z.YML"),
		filepath.Join(dir, "b.yaml"),
	}
	if !reflect.DeepEqual(visited, want) {
		t.Fatalf("Walk visited %q, want %q", visited, want)
	}
	wantMarkdown := []string{filepath.Join(dir, "a", "x.md"), filepath.Join(dir, "a", "y.markdown")}
	if !reflect.DeepEqual(markdown, wantMarkdown) {
		t.Fatalf("Walk reported Markdown %q, want %q", markdown, wantMarkdown)
	}
}

// Numbering reports skipped files in walk order, `archive list` in
// Filename order; a nested file and a sibling with a dash tell them apart.
func TestWalkOrderDiffersFromListOrder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a", "x.yaml"), "X")
	writeFile(t, filepath.Join(dir, "a-b.yaml"), "B")

	if got, want := walkPaths(t, Store{Dir: dir}), []string{
		filepath.Join(dir, "a", "x.yaml"),
		filepath.Join(dir, "a-b.yaml"),
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Walk visited %q, want %q", got, want)
	}

	entries, _, err := Store{Dir: dir}.List(func(string) (billing.ArchiveEntry, bool, error) { return billing.ArchiveEntry{}, true, nil })
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	var got []string
	for _, entry := range entries {
		got = append(got, entry.Filename)
	}
	if want := []string{"a-b.yaml", filepath.Join("a", "x.yaml")}; !reflect.DeepEqual(got, want) {
		t.Fatalf("List order = %q, want %q", got, want)
	}
}

func TestWalkReportsPathsBelowTheSymlinkedRoot(t *testing.T) {
	t.Parallel()

	real := t.TempDir()
	writeFile(t, filepath.Join(real, "customer-a", "first.yaml"), "")
	link := filepath.Join(t.TempDir(), "archive-link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	got := walkPaths(t, Store{Dir: link})
	if want := []string{filepath.Join(link, "customer-a", "first.yaml")}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Walk visited %q, want %q", got, want)
	}
}

func TestWalkWithoutArchive(t *testing.T) {
	t.Parallel()

	for name, dir := range map[string]string{
		"unset":   "",
		"missing": filepath.Join(t.TempDir(), "none"),
	} {
		if got := walkPaths(t, Store{Dir: dir}); len(got) != 0 {
			t.Fatalf("%s: Walk visited %q, want nothing", name, got)
		}
	}

	file := filepath.Join(t.TempDir(), "archive")
	writeFile(t, file, "")
	_, err := Store{Dir: file}.Walk(func(string) error { return nil })
	if want := file + ": archive.dir must point to a directory"; err == nil || err.Error() != want {
		t.Fatalf("Walk(file) error = %v, want %q", err, want)
	}
}

func TestListSortsByFilenameAndSkipsWhatTheReaderRejects(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "b.yaml"), "B")
	writeFile(t, filepath.Join(dir, "a", "c.yaml"), "C")
	writeFile(t, filepath.Join(dir, "a-d.yaml"), "")
	writeFile(t, filepath.Join(dir, "old.md"), "---\ncustomer_id: C\n---\n")

	entries, unread, err := Store{Dir: dir}.List(func(path string) (billing.ArchiveEntry, bool, error) {
		data, err := os.ReadFile(path)
		if err != nil {
			return billing.ArchiveEntry{}, false, err
		}
		if len(data) == 0 {
			return billing.ArchiveEntry{}, false, nil
		}
		return billing.ArchiveEntry{Number: string(data)}, true, nil
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	want := []billing.ArchiveEntry{
		{Path: filepath.Join(dir, "a", "c.yaml"), Filename: filepath.Join("a", "c.yaml"), Number: "C"},
		{Path: filepath.Join(dir, "b.yaml"), Filename: "b.yaml", Number: "B"},
	}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("List = %+v, want %+v", entries, want)
	}
	if want := (billing.Unread{Dir: dir, Markdown: []string{filepath.Join(dir, "old.md")}}); !reflect.DeepEqual(unread, want) {
		t.Fatalf("List unread = %+v, want %+v", unread, want)
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "archive")
	s := Store{Dir: dir + string(filepath.Separator)}
	tests := []struct {
		name    string
		want    Target
		wantErr string
	}{
		{name: " customer-a/first.yaml ", want: Target{Path: filepath.Join(dir, "customer-a", "first.yaml"), Rel: filepath.Join("customer-a", "first.yaml")}},
		{name: "./first.yaml", want: Target{Path: filepath.Join(dir, "first.yaml"), Rel: "first.yaml"}},
		{name: "", wantErr: "archive filename must not be empty"},
		{name: ".", wantErr: "archive filename must not be empty"},
		{name: filepath.Join(dir, "first.yaml"), wantErr: "archive filename must be relative to archive.dir, got " + filepath.Join(dir, "first.yaml")},
		{name: "../first.yaml", wantErr: filepath.Join("..", "first.yaml") + " must stay within " + dir},
		{name: ".history/first.yaml", wantErr: filepath.Join(".history", "first.yaml") + " is a backup in .history, not an archived invoice"},
		{name: ".HISTORY/first.yaml", wantErr: filepath.Join(".HISTORY", "first.yaml") + " is a backup in .history, not an archived invoice"},
	}
	for _, tt := range tests {
		got, err := s.Resolve(tt.name)
		if tt.wantErr != "" {
			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("Resolve(%q) error = %v, want %q", tt.name, err, tt.wantErr)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("Resolve(%q) = %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
	}
}

func TestFindRequiresAFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "first.yaml"), "")
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := Store{Dir: dir}

	got, err := s.Find("first.yaml")
	if want := (Target{Path: filepath.Join(dir, "first.yaml"), Rel: "first.yaml"}); err != nil || got != want {
		t.Fatalf("Find(first.yaml) = %+v, %v; want %+v", got, err, want)
	}
	_, err = s.Find("second.yaml")
	if want := "second.yaml does not exist in " + dir; err == nil || err.Error() != want {
		t.Fatalf("Find(second.yaml) error = %v, want %q", err, want)
	}
	_, err = s.Find("sub")
	if want := filepath.Join(dir, "sub") + ": archived invoice must be a file"; err == nil || err.Error() != want {
		t.Fatalf("Find(sub) error = %v, want %q", err, want)
	}
	writeFile(t, filepath.Join(dir, "old.md"), "---\ncustomer_id: C\n---\n")
	_, err = s.Find("old.md")
	if want := filepath.Join(dir, "old.md") + " is a Markdown invoice, which invox no longer reads; convert it to .yaml"; err == nil || err.Error() != want {
		t.Fatalf("Find(old.md) error = %v, want %q", err, want)
	}
}

func TestBackupKeepsEveryVersion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "customer-a", "first.yaml")
	writeFile(t, path, "v1")
	now := time.Date(2026, 10, 5, 14, 30, 45, 0, time.FixedZone("CEST", 2*60*60))
	s := Store{Dir: dir}

	first, err := s.Backup([]string{path}, now)
	if err != nil {
		t.Fatalf("Backup returned error: %v", err)
	}
	want := []billing.Backup{{Path: path, BackupPath: filepath.Join(dir, ".history", "customer-a", "first.20261005T123045Z.yaml")}}
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("Backup = %+v, want %+v", first, want)
	}

	writeFile(t, path, "v2")
	second, err := s.Backup([]string{path}, now)
	if err != nil {
		t.Fatalf("second Backup returned error: %v", err)
	}
	if want := filepath.Join(dir, ".history", "customer-a", "first.20261005T123045Z-2.yaml"); len(second) != 1 || second[0].BackupPath != want {
		t.Fatalf("second Backup = %+v, want BackupPath %q", second, want)
	}
	for backupPath, content := range map[string]string{first[0].BackupPath: "v1", second[0].BackupPath: "v2"} {
		if data, err := os.ReadFile(backupPath); err != nil || string(data) != content {
			t.Fatalf("%s = %q, %v; want %q", backupPath, data, err, content)
		}
	}
	if s.HistoryDir() != filepath.Join(dir, ".history") {
		t.Fatalf("HistoryDir = %q", s.HistoryDir())
	}
}

func TestExistingFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "first.yaml")
	writeFile(t, file, "")

	got, err := ExistingFiles("", file, filepath.Join(dir, "missing.yaml"), file)
	if err != nil || !reflect.DeepEqual(got, []string{file}) {
		t.Fatalf("ExistingFiles = %q, %v; want [%q]", got, err, file)
	}
	_, err = ExistingFiles(dir)
	if want := dir + ": archived invoice must be a file"; err == nil || err.Error() != want {
		t.Fatalf("ExistingFiles(dir) error = %v, want %q", err, want)
	}
}

func TestIsInvoiceFile(t *testing.T) {
	t.Parallel()

	for path, want := range map[string]bool{"a.yaml": true, "a.YML": true, "a.md": false, "a.Markdown": false, "a.txt": false, "yaml": false} {
		if got := isInvoiceFile(path); got != want {
			t.Errorf("isInvoiceFile(%q) = %v, want %v", path, got, want)
		}
	}
}

// Re-archiving backs up the archived file, then writes the source over it
// once, with the change applied, and removes the source.
func TestAddWritesTheChangedInvoiceOnce(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(t.TempDir(), "a.yaml")
	writeFile(t, src, "working copy\n")
	writeFile(t, filepath.Join(dir, "a.yaml"), "archived\n")

	var rewritten []string
	a := Archive{
		Locate: func() (string, error) { return dir, nil },
		Rewrite: func(path string, change func(*invoice.Invoice) error) ([]byte, error) {
			rewritten = append(rewritten, path)
			var inv invoice.Invoice
			if err := change(&inv); err != nil {
				return nil, err
			}
			return []byte("rewritten " + string(inv.CustomerID) + "\n"), nil
		},
		Now: func() time.Time { return time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC) },
	}
	workingCopy := invoice.Invoice{Archive: &invoice.ArchiveLink{ArchivePath: "a.yaml"}}
	result, err := a.Add(src, workingCopy, billing.AddOptions{Replace: true, Change: func(inv *invoice.Invoice) error {
		inv.CustomerID = "C-1"
		return nil
	}})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	archived := filepath.Join(dir, "a.yaml")
	backup := filepath.Join(dir, ".history", "a.20260306T120000Z.yaml")
	want := billing.ArchiveResult{
		Path:       archived,
		Replaced:   []billing.Backup{{Path: archived, BackupPath: backup}},
		HistoryDir: filepath.Join(dir, ".history"),
	}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("Add = %+v, want %+v", result, want)
	}
	if !reflect.DeepEqual(rewritten, []string{src}) {
		t.Fatalf("Rewrite read %v, want only the source", rewritten)
	}
	for path, content := range map[string]string{archived: "rewritten C-1\n", backup: "archived\n"} {
		if got, err := os.ReadFile(path); err != nil || string(got) != content {
			t.Fatalf("%s = %q, %v; want %q", path, got, err, content)
		}
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source should be removed, Stat err = %v", err)
	}
}

// When the invoice cannot be rewritten, Add moves nothing: the source
// stays, and the archive directory is not even created.
func TestAddMovesNothingWhenTheRewriteFails(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "archive")
	src := filepath.Join(t.TempDir(), "a.yaml")
	writeFile(t, src, "working copy\n")

	a := Archive{
		Locate: func() (string, error) { return dir, nil },
		Rewrite: func(string, func(*invoice.Invoice) error) ([]byte, error) {
			return nil, errors.New("a.yaml: root value must be a mapping")
		},
	}
	_, err := a.Add(src, invoice.Invoice{}, billing.AddOptions{Change: func(*invoice.Invoice) error { return nil }})
	if err == nil || err.Error() != "a.yaml: root value must be a mapping" {
		t.Fatalf("Add error = %v, want the rewrite error", err)
	}
	if got, err := os.ReadFile(src); err != nil || string(got) != "working copy\n" {
		t.Fatalf("source = %q, %v; want it unchanged", got, err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("archive directory should not be created, Stat err = %v", err)
	}
}
