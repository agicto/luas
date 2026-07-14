package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/modules/organization"
)

func init() {
	register("2026_07_14_000000_create_organizations_tables", &createOrganizationsTables{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createOrganizationsTables struct {
	migration.BaseMigration
}

// Up creates organizations before their memberships.
func (m *createOrganizationsTables) Up(db *gorm.DB) error {
	return db.AutoMigrate(
		&organization.OrganizationPO{},
		&organization.OrganizationMembershipPO{},
	)
}

// Down removes memberships before their parent organizations.
func (m *createOrganizationsTables) Down(db *gorm.DB) error {
	if err := db.Migrator().DropTable(&organization.OrganizationMembershipPO{}); err != nil {
		return err
	}
	return db.Migrator().DropTable(&organization.OrganizationPO{})
}
