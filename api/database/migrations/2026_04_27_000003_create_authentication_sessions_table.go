package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2026_04_27_000003_create_authentication_sessions_table", &createAuthenticationSessionsTable{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createAuthenticationSessionsTable struct {
	migration.BaseMigration
}

// Up creates the hash-only server-side user session authority.
func (m *createAuthenticationSessionsTable) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE TABLE authentication_sessions (
		    id varchar(32) NOT NULL,
		    created_at timestamptz,
		    updated_at timestamptz,
		    user_id bigint NOT NULL,
		    token_hash varchar(64) NOT NULL,
		    expires_at timestamptz NOT NULL,
		    idle_expires_at timestamptz NOT NULL,
		    last_seen_at timestamptz NOT NULL,
		    revoked_at timestamptz,
		    revocation_reason varchar(32) DEFAULT ''::varchar NOT NULL,
		    CONSTRAINT authentication_sessions_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_authentication_sessions_expires_at ON authentication_sessions (expires_at)`,
		`CREATE INDEX idx_authentication_sessions_idle_expires_at ON authentication_sessions (idle_expires_at)`,
		`CREATE INDEX idx_authentication_sessions_revoked_at ON authentication_sessions (revoked_at)`,
		`CREATE UNIQUE INDEX idx_authentication_sessions_token_hash ON authentication_sessions (token_hash)`,
		`CREATE INDEX idx_authentication_sessions_user_id ON authentication_sessions (user_id)`,
	)
}

// Down removes user authentication session state without touching accounts.
func (m *createAuthenticationSessionsTable) Down(db *gorm.DB) error {
	return execStatements(db,
		`DROP TABLE IF EXISTS authentication_sessions CASCADE`,
	)
}
