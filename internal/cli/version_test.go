package cli

import (
	"strings"
	"testing"
)

func TestRootHelpListsVersion(t *testing.T) {
	_, stdout, _ := captureRun(t, []string{"--help"})
	if !strings.Contains(stdout, "  version     Show the invox version\n") {
		t.Fatalf("root help does not list the version subcommand:\n%s", stdout)
	}
	if !strings.Contains(stdout, "      --version         Show the invox version\n") {
		t.Fatalf("root help does not list --version:\n%s", stdout)
	}
}
