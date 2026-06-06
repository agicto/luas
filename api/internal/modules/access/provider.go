package access

import (
	"github.com/google/wire"

	"github.com/zgiai/luas/api/internal/contracts"
	"github.com/zgiai/luas/api/internal/domain"
)

// ProviderSet is the provider set for this module.
var ProviderSet = wire.NewSet(
	NewRepository,
	wire.Bind(new(domain.AccessRepository), new(*repository)),
	NewService,
	wire.Bind(new(Service), new(*service)),
	NewHandler,
)

// NewStarterManifest describes how the access starter participates in the default scaffold.
func NewStarterManifest(handler *Handler) contracts.StarterManifest {
	return contracts.NewStaticStarterManifest(
		"access",
		contracts.WithStarterModule(handler),
		contracts.WithStarterMigrations(contracts.StarterMigration{Name: "2026_06_06_110303_create_access_roles_table"}),
	)
}
