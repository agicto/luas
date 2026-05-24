package migrations

import (
	"github.com/zgiai/zgo/internal/infra/migration"
	"github.com/zgiai/zgo/internal/modules/permission"
	"gorm.io/gorm"
)

func init() {
	register("2025_12_26_000001_create_permissions_table", &createPermissionsTable{})
}

// createPermissionsTable creates the permissions table.
type createPermissionsTable struct {
	migration.BaseMigration
}

// Up applies the migration.
func (m *createPermissionsTable) Up(db *gorm.DB) error {
	return db.AutoMigrate(&permission.PermissionPO{})
}

// Down reverts the migration.
func (m *createPermissionsTable) Down(db *gorm.DB) error {
	return db.Migrator().DropTable("permissions")
}
