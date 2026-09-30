package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2026_10_01_000000_create_platform_operators_table", &createPlatformOperatorsTable{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createPlatformOperatorsTable struct {
	migration.BaseMigration
}

// Up creates operator grants keyed by user; deleting a user removes the grant.
func (m *createPlatformOperatorsTable) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE TABLE platform_operators (
		    user_id bigint NOT NULL,
		    granted_at timestamptz NOT NULL,
		    CONSTRAINT platform_operators_pkey PRIMARY KEY (user_id)
		)`,
		`ALTER TABLE platform_operators ADD CONSTRAINT fk_platform_operators_user FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE`,
	)
}

// Down removes operator grants.
func (m *createPlatformOperatorsTable) Down(db *gorm.DB) error {
	return execStatements(db, `DROP TABLE IF EXISTS platform_operators CASCADE`)
}
