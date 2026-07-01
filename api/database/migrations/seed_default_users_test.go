package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/zgiai/luas/api/internal/modules/user"
)

func TestSeedDefaultUsersCreatesLoginableAdmin(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&user.UserPO{}))

	migration := &seedDefaultUsers{}
	require.NoError(t, migration.Up(db))

	var admin user.UserPO
	require.NoError(t, db.Where("username = ?", "admin").First(&admin).Error)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte("secret")))
}
