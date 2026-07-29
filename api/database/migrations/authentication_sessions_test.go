package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testplatform "github.com/zgiai/luas/api/internal/infra/testing"
	"github.com/zgiai/luas/api/internal/modules/user"
)

func TestCreateAuthenticationSessionsTableUpAndDown(t *testing.T) {
	db := testplatform.OpenPostgres(t, nil)
	migration := &createAuthenticationSessionsTable{}

	require.NoError(t, migration.Up(db))
	assert.True(t, db.Migrator().HasTable(&user.AuthenticationSessionPO{}))
	for _, index := range []string{
		"idx_authentication_sessions_user_id",
		"idx_authentication_sessions_token_hash",
		"idx_authentication_sessions_expires_at",
		"idx_authentication_sessions_idle_expires_at",
		"idx_authentication_sessions_revoked_at",
	} {
		assert.True(t, db.Migrator().HasIndex(&user.AuthenticationSessionPO{}, index), index)
	}

	columns, err := db.Migrator().ColumnTypes(&user.AuthenticationSessionPO{})
	require.NoError(t, err)
	names := make([]string, 0, len(columns))
	for _, column := range columns {
		names = append(names, column.Name())
	}
	assert.Contains(t, names, "token_hash")
	assert.NotContains(t, names, "access_token")
	assert.NotContains(t, names, "credential")
	assert.NotContains(t, names, "ip_address")
	assert.NotContains(t, names, "user_agent")

	require.NoError(t, migration.Down(db))
	assert.False(t, db.Migrator().HasTable(&user.AuthenticationSessionPO{}))
}
