package migrations

import (
	"github.com/zgiai/zgo/internal/infra/migration"
	"github.com/zgiai/zgo/internal/modules/user"
	"gorm.io/gorm"
)

func init() {
	register("2026_04_27_000001_add_unique_index_to_users_username", &addUniqueIndexToUsersUsername{})
}

// addUniqueIndexToUsersUsername ensures usernames are globally unique for login semantics.
type addUniqueIndexToUsersUsername struct {
	migration.BaseMigration
}

// Up applies the migration.
func (m *addUniqueIndexToUsersUsername) Up(db *gorm.DB) error {
	if db.Migrator().HasIndex(&user.UserPO{}, "Username") {
		return nil
	}
	return db.Migrator().CreateIndex(&user.UserPO{}, "Username")
}

// Down reverts the migration.
func (m *addUniqueIndexToUsersUsername) Down(db *gorm.DB) error {
	return db.Migrator().DropIndex(&user.UserPO{}, "Username")
}
