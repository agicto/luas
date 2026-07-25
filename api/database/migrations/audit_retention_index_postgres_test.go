package migrations

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestAddAuditRetentionIndexPostgres(t *testing.T) {
	dsn := os.Getenv("LUAS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("LUAS_TEST_POSTGRES_DSN is required for PostgreSQL migration evidence")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })

	schema := fmt.Sprintf("audit_retention_%d", time.Now().UnixNano())
	require.NoError(t, db.Exec("CREATE SCHEMA "+schema).Error)
	t.Cleanup(func() { db.Exec("DROP SCHEMA IF EXISTS " + schema + " CASCADE") })
	require.NoError(t, db.Exec("SET search_path TO "+schema).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE audit_logs (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL
		)
	`).Error)

	migration := &addAuditRetentionIndex{}
	assert.False(t, migration.WithinTransaction())
	require.NoError(t, migration.Up(db))
	assert.True(t, db.Migrator().HasIndex("audit_logs", "idx_audit_logs_created_id"))
	require.NoError(t, migration.Down(db))
	assert.False(t, db.Migrator().HasIndex("audit_logs", "idx_audit_logs_created_id"))
	require.NoError(t, migration.Up(db))
	assert.True(t, db.Migrator().HasIndex("audit_logs", "idx_audit_logs_created_id"))
}
