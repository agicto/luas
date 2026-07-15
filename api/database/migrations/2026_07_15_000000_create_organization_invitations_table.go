package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/modules/organization"
)

func init() {
	register("2026_07_15_000000_create_organization_invitations_table", &createOrganizationInvitationsTable{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createOrganizationInvitationsTable struct {
	migration.BaseMigration
}

// Up adds the invitation lifecycle without changing the deployed ownership tables.
func (m *createOrganizationInvitationsTable) Up(db *gorm.DB) error {
	return db.AutoMigrate(&organization.OrganizationInvitationPO{})
}

// Down removes only the additive invitation table.
func (m *createOrganizationInvitationsTable) Down(db *gorm.DB) error {
	return db.Migrator().DropTable(&organization.OrganizationInvitationPO{})
}
