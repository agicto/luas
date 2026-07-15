package webhook

import (
	"github.com/google/wire"

	"github.com/zgiai/luas/api/internal/domain"
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
)

// NewStarterManifest describes webhook dependencies, routes, and persistence ownership.
func NewStarterManifest(handler *Handler) assembly.StarterManifest {
	return assembly.NewStaticStarterManifest(
		"webhook",
		assembly.WithStarterDependencies("user", "audit", "organization"),
		assembly.WithStarterModule(handler),
		assembly.WithStarterMigrationNames("2026_07_15_060000_create_webhook_tables"),
	)
}
