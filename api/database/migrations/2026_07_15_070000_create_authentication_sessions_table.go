package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/modules/user"
)

func init() {
	register("2026_07_15_070000_create_authentication_sessions_table", &createAuthenticationSessionsTable{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createAuthenticationSessionsTable struct {
	migration.BaseMigration
}

// Up creates the hash-only server-side user session authority.
func (m *createAuthenticationSessionsTable) Up(db *gorm.DB) error {
	return db.AutoMigrate(&user.AuthenticationSessionPO{})
}

// Down removes user authentication session state without touching accounts.
func (m *createAuthenticationSessionsTable) Down(db *gorm.DB) error {
	return db.Migrator().DropTable(&user.AuthenticationSessionPO{})
}
