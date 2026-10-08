package archive

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/billing"
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
	var paths []string
	if err := s.Walk(func(path string) error {
		paths = append(paths, path)
		return nil
	}); err != nil {
		t.Fatalf("Walk returned error: %v", err)
	}
	return paths
}

func TestWalkVisitsInvoiceFilesBelowTheRootInOrder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, name := range []string{"b.yaml", "a/x.md", "a/y.markdown", "a/z.YML", "notes.txt", ".history/old.yaml", "a/.history/kept.yaml"} {
		writeFile(t, filepath.Join(dir, filepath.FromSlash(name)), "")
	}

	got := walkPaths(t, Store{Dir: dir})
	want := []string{
		filepath.Join(dir, "a", ".history", "kept.yaml"),
		filepath.Join(dir, "a", "x.md"),
		filepath.Join(dir, "a", "y.markdown"),
		filepath.Join(dir, "a", "z.YML"),
		filepath.Join(dir, "b.yaml"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Walk visited %q, want %q", got, want)
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

	entries, err := Store{Dir: dir}.List(func(string) (billing.ArchiveEntry, bool, error) { return billing.ArchiveEntry{}, true, nil })
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
	err := Store{Dir: file}.Walk(func(string) error { return nil })
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

	entries, err := Store{Dir: dir}.List(func(path string) (billing.ArchiveEntry, bool, error) {
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
}

func TestTargetEdit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		rel  string
		want Edit
	}{
		{rel: filepath.Join("customer-a", "first.yaml"), want: Edit{Filename: "first.yaml", Target: filepath.Join("customer-a", "first.yaml")}},
		{rel: filepath.Join("customer-a", "first.md"), want: Edit{Filename: "first.yaml", Target: filepath.Join("customer-a", "first.yaml"), Replace: filepath.Join("customer-a", "first.md")}},
		{rel: "first.MARKDOWN", want: Edit{Filename: "first.yaml", Target: "first.yaml", Replace: "first.MARKDOWN"}},
	}
	for _, tt := range tests {
		if got := (Target{Rel: tt.rel}).Edit(); got != tt.want {
			t.Errorf("Edit(%q) = %+v, want %+v", tt.rel, got, tt.want)
		}
	}
}

func TestBackupKeepsEveryVersion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "customer-a", "first.md")
	writeFile(t, path, "v1")
	now := time.Date(2026, 10, 5, 14, 30, 45, 0, time.FixedZone("CEST", 2*60*60))
	s := Store{Dir: dir}

	first, err := s.Backup([]string{path}, now)
	if err != nil {
		t.Fatalf("Backup returned error: %v", err)
	}
	want := []billing.Backup{{Path: path, BackupPath: filepath.Join(dir, ".history", "customer-a", "first.20261005T123045Z.md")}}
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("Backup = %+v, want %+v", first, want)
	}

	writeFile(t, path, "v2")
	second, err := s.Backup([]string{path}, now)
	if err != nil {
		t.Fatalf("second Backup returned error: %v", err)
	}
	if want := filepath.Join(dir, ".history", "customer-a", "first.20261005T123045Z-2.md"); len(second) != 1 || second[0].BackupPath != want {
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

	for path, want := range map[string]bool{"a.yaml": true, "a.YML": true, "a.md": true, "a.Markdown": true, "a.txt": false, "yaml": false} {
		if got := isInvoiceFile(path); got != want {
			t.Errorf("isInvoiceFile(%q) = %v, want %v", path, got, want)
		}
	}
}
