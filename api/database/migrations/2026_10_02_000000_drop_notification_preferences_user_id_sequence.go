package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2026_10_02_000000_drop_notification_preferences_user_id_sequence", &dropNotificationPreferencesUserIDSequence{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type dropNotificationPreferencesUserIDSequence struct {
	migration.BaseMigration
}

// Up removes the sequence default from a key that is always the owning user's ID.
func (m *dropNotificationPreferencesUserIDSequence) Up(db *gorm.DB) error {
	return execStatements(db,
		`ALTER TABLE notification_preferences ALTER COLUMN user_id DROP DEFAULT`,
		`DROP SEQUENCE IF EXISTS notification_preferences_user_id_seq`,
	)
}

// Down restores the unused sequence default created by the original table definition.
func (m *dropNotificationPreferencesUserIDSequence) Down(db *gorm.DB) error {
	return execStatements(db,
		`CREATE SEQUENCE IF NOT EXISTS notification_preferences_user_id_seq OWNED BY notification_preferences.user_id`,
		`ALTER TABLE notification_preferences ALTER COLUMN user_id SET DEFAULT nextval('notification_preferences_user_id_seq'::regclass)`,
	)
}
