package app

import (
	"testing"

	"github.com/0xboris/invox/internal/adapters/programs/run"
	"github.com/0xboris/invox/internal/utilities/env"
	"github.com/0xboris/invox/internal/utilities/iostreams"
)

func TestAppleMailIsWiredOnlyOnMacOS(t *testing.T) {
	for _, tc := range []struct {
		goos       string
		wantMailer bool
	}{
		{"darwin", true},
		{"linux", false},
		{"windows", false},
	} {
		t.Run(tc.goos, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			d := NewDependencies(ios, run.NewStub(t), env.Env{GOOS: tc.goos, Getenv: func(string) string { return "" }})
			if got := d.Mailer != nil; got != tc.wantMailer {
				t.Fatalf("Mailer set = %v, want %v", got, tc.wantMailer)
			}
		})
	}
}
