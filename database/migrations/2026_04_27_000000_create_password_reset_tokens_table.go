package migrations

import (
	"github.com/zgiai/zgo/internal/infra/migration"
	"github.com/zgiai/zgo/internal/modules/user"
	"gorm.io/gorm"
)

func init() {
	register("2026_04_27_000000_create_password_reset_tokens_table", &createPasswordResetTokensTable{})
}

// createPasswordResetTokensTable creates the password reset token table.
type createPasswordResetTokensTable struct {
	migration.BaseMigration
}

// Up applies the migration.
func (m *createPasswordResetTokensTable) Up(db *gorm.DB) error {
	return db.AutoMigrate(&user.PasswordResetTokenPO{})
}

// Down reverts the migration.
func (m *createPasswordResetTokensTable) Down(db *gorm.DB) error {
	return db.Migrator().DropTable("password_reset_tokens")
}
