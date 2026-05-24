package migrations

import (
	"github.com/zgiai/zgo/internal/infra/migration"
	"github.com/zgiai/zgo/internal/modules/apikey"
	"gorm.io/gorm"
)

func init() {
	register("2026_04_06_000000_create_api_keys_table", &createAPIKeysTable{})
}

// createAPIKeysTable creates the api_keys table.
type createAPIKeysTable struct {
	migration.BaseMigration
}

// Up applies the migration.
func (m *createAPIKeysTable) Up(db *gorm.DB) error {
	return db.AutoMigrate(&apikey.APIKeyPO{})
}

// Down reverts the migration.
func (m *createAPIKeysTable) Down(db *gorm.DB) error {
	return db.Migrator().DropTable("api_keys")
}
