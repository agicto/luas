package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2026_04_06_000000_create_api_keys_table", &createAPIKeysTable{})
}

type createAPIKeysTable struct {
	migration.BaseMigration
}

// Up creates the api_keys table.
func (m *createAPIKeysTable) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE TABLE api_keys (
		    id bigserial NOT NULL,
		    created_at timestamptz,
		    updated_at timestamptz,
		    deleted_at timestamptz,
		    user_id bigint NOT NULL,
		    name varchar(100) NOT NULL,
		    key_prefix varchar(32) NOT NULL,
		    key_hash varchar(64) NOT NULL,
		    scopes text,
		    last_used_at timestamptz,
		    expires_at timestamptz,
		    revoked_at timestamptz,
		    CONSTRAINT api_keys_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_api_keys_deleted_at ON api_keys (deleted_at)`,
		`CREATE UNIQUE INDEX idx_api_keys_key_hash ON api_keys (key_hash)`,
		`CREATE INDEX idx_api_keys_key_prefix ON api_keys (key_prefix)`,
		`CREATE INDEX idx_api_keys_user_id ON api_keys (user_id)`,
	)
}

// Down drops the api_keys table.
func (m *createAPIKeysTable) Down(db *gorm.DB) error {
	return execStatements(db,
		`DROP TABLE IF EXISTS api_keys CASCADE`,
	)
}
