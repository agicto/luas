package migrations

import (
	"github.com/zgiai/zgo/internal/infra/migration"
	"github.com/zgiai/zgo/internal/modules/permission"
	"gorm.io/gorm"
)

func init() {
	register("2025_12_26_000004_seed_default_roles", &seedDefaultRoles{})
}

// seedDefaultRoles seeds the default roles.
type seedDefaultRoles struct {
	migration.BaseMigration
}

// Up applies the migration.
func (m *seedDefaultRoles) Up(db *gorm.DB) error {
	roles := []permission.RolePO{
		{Name: "admin", DisplayName: "Administrator", Description: "Full access to all resources", IsDefault: false},
		{Name: "user", DisplayName: "User", Description: "Standard user access", IsDefault: true},
		{Name: "guest", DisplayName: "Guest", Description: "Read-only access", IsDefault: false},
	}
	for _, role := range roles {
		if err := db.FirstOrCreate(&role, permission.RolePO{Name: role.Name}).Error; err != nil {
			return err
		}
	}
	return nil
}

// Down reverts the migration.
func (m *seedDefaultRoles) Down(db *gorm.DB) error {
	return db.Where("name IN ?", []string{"admin", "user", "guest"}).Delete(&permission.RolePO{}).Error
}
