package user

import (
	"github.com/google/wire"
	"github.com/zgiai/zgo/internal/domain"
)

// ProviderSet is the provider set for this module
// It binds concrete implementations to domain interfaces
var ProviderSet = wire.NewSet(
	NewRepository,
	wire.Bind(new(domain.UserRepository), new(*repository)),
	NewService,
	wire.Bind(new(AuthService), new(*service)),
	wire.Bind(new(ProfileService), new(*service)),
	wire.Bind(new(UserQueryService), new(*service)),
	wire.Bind(new(Service), new(*service)),
	NewHandler,
)
