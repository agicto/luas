package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2026_07_25_000000_add_audit_retention_index", &addAuditRetentionIndex{
		BaseMigration: migration.BaseMigration{UseTransaction: false},
	})
}

type addAuditRetentionIndex struct {
	migration.BaseMigration
}

// Up adds the ordered cutoff index used by bounded PostgreSQL retention batches.
func (m *addAuditRetentionIndex) Up(db *gorm.DB) error {
	if db.Name() != "postgres" {
		return nil
	}
	return db.Exec(`
		CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_created_id
		ON audit_logs (created_at, id)
	`).Error
}

// Down removes only the retention-specific index.
func (m *addAuditRetentionIndex) Down(db *gorm.DB) error {
	if db.Name() != "postgres" {
		return nil
	}
	return db.Exec(`DROP INDEX CONCURRENTLY IF EXISTS idx_audit_logs_created_id`).Error
}
