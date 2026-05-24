package apikey

import (
	"github.com/google/wire"
	"github.com/zgiai/zgo/internal/contracts"
	"github.com/zgiai/zgo/internal/domain"
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
func NewStarterManifest(handler *Handler) contracts.StarterManifest {
	return contracts.NewStaticStarterManifest(
		"apikey",
		contracts.WithStarterModule(handler),
		contracts.WithStarterMigrationNames("2026_04_06_000000_create_api_keys_table"),
	)
}
