package starter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/infra/config"
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

func TestConfiguredManifestsEnableOrganizationAdditively(t *testing.T) {
	cfg := &config.Config{Starters: config.StarterConfig{Optional: []string{"organization"}}}

	manifests, err := ConfiguredManifests(cfg, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, manifests, 4)
	assert.Equal(t, "audit", manifests[0].Name())
	assert.Equal(t, "apikey", manifests[1].Name())
	assert.Equal(t, "user", manifests[2].Name())
	assert.Equal(t, "organization", manifests[3].Name())

	migrations, err := ConfiguredMigrations(cfg)
	require.NoError(t, err)
	assert.Len(t, migrations, 9)
	organizationMigration, exists := migrations["2026_07_14_000000_create_organizations_tables"]
	require.True(t, exists)
	assert.True(t, organizationMigration.WithinTransaction())
	invitationMigration, exists := migrations["2026_07_15_000000_create_organization_invitations_table"]
	require.True(t, exists)
	assert.True(t, invitationMigration.WithinTransaction())
}

func TestConfiguredManifestsEnablePermissionAfterOrganization(t *testing.T) {
	cfg := &config.Config{Starters: config.StarterConfig{Optional: []string{"permission", "organization"}}}

	manifests, err := ConfiguredManifests(cfg, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, manifests, 5)
	assert.Equal(t, "organization", manifests[3].Name())
	assert.Equal(t, "permission", manifests[4].Name())

	migrations, err := ConfiguredMigrations(cfg)
	require.NoError(t, err)
	assert.Len(t, migrations, 10)
	permissionMigration, exists := migrations["2026_07_15_010000_create_permission_tables"]
	require.True(t, exists)
	assert.True(t, permissionMigration.WithinTransaction())
}

func TestConfiguredManifestsRequireOrganizationForPermission(t *testing.T) {
	cfg := &config.Config{Starters: config.StarterConfig{Optional: []string{"permission"}}}

	_, err := ConfiguredManifests(cfg, nil, nil, nil, nil, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `optional starter "permission" requires "organization"`)
}

func TestConfiguredManifestsRejectUnknownOptionalStarter(t *testing.T) {
	cfg := &config.Config{Starters: config.StarterConfig{Optional: []string{"billing"}}}

	_, err := ConfiguredManifests(cfg, nil, nil, nil, nil, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown optional starter")
}

func TestConfiguredManifestsEnableNotificationWithoutOrganization(t *testing.T) {
	cfg := &config.Config{Starters: config.StarterConfig{Optional: []string{"notification"}}}

	manifests, err := ConfiguredManifests(cfg, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, manifests, 4)
	assert.Equal(t, "notification", manifests[3].Name())

	migrations, err := ConfiguredMigrations(cfg)
	require.NoError(t, err)
	assert.Len(t, migrations, 8)
	notificationMigration, exists := migrations["2026_07_15_020000_create_notification_tables"]
	require.True(t, exists)
	assert.True(t, notificationMigration.WithinTransaction())
}
