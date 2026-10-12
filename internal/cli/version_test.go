package cli_test

import (
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
)

func TestRootHelpListsVersion(t *testing.T) {
	x := clitest.New(t)

	_, stdout, _ := x.Run([]string{"--help"})
	if !strings.Contains(stdout, "  version     Show the invox version\n") {
		t.Fatalf("root help does not list the version subcommand:\n%s", stdout)
	}
	if !strings.Contains(stdout, "      --version         Show the invox version\n") {
		t.Fatalf("root help does not list --version:\n%s", stdout)
	}
}
