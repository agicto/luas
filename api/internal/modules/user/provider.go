package user

import (
	"github.com/google/wire"
	"github.com/zgiai/zgo/internal/contracts"
	"github.com/zgiai/zgo/internal/domain"
	"github.com/zgiai/zgo/internal/infra/email"
)

// ProviderSet is the provider set for this module
// It binds concrete implementations to domain interfaces
var ProviderSet = wire.NewSet(
	NewRepository,
	wire.Bind(new(domain.UserRepository), new(*repository)),
	wire.Bind(new(passwordResetStore), new(*repository)),
	NewUserMailer,
	NewService,
	wire.Bind(new(AuthService), new(*service)),
	wire.Bind(new(ProfileService), new(*service)),
	wire.Bind(new(UserQueryService), new(*service)),
	wire.Bind(new(Service), new(*service)),
	NewHandler,
)

// NewUserMailer adapts the shared email service to the user module email seam.
func NewUserMailer(service *email.Service) UserMailer {
	return service
}

// NewStarterManifest describes how the user starter participates in the default scaffold.
func NewStarterManifest(handler *Handler) contracts.StarterManifest {
	return contracts.NewStaticStarterManifest(
		"user",
		contracts.WithStarterModule(handler),
		contracts.WithStarterMigrationNames(
			"2025_06_18_000000_create_users_table",
			"2026_04_27_000000_create_password_reset_tokens_table",
			"2026_04_27_000001_add_unique_index_to_users_username",
			"2025_06_18_000001_seed_default_users",
		),
		contracts.WithStarterSeederNames("users"),
	)
}
