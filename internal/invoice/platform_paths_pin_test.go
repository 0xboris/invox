//go:build linux

package invoice

import "testing"

func TestPlatformPathsOnLinuxPin(t *testing.T) {
	tests := []struct {
		name                         string
		home, xdgConfig, xdgData     string
		wantConfig, wantLegacy       string
		wantArchive, wantTemplateDir string
		wantTilde                    string
	}{
		{
			name: "XDG set with spaces", home: "/home/ada", xdgConfig: "  /cfg  ", xdgData: " /data ",
			wantConfig: "/cfg/invox", wantLegacy: "/cfg/invoice-tool/config.yaml",
			wantArchive: "/data/invox/invoices", wantTemplateDir: "/data/invox/invoices", wantTilde: "/home/ada/x.yaml",
		},
		{
			name: "XDG unset", home: "/home/ada",
			wantConfig: "/home/ada/.config/invox", wantLegacy: "/home/ada/.config/invoice-tool/config.yaml",
			wantArchive: "/home/ada/.local/share/invox/invoices", wantTemplateDir: "~/.local/share/invox/invoices", wantTilde: "/home/ada/x.yaml",
		},
		{
			name:       "HOME empty",
			wantConfig: "", wantLegacy: "config.yaml",
			wantArchive: "", wantTemplateDir: "", wantTilde: "~/x.yaml",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOME", tc.home)
			t.Setenv("XDG_CONFIG_HOME", tc.xdgConfig)
			t.Setenv("XDG_DATA_HOME", tc.xdgData)
			if got := ConfigDir(); got != tc.wantConfig {
				t.Errorf("ConfigDir() = %q, want %q", got, tc.wantConfig)
			}
			if got := LegacyConfigPath(); got != tc.wantLegacy {
				t.Errorf("LegacyConfigPath() = %q, want %q", got, tc.wantLegacy)
			}
			if got := DefaultArchiveDir(); got != tc.wantArchive {
				t.Errorf("DefaultArchiveDir() = %q, want %q", got, tc.wantArchive)
			}
			if got := configTemplatePath(DefaultArchiveDir()); got != tc.wantTemplateDir {
				t.Errorf("configTemplatePath(DefaultArchiveDir()) = %q, want %q", got, tc.wantTemplateDir)
			}
			if got := expandHomePath("~/x.yaml"); got != tc.wantTilde {
				t.Errorf("expandHomePath(~/x.yaml) = %q, want %q", got, tc.wantTilde)
			}
		})
	}
}
