package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2026_10_04_000000_add_audience_to_authentication_sessions", &addAudienceToAuthenticationSessions{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type addAudienceToAuthenticationSessions struct {
	migration.BaseMigration
}

// Up records which sign-in surface issued each session. Existing sessions become public-login
// sessions, so operators sign in to the console again once after this migration.
func (m *addAudienceToAuthenticationSessions) Up(db *gorm.DB) error {
	return execStatements(db,
		`ALTER TABLE authentication_sessions ADD COLUMN audience varchar(16) DEFAULT 'user'::varchar NOT NULL`,
		`ALTER TABLE authentication_sessions ADD CONSTRAINT chk_authentication_sessions_audience
		    CHECK (audience IN ('user', 'operator'))`,
	)
}

// Down removes the audience and with it the separation between public and operator sessions.
func (m *addAudienceToAuthenticationSessions) Down(db *gorm.DB) error {
	return execStatements(db,
		`ALTER TABLE authentication_sessions DROP CONSTRAINT IF EXISTS chk_authentication_sessions_audience`,
		`ALTER TABLE authentication_sessions DROP COLUMN IF EXISTS audience`,
	)
}
