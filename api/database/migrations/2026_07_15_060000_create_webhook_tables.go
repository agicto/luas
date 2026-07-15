package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/modules/webhook"
)

func init() {
	register("2026_07_15_060000_create_webhook_tables", &createWebhookTables{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createWebhookTables struct {
	migration.BaseMigration
}

// Up creates endpoint custody, durable events, deliveries, and minimized attempts.
func (m *createWebhookTables) Up(db *gorm.DB) error {
	return db.AutoMigrate(
		&webhook.EndpointPO{},
		&webhook.SubscriptionPO{},
		&webhook.EventPO{},
		&webhook.DeliveryPO{},
		&webhook.AttemptPO{},
	)
}

// Down removes attempt and delivery history before endpoint and event owners.
func (m *createWebhookTables) Down(db *gorm.DB) error {
	return db.Migrator().DropTable(
		&webhook.AttemptPO{},
		&webhook.DeliveryPO{},
		&webhook.EventPO{},
		&webhook.SubscriptionPO{},
		&webhook.EndpointPO{},
	)
}
