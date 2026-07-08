package apikey

import (
	"github.com/google/wire"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/starter/assembly"
)

// ProviderSet is the provider set for the API key module.
var ProviderSet = wire.NewSet(
	NewRepository,
	wire.Bind(new(domain.APIKeyRepository), new(*repository)),
	NewService,
	wire.Bind(new(Service), new(*service)),
	NewHandler,
)

// NewStarterManifest describes how the API key starter participates in the default scaffold.
func NewStarterManifest(handler *Handler) assembly.StarterManifest {
	return assembly.NewStaticStarterManifest(
		"apikey",
		assembly.WithStarterModule(handler),
		assembly.WithStarterMigrationNames("2026_04_06_000000_create_api_keys_table"),
	)
}
