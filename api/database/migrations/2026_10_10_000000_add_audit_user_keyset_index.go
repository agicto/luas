package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2026_10_10_000000_add_audit_user_keyset_index", &addAuditUserKeysetIndex{
		BaseMigration: migration.BaseMigration{UseTransaction: false},
	})
}

type addAuditUserKeysetIndex struct {
	migration.BaseMigration
}

// Up adds the (user_id, id) index behind keyset pages of one user's history and drops the
// single-column user_id index it makes redundant, so each audit insert maintains one index fewer.
// Both statements run concurrently and never block audit writes on a large table.
func (m *addAuditUserKeysetIndex) Up(db *gorm.DB) error {
	if err := db.Exec(`
		CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_user_id_id
		ON audit_logs (user_id, id)
	`).Error; err != nil {
		return err
	}
	return db.Exec(`DROP INDEX CONCURRENTLY IF EXISTS idx_audit_logs_user_id`).Error
}

// Down restores the single-column index before dropping the composite one.
func (m *addAuditUserKeysetIndex) Down(db *gorm.DB) error {
	if err := db.Exec(`
		CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_user_id
		ON audit_logs (user_id)
	`).Error; err != nil {
		return err
	}
	return db.Exec(`DROP INDEX CONCURRENTLY IF EXISTS idx_audit_logs_user_id_id`).Error
}
