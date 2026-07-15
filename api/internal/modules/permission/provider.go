package permission

import (
	"github.com/google/wire"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/starter/assembly"
)

// ProviderSet wires the optional permission starter and its replaceable authorizer.
var ProviderSet = wire.NewSet(
	NewRepository,
	wire.Bind(new(domain.PermissionRepository), new(*repository)),
	NewDefaultCatalog,
	NewService,
	wire.Bind(new(Service), new(*service)),
	wire.Bind(new(domain.PermissionAuthorizer), new(*service)),
	NewGuard,
	NewHandler,
)

// NewStarterManifest describes permission runtime and migration ownership.
func NewStarterManifest(handler *Handler) assembly.StarterManifest {
	return assembly.NewStaticStarterManifest(
		"permission",
		assembly.WithStarterDependencies("organization"),
		assembly.WithStarterModule(handler),
		assembly.WithStarterMigrationNames("2026_07_15_010000_create_permission_tables"),
	)
}
