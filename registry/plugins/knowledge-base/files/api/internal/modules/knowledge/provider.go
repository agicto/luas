package knowledge

import (
	"github.com/google/wire"

	"github.com/fastagi/fastbuild/api/internal/infra/config"
	"github.com/fastagi/fastbuild/api/internal/starter/assembly"
)

// ProviderSet wires the knowledge starter.
var ProviderSet = wire.NewSet(
	NewRepository,
	wire.Bind(new(store), new(*repository)),
	NewService,
	wire.Bind(new(Service), new(*service)),
	NewHandler,
)

// NewStarterManifest describes the knowledge routes, owner dependencies, and persistence.
func NewStarterManifest(handler *Handler) assembly.StarterManifest {
	return assembly.NewStaticStarterManifest(
		config.StarterKnowledge,
		assembly.WithStarterSummary("Organization knowledge articles with full-text and substring search."),
		assembly.WithStarterDependencies(config.StarterUser, config.StarterAudit, config.StarterOrganization),
		assembly.WithStarterModule(handler),
		assembly.WithStarterMigrationNames("2026_10_11_000000_create_knowledge_articles_table"),
	)
}
