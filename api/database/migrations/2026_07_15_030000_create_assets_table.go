package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
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
	return execStatements(db,
		`CREATE TABLE assets (
		    id varchar(36) NOT NULL,
		    user_id bigint NOT NULL,
		    idempotency_key varchar(128) NOT NULL,
		    request_hash varchar(64) NOT NULL,
		    original_name varchar(255) NOT NULL,
		    media_type varchar(100) NOT NULL,
		    size_bytes bigint NOT NULL,
		    status varchar(24) NOT NULL,
		    staging_key varchar(255) NOT NULL,
		    object_key varchar(255) NOT NULL,
		    checksum_sha256 varchar(64) DEFAULT ''::varchar NOT NULL,
		    rejection_code varchar(64) DEFAULT ''::varchar NOT NULL,
		    pending_expires_at timestamptz NOT NULL,
		    ready_at timestamptz,
		    deleted_at timestamptz,
		    operation_kind varchar(24) DEFAULT ''::varchar NOT NULL,
		    operation_token varchar(64) DEFAULT ''::varchar NOT NULL,
		    operation_until timestamptz,
		    created_at timestamptz NOT NULL,
		    updated_at timestamptz NOT NULL,
		    CONSTRAINT assets_status_check CHECK ((status IN ('pending', 'ready', 'rejected', 'deleted'))),
		    CONSTRAINT assets_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_assets_cleanup ON assets (status, pending_expires_at)`,
		`CREATE INDEX idx_assets_operation_until ON assets (operation_until)`,
		`CREATE INDEX idx_assets_user_deleted_created ON assets (user_id, deleted_at, created_at DESC)`,
		`CREATE UNIQUE INDEX idx_assets_user_idempotency ON assets (user_id, idempotency_key)`,
		`CREATE INDEX idx_assets_user_status_deleted_created ON assets (user_id, status, deleted_at, created_at DESC)`,
		`ALTER TABLE assets ADD CONSTRAINT fk_assets_user FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE RESTRICT`,
	)
}

// Down removes asset metadata. Operators must remove provider objects before rollback.
func (m *createAssetsTable) Down(db *gorm.DB) error {
	return execStatements(db,
		`DROP TABLE IF EXISTS assets CASCADE`,
	)
}
