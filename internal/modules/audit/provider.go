package audit

import (
	"github.com/google/wire"
	"github.com/zgiai/zgo/internal/contracts"
	"github.com/zgiai/zgo/internal/domain"
)

// ProviderSet wires the audit starter.
var ProviderSet = wire.NewSet(
	NewRepository,
	wire.Bind(new(domain.AuditLogRepository), new(*repository)),
	NewService,
	wire.Bind(new(Service), new(*service)),
	NewHandler,
)

// NewStarterManifest describes how the audit starter participates in the default scaffold.
func NewStarterManifest(handler *Handler) contracts.StarterManifest {
	return contracts.NewStaticStarterManifest(
		"audit",
		contracts.WithStarterModule(handler),
		contracts.WithStarterMigrationNames("2026_04_26_000000_create_audit_logs_table"),
		contracts.WithStarterMigrationNames("2026_04_27_000002_add_business_fields_to_audit_logs"),
	)
}
