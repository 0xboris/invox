package store

import (
	"path/filepath"
	"testing"
)

func TestReplaceExt(t *testing.T) {
	tests := []struct {
		path, ext, want string
	}{
		{filepath.Join("a", "invoice.yaml"), ".pdf", filepath.Join("a", "invoice.pdf")},
		{"invoice", ".pdf", "invoice.pdf"},
		{filepath.Join("v1.2", "invoice"), ".eml", filepath.Join("v1.2", "invoice.eml")},
		{"invoice.tar.yaml", ".pdf", "invoice.tar.pdf"},
		{"", ".pdf", ""},
		{"  ", ".pdf", ""},
	}
	for _, tt := range tests {
		if got := ReplaceExt(tt.path, tt.ext); got != tt.want {
			t.Errorf("ReplaceExt(%q, %q) = %q, want %q", tt.path, tt.ext, got, tt.want)
		}
	}
}
