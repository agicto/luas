package asset

import (
	"github.com/google/wire"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/starter/assembly"
)

// ProviderSet wires the optional asset starter and provider-neutral storage seam.
var ProviderSet = wire.NewSet(
	NewRepository,
	wire.Bind(new(assetStore), new(*repository)),
	NewContentInspector,
	NewTransferSigner,
	NewService,
	wire.Bind(new(Service), new(*service)),
	wire.Bind(new(domain.AssetReader), new(*service)),
	wire.Bind(new(domain.AssetMaintainer), new(*service)),
	NewHandler,
)

// NewStarterManifest describes asset routes, dependencies, and persistence ownership.
func NewStarterManifest(handler *Handler) assembly.StarterManifest {
	return assembly.NewStaticStarterManifest(
		"asset",
		assembly.WithStarterDependencies("user", "audit"),
		assembly.WithStarterModule(handler),
		assembly.WithStarterMigrationNames("2026_07_15_030000_create_assets_table"),
	)
}
