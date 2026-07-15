package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/modules/asset"
)

func init() {
	register("2026_07_15_030000_create_assets_table", &createAssetsTable{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createAssetsTable struct {
	migration.BaseMigration
}

// Up creates private asset metadata after the default user table.
func (m *createAssetsTable) Up(db *gorm.DB) error {
	return db.AutoMigrate(&asset.AssetPO{})
}

// Down removes asset metadata. Operators must remove provider objects before rollback.
func (m *createAssetsTable) Down(db *gorm.DB) error {
	return db.Migrator().DropTable(&asset.AssetPO{})
}
