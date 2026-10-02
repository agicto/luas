package webhook

import (
	"github.com/google/wire"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/starter/assembly"
)

// ProviderSet wires the optional durable outbound webhook starter.
var ProviderSet = wire.NewSet(
	NewDefaultCatalog,
	NewRepository,
	wire.Bind(new(webhookStore), new(*repository)),
	NewSecretProtector,
	NewTargetPolicy,
	NewSender,
	NewService,
	wire.Bind(new(Service), new(*service)),
	wire.Bind(new(domain.WebhookPublisher), new(*service)),
	wire.Bind(new(domain.WebhookDispatcher), new(*service)),
	wire.Bind(new(domain.WebhookTester), new(*service)),
	wire.Bind(new(domain.WebhookMaintainer), new(*service)),
	NewHandler,
	NewOperatorHandler,
)

// NewStarterManifest describes webhook dependencies, routes, and persistence ownership.
func NewStarterManifest(handler *Handler) assembly.StarterManifest {
	return assembly.NewStaticStarterManifest(
		config.StarterWebhook,
		assembly.WithStarterDependencies(config.StarterUser, config.StarterAudit, config.StarterOrganization),
		assembly.WithStarterModule(handler),
		assembly.WithStarterMigrationNames("2026_07_15_060000_create_webhook_tables"),
	)
}
