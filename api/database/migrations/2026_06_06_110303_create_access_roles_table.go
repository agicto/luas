package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/modules/access"
)

func init() {
	register("2026_06_06_110303_create_access_roles_table", &createAccessRolesTable{})
}

// createAccessRolesTable creates the access_roles table.
type createAccessRolesTable struct {
	migration.BaseMigration
}

// Up applies the migration.
func (m *createAccessRolesTable) Up(db *gorm.DB) error {
	return db.AutoMigrate(&access.AccessRolePO{})
}

// Down reverts the migration.
func (m *createAccessRolesTable) Down(db *gorm.DB) error {
	return db.Migrator().DropTable("access_roles")
}
