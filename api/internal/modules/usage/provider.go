package usage

import (
	"github.com/google/wire"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/starter/assembly"
)

// ProviderSet wires the optional usage accounting and quota starter.
var ProviderSet = wire.NewSet(
	NewDefaultCatalog,
	NewRepository,
	wire.Bind(new(usageStore), new(*repository)),
	NewService,
	wire.Bind(new(Service), new(*service)),
	wire.Bind(new(domain.UsageReader), new(*service)),
	wire.Bind(new(domain.UsageRecorder), new(*service)),
	wire.Bind(new(domain.UsageConsumer), new(*service)),
	wire.Bind(new(domain.UsageQuotaWriter), new(*service)),
	wire.Bind(new(domain.UsageMaintainer), new(*service)),
	NewHandler,
)

// NewStarterManifest describes usage routes, owner dependencies, and persistence.
func NewStarterManifest(handler *Handler) assembly.StarterManifest {
	return assembly.NewStaticStarterManifest(
		config.StarterUsage,
		assembly.WithStarterSummary("Trusted idempotent metering, atomic quota decisions, and private summaries."),
		assembly.WithStarterDependencies(config.StarterUser, config.StarterAudit, config.StarterOrganization),
		assembly.WithStarterModule(handler),
		assembly.WithStarterMigrationNames("2026_07_15_050000_create_usage_tables"),
	)
}
