package starter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultManifestsRegisterDefaultAssets(t *testing.T) {
	registry := NewRegistry()

	manifests := DefaultManifests(nil, nil)
	require.Len(t, manifests, 2)
	assert.Equal(t, "apikey", manifests[0].Name())
	assert.Equal(t, "user", manifests[1].Name())

	for _, manifest := range manifests {
		require.NoError(t, registry.ApplyManifest(manifest))
	}

	migrations := registry.Migrations()
	assert.Len(t, migrations, 3)
	assert.Contains(t, migrations, "2025_06_18_000000_create_users_table")
	assert.Contains(t, migrations, "2025_06_18_000001_seed_default_users")
	assert.Contains(t, migrations, "2026_04_06_000000_create_api_keys_table")

	seeders := registry.Seeders()
	require.Len(t, seeders, 1)
	assert.Equal(t, "users", seeders[0].Name())
}
