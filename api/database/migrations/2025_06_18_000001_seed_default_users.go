package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2025_06_18_000001_seed_default_users", &seedDefaultUsers{})
}

// seedDefaultUsers seeds the default admin user.
type seedDefaultUsers struct {
	migration.BaseMigration
}

// Up creates the admin account (password: secret) only when no active user exists.
func (m *seedDefaultUsers) Up(db *gorm.DB) error {
	return execStatements(db,
		`INSERT INTO users (created_at, updated_at, username, password, email, nickname, avatar, phone, bio, status)
		SELECT now(), now(), 'admin', '$2a$10$OkAgF/Pm/v3pdzkUhKJEeOhehkbTRZar9Rk3X2nEjCcrlsluiTnay',
		       'admin@example.com', 'Admin User', '', '', '', 1
		WHERE NOT EXISTS (SELECT 1 FROM users WHERE deleted_at IS NULL)`,
	)
}

// Down soft-deletes the seeded admin account, matching the account lifecycle.
func (m *seedDefaultUsers) Down(db *gorm.DB) error {
	return execStatements(db,
		`UPDATE users SET deleted_at = now() WHERE username = 'admin' AND deleted_at IS NULL`,
	)
}
