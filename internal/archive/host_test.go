package archive_test

import (
	"testing"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/testfixture"
)

// service returns the use cases as invox wires them for h, so its
// Archives is the archive of h's config.
func service(t *testing.T, h testfixture.Host) *billing.Service {
	t.Helper()
	return factorytest.New(t, nil, factorytest.Options{
		Home: h.Home,
		Vars: map[string]string{"XDG_CONFIG_HOME": h.ConfigHome},
	}).Service(cmdutil.Files{})
}
