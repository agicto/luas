package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/modules/setting"
)

func init() {
	register("2026_07_15_040000_create_settings_table", &createSettingsTable{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createSettingsTable struct {
	migration.BaseMigration
}

// Up creates typed setting override history after user and organization ownership tables.
func (m *createSettingsTable) Up(db *gorm.DB) error {
	return db.AutoMigrate(&setting.SettingPO{})
}

// Down removes setting overrides and reset tombstones.
func (m *createSettingsTable) Down(db *gorm.DB) error {
	return db.Migrator().DropTable(&setting.SettingPO{})
}
