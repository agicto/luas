package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
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
	return execStatements(db,
		`CREATE TABLE notifications (
		    id bigserial NOT NULL,
		    user_id bigint NOT NULL,
		    idempotency_key varchar(128) NOT NULL,
		    publication_hash varchar(64) NOT NULL,
		    kind varchar(100) NOT NULL,
		    title varchar(640) NOT NULL,
		    body text NOT NULL,
		    action_url varchar(2048) DEFAULT ''::varchar NOT NULL,
		    read_at timestamptz,
		    created_at timestamptz NOT NULL,
		    updated_at timestamptz NOT NULL,
		    CONSTRAINT notifications_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_notifications_kind ON notifications (kind)`,
		`CREATE INDEX idx_notifications_user_created ON notifications (user_id, created_at DESC)`,
		`CREATE UNIQUE INDEX idx_notifications_user_idempotency ON notifications (user_id, idempotency_key)`,
		`CREATE INDEX idx_notifications_user_read ON notifications (user_id, read_at)`,
		`CREATE TABLE notification_deliveries (
		    id bigserial NOT NULL,
		    notification_id bigint NOT NULL,
		    channel varchar(24) NOT NULL,
		    status varchar(24) NOT NULL,
		    attempts smallint DEFAULT 0 NOT NULL,
		    available_at timestamptz NOT NULL,
		    lease_token varchar(64) DEFAULT ''::varchar NOT NULL,
		    lease_expires_at timestamptz,
		    destination_hash varchar(64) DEFAULT ''::varchar NOT NULL,
		    delivered_at timestamptz,
		    last_failure_code varchar(64) DEFAULT ''::varchar NOT NULL,
		    created_at timestamptz NOT NULL,
		    updated_at timestamptz NOT NULL,
		    CONSTRAINT notification_deliveries_channel_check CHECK ((channel IN ('in_app', 'email'))),
		    CONSTRAINT notification_deliveries_status_check CHECK ((status IN ('pending', 'processing', 'delivered', 'failed'))),
		    CONSTRAINT notification_deliveries_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_notification_deliveries_leased ON notification_deliveries (channel, status, lease_expires_at)`,
		`CREATE UNIQUE INDEX idx_notification_deliveries_notification_channel ON notification_deliveries (notification_id, channel)`,
		`CREATE INDEX idx_notification_deliveries_notification_id ON notification_deliveries (notification_id)`,
		`CREATE INDEX idx_notification_deliveries_pending ON notification_deliveries (channel, status, available_at)`,
		`CREATE TABLE notification_preferences (
		    user_id bigserial NOT NULL,
		    in_app_enabled boolean NOT NULL,
		    email_enabled boolean NOT NULL,
		    created_at timestamptz NOT NULL,
		    updated_at timestamptz NOT NULL,
		    CONSTRAINT notification_preferences_pkey PRIMARY KEY (user_id)
		)`,
		`ALTER TABLE notifications ADD CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE notification_deliveries ADD CONSTRAINT fk_notifications_deliveries FOREIGN KEY (notification_id) REFERENCES notifications(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE notification_preferences ADD CONSTRAINT fk_notification_preferences_user FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE`,
	)
}

// Down removes channel-owned rows before their notification parents.
func (m *createNotificationTables) Down(db *gorm.DB) error {
	return execStatements(db,
		`DROP TABLE IF EXISTS notification_preferences CASCADE`,
		`DROP TABLE IF EXISTS notification_deliveries CASCADE`,
		`DROP TABLE IF EXISTS notifications CASCADE`,
	)
}
