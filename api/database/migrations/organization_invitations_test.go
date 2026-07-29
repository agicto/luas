package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testplatform "github.com/zgiai/luas/api/internal/infra/testing"
	"github.com/zgiai/luas/api/internal/modules/organization"
	"github.com/zgiai/luas/api/internal/modules/user"
)

func TestCreateOrganizationInvitationsTableUpAndDown(t *testing.T) {
	db := testplatform.OpenPostgres(t, nil)

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
