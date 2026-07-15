package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/modules/permission"
)

func init() {
	register("2026_07_15_010000_create_permission_tables", &createPermissionTables{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createPermissionTables struct {
	migration.BaseMigration
}

// Up creates roles before their grants and membership assignments.
func (m *createPermissionTables) Up(db *gorm.DB) error {
	return db.AutoMigrate(
		&permission.AccessRolePO{},
		&permission.AccessRolePermissionPO{},
		&permission.AccessRoleAssignmentPO{},
	)
}

// Down removes assignments and grants before their parent roles.
func (m *createPermissionTables) Down(db *gorm.DB) error {
	if err := db.Migrator().DropTable(&permission.AccessRoleAssignmentPO{}); err != nil {
		return err
	}
	if err := db.Migrator().DropTable(&permission.AccessRolePermissionPO{}); err != nil {
		return err
	}
	return db.Migrator().DropTable(&permission.AccessRolePO{})
}
