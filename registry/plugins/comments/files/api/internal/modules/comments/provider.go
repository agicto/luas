package comments

import (
	"github.com/google/wire"

	"github.com/fastagi/fastbuild/api/internal/domain"
	"github.com/fastagi/fastbuild/api/internal/infra/config"
	"github.com/fastagi/fastbuild/api/internal/starter/assembly"
)

// ProviderSet wires the comments starter.
var ProviderSet = wire.NewSet(
	NewRepository,
	wire.Bind(new(store), new(*repository)),
	ProvideResourceCatalog,
	ProvideMemberDirectory,
	ProvideMentionPublisher,
	NewService,
	wire.Bind(new(Service), new(*service)),
	NewHandler,
)

// ProvideMemberDirectory adapts the organization starter's membership lookup.
func ProvideMemberDirectory(organizations domain.OrganizationRepository) memberDirectory {
	return organizations
}

// ProvideMentionPublisher returns the notification starter's publisher when that starter is
// selected, and nil otherwise, so mentions are stored without notifications.
func ProvideMentionPublisher(cfg *config.Config, publisher domain.NotificationPublisher) mentionPublisher {
	if cfg == nil || publisher == nil || !cfg.Starters.Selected(config.StarterNotification) {
		return nil
	}
	return publisher
}

// NewStarterManifest describes the comments routes, owner dependencies, and persistence.
func NewStarterManifest(handler *Handler) assembly.StarterManifest {
	return assembly.NewStaticStarterManifest(
		config.StarterComments,
		assembly.WithStarterSummary("Threaded comments on organization resources with @mentions."),
		assembly.WithStarterDependencies(config.StarterUser, config.StarterAudit, config.StarterOrganization),
		assembly.WithStarterModule(handler),
		assembly.WithStarterMigrationNames("2026_10_11_010000_create_comments_table"),
	)
}
