package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/zgiai/luas/api/internal/modules/organization"
	"github.com/zgiai/luas/api/internal/modules/user"
)

func TestCreateOrganizationInvitationsTableUpAndDown(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&user.UserPO{}))
	require.NoError(t, (&createOrganizationsTables{}).Up(db))
	migration := &createOrganizationInvitationsTable{}

	require.NoError(t, migration.Up(db))
	assert.True(t, db.Migrator().HasTable(&organization.OrganizationInvitationPO{}))
	assert.True(t, db.Migrator().HasIndex(
		&organization.OrganizationInvitationPO{},
		"idx_organization_invitations_pending_key",
	))
	assert.True(t, db.Migrator().HasIndex(
		&organization.OrganizationInvitationPO{},
		"idx_organization_invitations_token_hash",
	))

	require.NoError(t, migration.Down(db))
	assert.False(t, db.Migrator().HasTable(&organization.OrganizationInvitationPO{}))
	assert.True(t, db.Migrator().HasTable(&organization.OrganizationPO{}))
	assert.True(t, db.Migrator().HasTable(&organization.OrganizationMembershipPO{}))
}
