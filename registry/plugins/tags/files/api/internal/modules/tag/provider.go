package tag

import (
	"github.com/google/wire"

	"github.com/fastagi/fastbuild/api/internal/infra/config"
	"github.com/fastagi/fastbuild/api/internal/starter/assembly"
)

// ProviderSet wires the tag starter. It provides *ResourceTypes, which modules take in their own
// constructors to register the resource types they let organizations tag.
var ProviderSet = wire.NewSet(
	NewResourceTypes,
	NewRepository,
	wire.Bind(new(store), new(*repository)),
	NewService,
	wire.Bind(new(Service), new(*service)),
	NewHandler,
)

// NewStarterManifest describes the tag routes, owner dependencies, and persistence.
func NewStarterManifest(handler *Handler) assembly.StarterManifest {
	return assembly.NewStaticStarterManifest(
		config.StarterTag,
		assembly.WithStarterSummary("Organization tags that resources of registered types can carry."),
		assembly.WithStarterDependencies(config.StarterUser, config.StarterAudit, config.StarterOrganization),
		assembly.WithStarterModule(handler),
		assembly.WithStarterMigrationNames("2026_10_11_000100_create_tags_tables"),
	)
}
