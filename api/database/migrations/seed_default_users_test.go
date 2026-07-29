package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	testplatform "github.com/zgiai/luas/api/internal/infra/testing"
	"github.com/zgiai/luas/api/internal/modules/user"
)

func TestSeedDefaultUsersCreatesLoginableAdmin(t *testing.T) {
	db := testplatform.OpenPostgres(t, nil, &user.UserPO{})

	migration := &seedDefaultUsers{}
	require.NoError(t, migration.Up(db))

	var admin user.UserPO
	require.NoError(t, db.Where("username = ?", "admin").First(&admin).Error)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte("secret")))
}
