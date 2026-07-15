package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/modules/usage"
)

func init() {
	register("2026_07_15_050000_create_usage_tables", &createUsageTables{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createUsageTables struct {
	migration.BaseMigration
}

// Up creates quota history, current counters, and minimized idempotency receipts.
func (m *createUsageTables) Up(db *gorm.DB) error {
	return db.AutoMigrate(
		&usage.UsageCounterPO{},
		&usage.UsageQuotaPO{},
		&usage.UsageEventPO{},
	)
}

// Down removes usage receipts before quota and counter ownership tables.
func (m *createUsageTables) Down(db *gorm.DB) error {
	return db.Migrator().DropTable(
		&usage.UsageEventPO{},
		&usage.UsageQuotaPO{},
		&usage.UsageCounterPO{},
	)
}
