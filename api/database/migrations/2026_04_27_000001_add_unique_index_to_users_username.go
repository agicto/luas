package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2026_04_27_000001_add_unique_index_to_users_username", &addUniqueIndexToUsersUsername{})
}

// addUniqueIndexToUsersUsername ensures usernames are globally unique for login semantics.
type addUniqueIndexToUsersUsername struct {
	migration.BaseMigration
}

// Up adds the index to installations whose users table predates it; fresh tables already have it.
func (m *addUniqueIndexToUsersUsername) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username)`,
	)
}

// Down reverts the migration.
func (m *addUniqueIndexToUsersUsername) Down(db *gorm.DB) error {
	return execStatements(db,
		`DROP INDEX IF EXISTS idx_users_username`,
	)
}
