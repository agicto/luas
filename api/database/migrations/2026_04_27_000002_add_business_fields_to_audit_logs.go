package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2026_04_27_000002_add_business_fields_to_audit_logs", &addBusinessFieldsToAuditLogs{})
}

// addBusinessFieldsToAuditLogs adds business-level audit columns to existing installations.
type addBusinessFieldsToAuditLogs struct {
	migration.BaseMigration
}

// Up adds the columns and their indexes when the table predates them; fresh tables already have them.
func (m *addBusinessFieldsToAuditLogs) Up(db *gorm.DB) error {
	return execStatements(db,
		`ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS target_type varchar(80)`,
		`ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS target_id varchar(120)`,
		`ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS result varchar(40)`,
		`ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS changes text`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_target_type ON audit_logs (target_type)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_target_id ON audit_logs (target_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_result ON audit_logs (result)`,
	)
}

// Down removes the business-level audit columns and their indexes.
func (m *addBusinessFieldsToAuditLogs) Down(db *gorm.DB) error {
	return execStatements(db,
		`ALTER TABLE audit_logs DROP COLUMN IF EXISTS target_type`,
		`ALTER TABLE audit_logs DROP COLUMN IF EXISTS target_id`,
		`ALTER TABLE audit_logs DROP COLUMN IF EXISTS result`,
		`ALTER TABLE audit_logs DROP COLUMN IF EXISTS changes`,
	)
}
