package store

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", path, err)
	}
}

func archiveConfig(dir string) string {
	return "archive:\n  dir: '" + dir + "'\n"
}

func TestConfigIsReadOncePerHost(t *testing.T) {
	root := t.TempDir()
	firstArchive := filepath.Join(root, "first-archive")
	h := writeConfigFile(t, archiveConfig(firstArchive)+
		"numbering:\n  start: 5\n  pattern: '{customer_code}_{counter:02}'\n"+
		"email:\n  subject: 'First {invoice_number}'\n"+
		"paths:\n  customers: '"+filepath.Join(root, "first-customers.yaml")+"'\n")

	read := func(h Host) []string {
		t.Helper()
		archiveDir, err := h.ResolveArchiveDir()
		if err != nil {
			t.Fatalf("ResolveArchiveDir returned error: %v", err)
		}
		numbering, err := h.ResolveNumberingSettings()
		if err != nil {
			t.Fatalf("ResolveNumberingSettings returned error: %v", err)
		}
		settings, err := h.Settings()
		if err != nil {
			t.Fatalf("Settings returned error: %v", err)
		}
		subject := strings.ReplaceAll(settings.EmailSubject, "{invoice_number}", "C-01")
		customers, err := h.ResolveSupportFile(Customers, t.TempDir())
		if err != nil {
			t.Fatalf("ResolveSupportFile returned error: %v", err)
		}
		return []string{archiveDir, numbering.Pattern, subject, filepath.Base(customers.Path)}
	}
	want := []string{firstArchive, "{customer_code}_{counter:02}", "First C-01", "first-customers.yaml"}

	if got := read(h); !reflect.DeepEqual(got, want) {
		t.Fatalf("first read = %q, want %q", got, want)
	}
	if err := os.WriteFile(filepath.Join(h.ConfigDir(), "config.yaml"), []byte("archive: [broken\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	copied := h
	for name, host := range map[string]Host{"same Host": h, "copy of the Host": copied} {
		if got := read(host); !reflect.DeepEqual(got, want) {
			t.Fatalf("%s after rewriting config.yaml = %q, want the first values %q", name, got, want)
		}
	}
}

// configDirs lays out a config home with config.yaml in the invox dir, the
// legacy dir and an env dir, plus an explicit config file. Each sets a
// different archive.dir, named after where it lives.
func configDirs(t *testing.T) (root string, in HostInputs) {
	t.Helper()
	root = t.TempDir()
	configHome := filepath.Join(root, "config-home")
	for _, dir := range []string{
		filepath.Join(configHome, "invox"),
		filepath.Join(configHome, "invoice-tool"),
		filepath.Join(root, "env"),
	} {
		writeFile(t, filepath.Join(dir, "config.yaml"), archiveConfig(filepath.Join(root, filepath.Base(dir)+"-archive")))
	}
	explicit := filepath.Join(root, "explicit.yaml")
	writeFile(t, explicit, archiveConfig(filepath.Join(root, "explicit-archive")))
	return root, HostInputs{GOOS: "linux", Home: filepath.Join(root, "home"), XDGConfigHome: configHome}
}

func TestConfigFilePrecedence(t *testing.T) {
	tests := []struct {
		name          string
		explicit, env bool
		removeDefault bool
		want          string
	}{
		{name: "explicit file beats env dir", explicit: true, env: true, want: "explicit-archive"},
		{name: "env dir beats default dir", env: true, want: "env-archive"},
		{name: "default dir beats legacy dir", want: "invox-archive"},
		{name: "legacy dir when default has no config.yaml", removeDefault: true, want: "invoice-tool-archive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, in := configDirs(t)
			if tt.explicit {
				in.ConfigFile = filepath.Join(root, "explicit.yaml")
			}
			if tt.env {
				in.ConfigDir = filepath.Join(root, "env")
			}
			if tt.removeDefault {
				if err := os.Remove(filepath.Join(in.XDGConfigHome, "invox", "config.yaml")); err != nil {
					t.Fatalf("Remove returned error: %v", err)
				}
			}

			got, err := NewHost(in).ResolveArchiveDir()
			if err != nil {
				t.Fatalf("ResolveArchiveDir returned error: %v", err)
			}
			if want := filepath.Join(root, tt.want); got != want {
				t.Fatalf("ResolveArchiveDir() = %q, want %q", got, want)
			}
		})
	}
}

func TestConfigLocationErrors(t *testing.T) {
	root, in := configDirs(t)

	missingFile := in
	missingFile.ConfigFile = filepath.Join(root, "missing.yaml")
	_, err := NewHost(missingFile).Config()
	if want := "config file " + missingFile.ConfigFile + " does not exist"; err == nil || err.Error() != want {
		t.Fatalf("Config() with a missing explicit file error = %v, want %q", err, want)
	}

	missingDir := in
	missingDir.ConfigDir = filepath.Join(root, "no-such-dir")
	_, err = NewHost(missingDir).ResolveArchiveDir()
	if want := "config directory " + missingDir.ConfigDir + " does not exist"; err == nil || err.Error() != want {
		t.Fatalf("ResolveArchiveDir() with a missing env dir error = %v, want %q", err, want)
	}

	emptyDir := in
	emptyDir.ConfigDir = t.TempDir()
	got, err := NewHost(emptyDir).ResolveArchiveDir()
	if err != nil {
		t.Fatalf("ResolveArchiveDir() with an env dir without config.yaml returned error: %v", err)
	}
	if want := filepath.Join(in.Home, ".local", "share", "invox", "invoices"); got != want {
		t.Fatalf("ResolveArchiveDir() = %q, want the default %q", got, want)
	}
}

func TestLegacyFallbackIsPerFileAndRecorded(t *testing.T) {
	root, in := configDirs(t)
	invoxDir := filepath.Join(in.XDGConfigHome, "invox")
	legacyDir := filepath.Join(in.XDGConfigHome, "invoice-tool")
	writeFile(t, filepath.Join(invoxDir, "customers.yaml"), "{}\n")
	writeFile(t, filepath.Join(legacyDir, "customers.yaml"), "{}\n")
	writeFile(t, filepath.Join(legacyDir, "issuer.yaml"), "{}\n")
	work := filepath.Join(root, "work")

	h := NewHost(in)
	customers, err := h.ResolveSupportFile(Customers, work)
	if err != nil {
		t.Fatalf("ResolveSupportFile(Customers) returned error: %v", err)
	}
	issuer, err := h.ResolveSupportFile(Issuer, work)
	if err != nil {
		t.Fatalf("ResolveSupportFile(Issuer) returned error: %v", err)
	}
	if want := (Resolved{Path: filepath.Join(invoxDir, "customers.yaml"), Source: SourceDefault}); customers != want {
		t.Fatalf("customers = %+v, want %+v", customers, want)
	}
	if want := (Resolved{Path: filepath.Join(legacyDir, "issuer.yaml"), Source: SourceLegacy}); issuer != want {
		t.Fatalf("issuer = %+v, want %+v", issuer, want)
	}
	copied := h
	if got, want := copied.LegacyFilesUsed(), []string{filepath.Join(legacyDir, "issuer.yaml")}; !reflect.DeepEqual(got, want) {
		t.Fatalf("LegacyFilesUsed() = %q, want %q", got, want)
	}

	in.ConfigDir = filepath.Join(root, "env")
	h = NewHost(in)
	issuer, err = h.ResolveSupportFile(Issuer, work)
	if err != nil {
		t.Fatalf("ResolveSupportFile(Issuer) with an env dir returned error: %v", err)
	}
	if issuer != (Resolved{}) || len(h.LegacyFilesUsed()) != 0 {
		t.Fatalf("with an env dir: issuer = %+v, legacy used %q; want nothing from the legacy dir", issuer, h.LegacyFilesUsed())
	}
}

func TestPathsReportsEachSource(t *testing.T) {
	root, in := configDirs(t)
	invoxDir := filepath.Join(in.XDGConfigHome, "invox")
	legacyDir := filepath.Join(in.XDGConfigHome, "invoice-tool")
	work := filepath.Join(root, "work")
	writeFile(t, filepath.Join(work, "customers.yaml"), "{}\n")
	writeFile(t, filepath.Join(legacyDir, "issuer.yaml"), "{}\n")
	writeFile(t, filepath.Join(invoxDir, "template.tex"), "x\n")
	writeFile(t, filepath.Join(invoxDir, "config.yaml"), "paths:\n  defaults: 'd.yaml'\n")

	got, err := NewHost(in).Paths(work)
	if err != nil {
		t.Fatalf("Paths returned error: %v", err)
	}
	want := []PathReport{
		{"config-dir", Resolved{invoxDir, SourceDefault}},
		{"config", Resolved{filepath.Join(invoxDir, "config.yaml"), SourceDefault}},
		{"customers", Resolved{filepath.Join(work, "customers.yaml"), SourceProject}},
		{"issuer", Resolved{filepath.Join(legacyDir, "issuer.yaml"), SourceLegacy}},
		{"defaults", Resolved{filepath.Join(invoxDir, "d.yaml"), SourceConfig}},
		{"template", Resolved{filepath.Join(invoxDir, "template.tex"), SourceDefault}},
		{"archive", Resolved{filepath.Join(in.Home, ".local", "share", "invox", "invoices"), SourceDefault}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Paths() =\n%+v\nwant\n%+v", got, want)
	}

	writeFile(t, filepath.Join(root, "bad.yaml"), "numbering:\n  patern: x\n")
	in.ConfigFile = filepath.Join(root, "bad.yaml")
	if _, err := NewHost(in).Paths(work); err == nil || !strings.Contains(err.Error(), `bad.yaml:2: unknown key "patern" in numbering`) {
		t.Fatalf("Paths() with a broken config error = %v, want the unknown key", err)
	}
}

func TestUpwardSearchIsBounded(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		start string
		want  string
	}{
		{name: "stops below home", files: []string{"home/customers.yaml"}, start: "home/a/b", want: ""},
		{name: "walks the directories below home", files: []string{"home/customers.yaml", "home/a/customers.yaml"}, start: "home/a/b", want: "home/a/customers.yaml"},
		{name: "searches home when it is the start", files: []string{"home/customers.yaml"}, start: "home", want: "home/customers.yaml"},
		{name: "stops at a .git directory", files: []string{"home/p/customers.yaml", "home/p/proj/.git/HEAD"}, start: "home/p/proj/a", want: ""},
		{name: "stops at a .git file", files: []string{"home/p/customers.yaml", "home/p/proj/.git"}, start: "home/p/proj/a", want: ""},
		{name: "includes the marker directory", files: []string{"home/p/proj/customers.yaml", "home/p/proj/.git/HEAD"}, start: "home/p/proj/a", want: "home/p/proj/customers.yaml"},
		{name: "stops at invoice_defaults.yaml", files: []string{"home/p/customers.yaml", "home/p/proj/invoice_defaults.yaml"}, start: "home/p/proj/a", want: ""},
		{name: "stops at invox.yaml", files: []string{"home/p/customers.yaml", "home/p/proj/invox.yaml"}, start: "home/p/proj", want: ""},
		{name: "outside home only the start dir", files: []string{"out/customers.yaml"}, start: "out/sub", want: ""},
		{name: "outside home the start dir itself", files: []string{"out/customers.yaml"}, start: "out", want: "out/customers.yaml"},
		{name: "outside home up to a marker", files: []string{"out/customers.yaml", "out/.git/HEAD"}, start: "out/sub", want: "out/customers.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for _, file := range tt.files {
				writeFile(t, filepath.Join(root, filepath.FromSlash(file)), "{}\n")
			}
			start := filepath.Join(root, filepath.FromSlash(tt.start))
			if err := os.MkdirAll(start, 0o755); err != nil {
				t.Fatalf("MkdirAll returned error: %v", err)
			}
			h := testHost(filepath.Join(root, "config-home"), filepath.Join(root, "home"))

			got, err := h.ResolveSupportFile(Customers, start)
			if err != nil {
				t.Fatalf("ResolveSupportFile returned error: %v", err)
			}
			want := Resolved{}
			if tt.want != "" {
				want = Resolved{Path: filepath.Join(root, filepath.FromSlash(tt.want)), Source: SourceProject}
			}
			if got != want {
				t.Fatalf("ResolveSupportFile(Customers, %s) = %+v, want %+v", tt.start, got, want)
			}
		})
	}
}

func TestUpwardSearchComparesHomeIgnoringCaseOnMacOS(t *testing.T) {
	root := t.TempDir()
	start := filepath.Join(root, "home", "a", "b")
	h := NewHost(HostInputs{GOOS: "darwin", Home: filepath.Join(root, "HOME"), XDGConfigHome: filepath.Join(root, "config-home")})

	want := []string{start, filepath.Dir(start)}
	if got := h.projectDirs(start); !reflect.DeepEqual(got, want) {
		t.Fatalf("projectDirs(%s) = %q, want %q", start, got, want)
	}
}

func TestCopyLegacyFilesNeverReplacesAndIsIdempotent(t *testing.T) {
	_, in := configDirs(t)
	invoxDir := filepath.Join(in.XDGConfigHome, "invox")
	legacyDir := filepath.Join(in.XDGConfigHome, "invoice-tool")
	writeFile(t, filepath.Join(legacyDir, "customers.yaml"), "legacy customers\n")
	writeFile(t, filepath.Join(legacyDir, "logos", "logo.png"), "png\n")
	h := NewHost(in)

	missing, err := h.LegacyFilesToCopy()
	if err != nil {
		t.Fatalf("LegacyFilesToCopy returned error: %v", err)
	}
	if want := []string{"customers.yaml", filepath.Join("logos", "logo.png")}; !reflect.DeepEqual(missing, want) {
		t.Fatalf("LegacyFilesToCopy() = %q, want %q (config.yaml exists in both)", missing, want)
	}
	copied, err := h.CopyLegacyFiles()
	if err != nil || !reflect.DeepEqual(copied, missing) {
		t.Fatalf("CopyLegacyFiles() = %q, %v; want %q", copied, err, missing)
	}
	for rel, want := range map[string]string{
		"customers.yaml":                   "legacy customers\n",
		filepath.Join("logos", "logo.png"): "png\n",
		"config.yaml":                      archiveConfig(filepath.Join(filepath.Dir(in.XDGConfigHome), "invox-archive")),
		filepath.Join("..", "invoice-tool", "customers.yaml"): "legacy customers\n",
	} {
		got, err := os.ReadFile(filepath.Join(invoxDir, rel))
		if err != nil || string(got) != want {
			t.Errorf("%s = %q, %v; want %q", rel, got, err, want)
		}
	}

	copied, err = NewHost(in).CopyLegacyFiles()
	if err != nil || len(copied) != 0 {
		t.Fatalf("second CopyLegacyFiles() = %q, %v; want nothing copied", copied, err)
	}
	issuer, err := NewHost(in).ResolveSupportFile(Customers, t.TempDir())
	if err != nil || issuer.Source != SourceDefault {
		t.Fatalf("customers after the copy = %+v, %v; want it from the invox directory", issuer, err)
	}
}

func TestCopyLegacyFilesFollowsSymlinks(t *testing.T) {
	_, in := configDirs(t)
	legacyDir := filepath.Join(in.XDGConfigHome, "invoice-tool")
	target := filepath.Join(t.TempDir(), "dotfiles-customers.yaml")
	writeFile(t, target, "linked customers\n")
	if err := os.Symlink(target, filepath.Join(legacyDir, "customers.yaml")); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
	h := NewHost(in)

	copied, err := h.CopyLegacyFiles()
	if err != nil || !reflect.DeepEqual(copied, []string{"customers.yaml"}) {
		t.Fatalf("CopyLegacyFiles() = %q, %v; want [customers.yaml]", copied, err)
	}
	got, err := os.ReadFile(filepath.Join(in.XDGConfigHome, "invox", "customers.yaml"))
	if err != nil || string(got) != "linked customers\n" {
		t.Fatalf("copied customers.yaml = %q, %v; want the symlink target's content", got, err)
	}
}

func TestUpwardSearchStopsBelowASymlinkedHome(t *testing.T) {
	_, in := configDirs(t)
	realHome := filepath.Join(t.TempDir(), "real-home")
	writeFile(t, filepath.Join(realHome, ".git", "HEAD"), "ref: refs/heads/main\n")
	// The working directory comes back with symlinks resolved, and the temp
	// directory is itself behind one on macOS (/var) and Windows (8.3 names).
	realHome, err := filepath.EvalSymlinks(realHome)
	if err != nil {
		t.Fatal(err)
	}
	stray := filepath.Join(realHome, "customers.yaml")
	writeFile(t, stray, "stray\n")
	start := filepath.Join(realHome, "invoices", "2026")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}
	linkedHome := filepath.Join(t.TempDir(), "home")
	if err := os.Symlink(realHome, linkedHome); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
	in.Home = linkedHome

	got, err := NewHost(in).ResolveSupportFile(Customers, start)
	if err != nil {
		t.Fatalf("ResolveSupportFile returned error: %v", err)
	}
	if got.Path == stray {
		t.Fatalf("ResolveSupportFile found %s in the symlinked home; want the search to stop below it", stray)
	}
}
