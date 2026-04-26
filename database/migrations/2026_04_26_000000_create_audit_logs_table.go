package migrations

import (
	"github.com/zgiai/zgo/internal/infra/migration"
	"github.com/zgiai/zgo/internal/modules/audit"
	"gorm.io/gorm"
)

func init() {
	register("2026_04_26_000000_create_audit_logs_table", &createAuditLogsTable{})
}

// createAuditLogsTable creates the audit_logs table.
type createAuditLogsTable struct {
	migration.BaseMigration
}

func (m *createAuditLogsTable) Up(db *gorm.DB) error {
	return db.AutoMigrate(&audit.AuditLogPO{})
}

func (m *createAuditLogsTable) Down(db *gorm.DB) error {
	return db.Migrator().DropTable("audit_logs")
}
