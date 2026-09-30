package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2026_04_27_000000_create_password_reset_tokens_table", &createPasswordResetTokensTable{})
}

type createPasswordResetTokensTable struct {
	migration.BaseMigration
}

// Up creates the password reset token table.
func (m *createPasswordResetTokensTable) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE TABLE password_reset_tokens (
		    id bigserial NOT NULL,
		    created_at timestamptz,
		    updated_at timestamptz,
		    user_id bigint NOT NULL,
		    token_hash varchar(64) NOT NULL,
		    expires_at timestamptz NOT NULL,
		    used_at timestamptz,
		    CONSTRAINT password_reset_tokens_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_password_reset_tokens_expires_at ON password_reset_tokens (expires_at)`,
		`CREATE UNIQUE INDEX idx_password_reset_tokens_token_hash ON password_reset_tokens (token_hash)`,
		`CREATE INDEX idx_password_reset_tokens_used_at ON password_reset_tokens (used_at)`,
		`CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens (user_id)`,
	)
}

// Down drops the password reset token table.
func (m *createPasswordResetTokensTable) Down(db *gorm.DB) error {
	return execStatements(db,
		`DROP TABLE IF EXISTS password_reset_tokens CASCADE`,
	)
}
