package migrations

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMigrationsDoNotDependOnLiveModelPackages keeps released migrations frozen: production migration
// code must not import module or capability packages whose persistence structs keep evolving.
// Tests may still import them to assert behavior.
func TestMigrationsDoNotDependOnLiveModelPackages(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").CombinedOutput()
	require.NoError(t, err, string(out))

	for _, pkg := range strings.Fields(string(out)) {
		for _, forbidden := range []string{
			"github.com/zgiai/luas/api/internal/modules/",
			"github.com/zgiai/luas/api/internal/capabilities/",
		} {
			require.Falsef(t, strings.HasPrefix(pkg, forbidden),
				"database/migrations depends on %s; freeze migration DDL as SQL instead", pkg)
		}
	}
}
