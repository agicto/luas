package audit

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRepositoryPruneBeforePostgres(t *testing.T) {
	dsn := os.Getenv("LUAS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("LUAS_TEST_POSTGRES_DSN is required for PostgreSQL repository evidence")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })

	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { tx.Rollback() })
	require.NoError(t, tx.Exec(`
		CREATE TEMP TABLE audit_logs (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL
		) ON COMMIT DROP
	`).Error)
	require.NoError(t, tx.Exec(`
		CREATE INDEX idx_audit_logs_created_id ON audit_logs (created_at, id)
	`).Error)

	cutoff := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	for _, createdAt := range []time.Time{
		cutoff.Add(-72 * time.Hour),
		cutoff.Add(-48 * time.Hour),
		cutoff.Add(-24 * time.Hour),
		cutoff.Add(24 * time.Hour),
	} {
		require.NoError(t, tx.Exec("INSERT INTO audit_logs (created_at) VALUES (?)", createdAt).Error)
	}

	repository := NewRepository(tx)
	count, err := repository.PruneBefore(context.Background(), cutoff, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	var remainingOldIDs []uint
	require.NoError(t, tx.Raw(
		"SELECT id FROM audit_logs WHERE created_at < ? ORDER BY created_at, id",
		cutoff,
	).Scan(&remainingOldIDs).Error)
	assert.Equal(t, []uint{3}, remainingOldIDs)

	count, err = repository.PruneBefore(context.Background(), cutoff, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	var total int64
	require.NoError(t, tx.Table("audit_logs").Count(&total).Error)
	assert.Equal(t, int64(1), total)
}

func TestRepositoryPruneBeforePostgresSkipsLockedRows(t *testing.T) {
	dsn := os.Getenv("LUAS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("LUAS_TEST_POSTGRES_DSN is required for PostgreSQL repository evidence")
	}

	schema := fmt.Sprintf("audit_prune_%d", time.Now().UnixNano())
	admin := openPostgresTestDB(t, dsn)
	require.NoError(t, admin.Exec("CREATE SCHEMA "+schema).Error)
	t.Cleanup(func() { admin.Exec("DROP SCHEMA IF EXISTS " + schema + " CASCADE") })

	schemaDSN := dsn + "&search_path=" + schema
	if !strings.Contains(dsn, "?") {
		schemaDSN = dsn + "?search_path=" + schema
	}
	lockerDB := openPostgresTestDB(t, schemaDSN)
	prunerDB := openPostgresTestDB(t, schemaDSN)
	require.NoError(t, lockerDB.Exec(`
		CREATE TABLE audit_logs (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL
		)
	`).Error)

	cutoff := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	for _, createdAt := range []time.Time{cutoff.Add(-48 * time.Hour), cutoff.Add(-24 * time.Hour)} {
		require.NoError(t, lockerDB.Exec("INSERT INTO audit_logs (created_at) VALUES (?)", createdAt).Error)
	}

	locker := lockerDB.Begin()
	require.NoError(t, locker.Error)
	t.Cleanup(func() { locker.Rollback() })
	var lockedID uint
	require.NoError(t, locker.Raw(`
		SELECT id FROM audit_logs ORDER BY created_at, id LIMIT 1 FOR UPDATE
	`).Scan(&lockedID).Error)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	count, err := NewRepository(prunerDB).PruneBefore(ctx, cutoff, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	var remainingIDs []uint
	require.NoError(t, locker.Raw("SELECT id FROM audit_logs ORDER BY id").Scan(&remainingIDs).Error)
	assert.Equal(t, []uint{lockedID}, remainingIDs)
}

func openPostgresTestDB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })
	return db
}
