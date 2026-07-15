package audit

import (
	"github.com/google/wire"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/starter/assembly"
)

// ProviderSet wires the audit starter.
var ProviderSet = wire.NewSet(
	NewRepository,
	wire.Bind(new(domain.AuditLogRepository), new(*repository)),
	NewService,
	wire.Bind(new(Service), new(*service)),
	wire.Bind(new(domain.AuditLogRecorder), new(*service)),
	NewHandler,
)

// NewStarterManifest describes how the audit starter participates in the default scaffold.
func NewStarterManifest(handler *Handler) assembly.StarterManifest {
	return assembly.NewStaticStarterManifest(
		"audit",
		assembly.WithStarterModule(handler),
		assembly.WithStarterMigrationNames("2026_04_26_000000_create_audit_logs_table"),
		assembly.WithStarterMigrationNames("2026_04_27_000002_add_business_fields_to_audit_logs"),
	)
}
