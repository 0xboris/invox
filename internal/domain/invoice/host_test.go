package invoice

import (
	"path/filepath"
	"testing"
)

func TestNewHostResolvesUserDirectories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                         string
		goos                         string
		home, xdgConfig, xdgData     string
		appData                      string
		wantConfig, wantLegacy       string
		wantArchive, wantTemplateDir string
		wantTilde, wantHome          string
	}{
		{
			name: "linux XDG set with spaces", goos: "linux", home: "/home/ada", xdgConfig: "  /cfg  ", xdgData: " /data ",
			wantConfig: "/cfg/invox", wantLegacy: "/cfg/invoice-tool",
			wantArchive: "/data/invox/invoices", wantTemplateDir: "/data/invox/invoices",
			wantTilde: "/home/ada/x.yaml", wantHome: "/home/ada",
		},
		{
			name: "linux XDG unset", goos: "linux", home: "/home/ada", appData: "/ignored",
			wantConfig: "/home/ada/.config/invox", wantLegacy: "/home/ada/.config/invoice-tool",
			wantArchive: "/home/ada/.local/share/invox/invoices", wantTemplateDir: "~/.local/share/invox/invoices",
			wantTilde: "/home/ada/x.yaml", wantHome: "/home/ada",
		},
		{
			name: "linux home empty", goos: "linux",
			wantConfig: "", wantLegacy: "",
			wantArchive: "", wantTemplateDir: "",
			wantTilde: "~/x.yaml", wantHome: "~",
		},
		{
			name: "linux home blank", goos: "linux", home: "   ",
			wantConfig: "", wantLegacy: "",
			wantArchive: "", wantTemplateDir: "",
			wantTilde: "~/x.yaml", wantHome: "~",
		},
		{
			name: "darwin honours XDG data", goos: "darwin", home: "/Users/ada", xdgData: "/data",
			wantConfig: "/Users/ada/.config/invox", wantLegacy: "/Users/ada/.config/invoice-tool",
			wantArchive: "/data/invox/invoices", wantTemplateDir: "/data/invox/invoices",
			wantTilde: "/Users/ada/x.yaml", wantHome: "/Users/ada",
		},
		{
			name: "darwin XDG config set with spaces", goos: "darwin", home: "/Users/ada", xdgConfig: " /cfg ",
			wantConfig: "/cfg/invox", wantLegacy: "/cfg/invoice-tool",
			wantArchive: "/Users/ada/Library/Application Support/invox/invoices", wantTemplateDir: "~/Library/Application Support/invox/invoices",
			wantTilde: "/Users/ada/x.yaml", wantHome: "/Users/ada",
		},
		{
			name: "darwin home empty", goos: "darwin",
			wantConfig: "", wantLegacy: "",
			wantArchive: "", wantTemplateDir: "",
			wantTilde: "~/x.yaml", wantHome: "~",
		},
		{
			name: "windows APPDATA unset", goos: "windows", home: "C:/Users/ada", xdgData: "/data",
			wantConfig: "C:/Users/ada/.config/invox", wantLegacy: "C:/Users/ada/.config/invoice-tool",
			wantArchive: "C:/Users/ada/AppData/Roaming/invox/invoices", wantTemplateDir: "~/AppData/Roaming/invox/invoices",
			wantTilde: "C:/Users/ada/x.yaml", wantHome: "C:/Users/ada",
		},
		{
			name: "windows APPDATA set with spaces", goos: "windows", home: "C:/Users/ada", appData: "  D:/Roaming  ",
			wantConfig: "C:/Users/ada/.config/invox", wantLegacy: "C:/Users/ada/.config/invoice-tool",
			wantArchive: "D:/Roaming/invox/invoices", wantTemplateDir: "D:/Roaming/invox/invoices",
			wantTilde: "C:/Users/ada/x.yaml", wantHome: "C:/Users/ada",
		},
		{
			name: "windows home empty with APPDATA", goos: "windows", appData: "D:/Roaming",
			wantConfig: "", wantLegacy: "",
			wantArchive: "D:/Roaming/invox/invoices", wantTemplateDir: "D:/Roaming/invox/invoices",
			wantTilde: "~/x.yaml", wantHome: "~",
		},
		{
			name: "windows home empty without APPDATA", goos: "windows", appData: "  ",
			wantConfig: "", wantLegacy: "",
			wantArchive: "", wantTemplateDir: "",
			wantTilde: "~/x.yaml", wantHome: "~",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := NewHost(HostInputs{
				GOOS:          tc.goos,
				Home:          filepath.FromSlash(tc.home),
				XDGConfigHome: filepath.FromSlash(tc.xdgConfig),
				XDGDataHome:   filepath.FromSlash(tc.xdgData),
				AppData:       filepath.FromSlash(tc.appData),
			})
			for _, check := range []struct{ what, got, want string }{
				{"ConfigDir()", h.ConfigDir(), tc.wantConfig},
				{"LegacyConfigDir()", h.LegacyConfigDir(), tc.wantLegacy},
				{"DefaultArchiveDir()", h.DefaultArchiveDir(), tc.wantArchive},
				{"configTemplatePath(DefaultArchiveDir())", h.configTemplatePath(h.DefaultArchiveDir()), tc.wantTemplateDir},
				{"expandHomePath(~/x.yaml)", h.expandHomePath("~/x.yaml"), tc.wantTilde},
				{"expandHomePath(~)", h.expandHomePath("~"), tc.wantHome},
			} {
				if got := filepath.ToSlash(check.got); got != check.want {
					t.Errorf("%s = %q, want %q", check.what, got, check.want)
				}
			}
		})
	}
}
