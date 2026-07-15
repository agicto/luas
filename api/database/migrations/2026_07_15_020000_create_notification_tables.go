package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/modules/notification"
)

func init() {
	register("2026_07_15_020000_create_notification_tables", &createNotificationTables{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createNotificationTables struct {
	migration.BaseMigration
}

// Up creates notification events before their channel deliveries and user preferences.
func (m *createNotificationTables) Up(db *gorm.DB) error {
	return db.AutoMigrate(
		&notification.NotificationPO{},
		&notification.NotificationDeliveryPO{},
		&notification.NotificationPreferencePO{},
	)
}

// Down removes channel-owned rows before their notification parents.
func (m *createNotificationTables) Down(db *gorm.DB) error {
	if err := db.Migrator().DropTable(&notification.NotificationPreferencePO{}); err != nil {
		return err
	}
	if err := db.Migrator().DropTable(&notification.NotificationDeliveryPO{}); err != nil {
		return err
	}
	return db.Migrator().DropTable(&notification.NotificationPO{})
}
