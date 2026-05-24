package migrations

import (
	"github.com/zgiai/zgo/internal/infra/migration"
	"github.com/zgiai/zgo/internal/modules/audit"
	"gorm.io/gorm"
)

func init() {
	register("2026_04_27_000002_add_business_fields_to_audit_logs", &addBusinessFieldsToAuditLogs{})
}

// addBusinessFieldsToAuditLogs adds business-level audit columns to existing installations.
type addBusinessFieldsToAuditLogs struct {
	migration.BaseMigration
}

func (m *addBusinessFieldsToAuditLogs) Up(db *gorm.DB) error {
	return db.AutoMigrate(&audit.AuditLogPO{})
}

func (m *addBusinessFieldsToAuditLogs) Down(db *gorm.DB) error {
	migrator := db.Migrator()
	for _, column := range []string{"target_type", "target_id", "result", "changes"} {
		if migrator.HasColumn("audit_logs", column) {
			if err := migrator.DropColumn("audit_logs", column); err != nil {
				return err
			}
		}
	}
	return nil
}
