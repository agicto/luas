package notification

import (
	"github.com/google/wire"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/email"
	"github.com/zgiai/luas/api/internal/infra/events"
	"github.com/zgiai/luas/api/internal/starter/assembly"
)

// ProviderSet wires the optional notification starter and its internal publisher/dispatcher seams.
var ProviderSet = wire.NewSet(
	NewRepository,
	wire.Bind(new(notificationStore), new(*repository)),
	ProvideEmailSender,
	ProvideEventPublisher,
	NewService,
	wire.Bind(new(Service), new(*service)),
	wire.Bind(new(domain.NotificationPublisher), new(*service)),
	wire.Bind(new(domain.NotificationDispatcher), new(*service)),
	NewHandler,
	NewOperatorHandler,
)

// ProvideEmailSender adapts the shared email capability to the starter-owned seam.
func ProvideEmailSender(service *email.Service) emailSender { return service }

// ProvideEventPublisher adapts the shared event bus to the starter-owned seam.
func ProvideEventPublisher(bus *events.EventBus) eventPublisher { return bus }

// NewStarterManifest describes notification routes, dependencies, and persistence ownership.
func NewStarterManifest(handler *Handler) assembly.StarterManifest {
	return assembly.NewStaticStarterManifest(
		config.StarterNotification,
		assembly.WithStarterSummary("User-scoped in-app notifications and durable email delivery with preferences and leases."),
		assembly.WithStarterDependencies(config.StarterUser, config.StarterAudit),
		assembly.WithStarterModule(handler),
		assembly.WithStarterMigrationNames(
			"2026_07_15_020000_create_notification_tables",
			"2026_10_02_000000_drop_notification_preferences_user_id_sequence",
		),
	)
}
