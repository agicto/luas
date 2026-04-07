package apikey

import (
	"github.com/google/wire"
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
