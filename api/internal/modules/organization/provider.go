package organization

import (
	"github.com/google/wire"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/starter/assembly"
)

// ProviderSet wires the optional organization starter.
var ProviderSet = wire.NewSet(
	NewRepository,
	wire.Bind(new(domain.OrganizationRepository), new(*repository)),
	wire.Bind(new(domain.OrganizationInvitationRepository), new(*repository)),
	NewInvitationPolicy,
	NewInvitationMailer,
	NewService,
	wire.Bind(new(Service), new(*service)),
	NewHandler,
)

// NewStarterManifest describes the optional organization starter surfaces.
func NewStarterManifest(handler *Handler) assembly.StarterManifest {
	return assembly.NewStaticStarterManifest(
		"organization",
		assembly.WithStarterModule(handler),
		assembly.WithStarterMigrationNames(
			"2026_07_14_000000_create_organizations_tables",
			"2026_07_15_000000_create_organization_invitations_table",
		),
	)
}
