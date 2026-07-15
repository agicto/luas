package permission

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

func TestCatalogValidatesSortsAndDefensivelyCopiesKeys(t *testing.T) {
	catalog, err := NewCatalog(
		domain.PermissionKey("projects.write"),
		domain.PermissionKey("projects.read"),
	)
	require.NoError(t, err)

	keys := catalog.Keys()
	assert.Equal(t, []domain.PermissionKey{"projects.read", "projects.write"}, keys)
	keys[0] = "changed.value"
	assert.Equal(t, domain.PermissionKey("projects.read"), catalog.Keys()[0])
	assert.True(t, catalog.Contains("projects.read"))
	assert.False(t, catalog.Contains("projects"))
}

func TestCatalogRejectsInvalidAndDuplicateKeys(t *testing.T) {
	tests := []struct {
		name string
		keys []domain.PermissionKey
	}{
		{name: "one segment", keys: []domain.PermissionKey{"projects"}},
		{name: "uppercase", keys: []domain.PermissionKey{"Projects.read"}},
		{name: "wildcard", keys: []domain.PermissionKey{"projects.*"}},
		{name: "duplicate", keys: []domain.PermissionKey{"projects.read", "projects.read"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCatalog(tt.keys...)
			require.Error(t, err)
		})
	}
}

func TestDefaultCatalogContainsPermissionManagementKeys(t *testing.T) {
	catalog, err := NewDefaultCatalog()
	require.NoError(t, err)

	assert.True(t, catalog.Contains(PermissionRolesRead))
	assert.True(t, catalog.Contains(PermissionRolesManage))
	assert.True(t, catalog.Contains(PermissionAssignmentsRead))
	assert.True(t, catalog.Contains(PermissionAssignmentsManage))
}
