package starter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultManifestsRegisterDefaultAssets(t *testing.T) {
	registry := NewRegistry()

	manifests := DefaultManifests(nil, nil, nil)
	require.Len(t, manifests, 3)
	assert.Equal(t, "audit", manifests[0].Name())
	assert.Equal(t, "apikey", manifests[1].Name())
	assert.Equal(t, "user", manifests[2].Name())

	for _, manifest := range manifests {
		require.NoError(t, registry.ApplyManifest(manifest))
	}

	migrations := registry.Migrations()
	assert.Len(t, migrations, 7)
	assert.Contains(t, migrations, "2026_04_26_000000_create_audit_logs_table")
	assert.Contains(t, migrations, "2026_04_27_000002_add_business_fields_to_audit_logs")
	assert.Contains(t, migrations, "2025_06_18_000000_create_users_table")
	assert.Contains(t, migrations, "2025_06_18_000001_seed_default_users")
	assert.Contains(t, migrations, "2026_04_27_000000_create_password_reset_tokens_table")
	assert.Contains(t, migrations, "2026_04_27_000001_add_unique_index_to_users_username")
	assert.Contains(t, migrations, "2026_04_06_000000_create_api_keys_table")

	seeders := registry.Seeders()
	require.Len(t, seeders, 1)
	assert.Equal(t, "users", seeders[0].Name())
}
