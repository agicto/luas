package operator

import (
	"github.com/google/wire"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/starter/assembly"
)

// ProviderSet wires the optional platform-operator starter.
var ProviderSet = wire.NewSet(
	NewRepository,
	wire.Bind(new(grantStore), new(*repository)),
	NewService,
	wire.Bind(new(domain.OperatorGrantStore), new(*service)),
	wire.Struct(new(Surfaces), "*"),
	NewHandler,
)

// NewStarterManifest describes operator dependencies, routes, and persistence ownership.
func NewStarterManifest(handler *Handler) assembly.StarterManifest {
	return assembly.NewStaticStarterManifest(
		config.StarterOperator,
		assembly.WithStarterDependencies(config.StarterUser, config.StarterAudit),
		assembly.WithStarterModule(handler),
		assembly.WithStarterMigrationNames("2026_10_01_000000_create_platform_operators_table"),
	)
}
