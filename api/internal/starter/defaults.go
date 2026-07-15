package starter

import (
	"fmt"

	"github.com/google/wire"

	"github.com/zgiai/luas/api/database/seeders"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/modules/apikey"
	"github.com/zgiai/luas/api/internal/modules/audit"
	"github.com/zgiai/luas/api/internal/modules/notification"
	"github.com/zgiai/luas/api/internal/modules/organization"
	permissionstarter "github.com/zgiai/luas/api/internal/modules/permission"
	"github.com/zgiai/luas/api/internal/modules/user"
	"github.com/zgiai/luas/api/internal/starter/assembly"
)

// ProviderSet wires every available starter provider and the configured registry.
var ProviderSet = wire.NewSet(
	audit.ProviderSet,
	apikey.ProviderSet,
	user.ProviderSet,
	organization.ProviderSet,
	permissionstarter.ProviderSet,
	notification.ProviderSet,
	NewConfiguredRegistry,
)

// NewConfiguredRegistry assembles defaults plus configured optional starters and registers their
// migrations on the application migrator from the same selection snapshot.
func NewConfiguredRegistry(
	cfg *config.Config,
	migrator *migration.Migrator,
	auditHandler *audit.Handler,
	apiKeyHandler *apikey.Handler,
	userHandler *user.Handler,
	organizationHandler *organization.Handler,
	permissionHandler *permissionstarter.Handler,
	notificationHandler *notification.Handler,
) (*Registry, error) {
	manifests, err := ConfiguredManifests(
		cfg,
		auditHandler,
		apiKeyHandler,
		userHandler,
		organizationHandler,
		permissionHandler,
		notificationHandler,
	)
	if err != nil {
		return nil, err
	}
	registry, err := registryFromManifests(manifests)
	if err != nil {
		return nil, err
	}
	if migrator != nil {
		migrator.RegisterMany(registry.Migrations())
	}
	return registry, nil
}

// NewDefaultRegistry creates the default scaffold starter registry.
func NewDefaultRegistry(
	auditHandler *audit.Handler,
	apiKeyHandler *apikey.Handler,
	userHandler *user.Handler,
) (*Registry, error) {
	return registryFromManifests(DefaultManifests(auditHandler, apiKeyHandler, userHandler))
}

// OptionalManifests returns the starters available for additive activation.
func OptionalManifests(
	organizationHandler *organization.Handler,
	permissionHandler *permissionstarter.Handler,
	notificationHandler *notification.Handler,
) []assembly.StarterManifest {
	return []assembly.StarterManifest{
		organization.NewStarterManifest(organizationHandler),
		permissionstarter.NewStarterManifest(permissionHandler),
		notification.NewStarterManifest(notificationHandler),
	}
}

// ConfiguredManifests resolves one validated configuration snapshot into active manifests.
func ConfiguredManifests(
	cfg *config.Config,
	auditHandler *audit.Handler,
	apiKeyHandler *apikey.Handler,
	userHandler *user.Handler,
	organizationHandler *organization.Handler,
	permissionHandler *permissionstarter.Handler,
	notificationHandler *notification.Handler,
) ([]assembly.StarterManifest, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required for starter selection")
	}
	catalog, err := NewCatalog(
		DefaultManifests(auditHandler, apiKeyHandler, userHandler),
		OptionalManifests(organizationHandler, permissionHandler, notificationHandler),
	)
	if err != nil {
		return nil, err
	}
	return catalog.Select(cfg.Starters.Optional)
}

// ValidateConfig fails before infrastructure work when starter selection is ambiguous.
func ValidateConfig(cfg *config.Config) error {
	_, err := ConfiguredManifests(cfg, nil, nil, nil, nil, nil, nil)
	return err
}

// DefaultManifests returns the starter manifests enabled in the default scaffold.
func DefaultManifests(auditHandler *audit.Handler, apiKeyHandler *apikey.Handler, userHandler *user.Handler) []assembly.StarterManifest {
	return []assembly.StarterManifest{
		audit.NewStarterManifest(auditHandler),
		apikey.NewStarterManifest(apiKeyHandler),
		user.NewStarterManifest(userHandler),
	}
}

// DefaultMigrations returns the migrations enabled by the default starters.
func DefaultMigrations() (map[string]migration.Migration, error) {
	registry, err := registryFromManifests(DefaultManifests(nil, nil, nil))
	if err != nil {
		return nil, err
	}
	return registry.Migrations(), nil
}

// ConfiguredMigrations returns migrations from the same additive starter selection as HTTP.
func ConfiguredMigrations(cfg *config.Config) (map[string]migration.Migration, error) {
	manifests, err := ConfiguredManifests(cfg, nil, nil, nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	registry, err := registryFromManifests(manifests)
	if err != nil {
		return nil, err
	}
	return registry.Migrations(), nil
}

// DefaultSeeders returns the seeders enabled by the default starters.
func DefaultSeeders() ([]seeders.Seeder, error) {
	registry, err := registryFromManifests(DefaultManifests(nil, nil, nil))
	if err != nil {
		return nil, err
	}
	return registry.Seeders(), nil
}

// ConfiguredSeeders returns seeders from the same additive starter selection as HTTP.
func ConfiguredSeeders(cfg *config.Config) ([]seeders.Seeder, error) {
	manifests, err := ConfiguredManifests(cfg, nil, nil, nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	registry, err := registryFromManifests(manifests)
	if err != nil {
		return nil, err
	}
	return registry.Seeders(), nil
}

func registryFromManifests(manifests []assembly.StarterManifest) (*Registry, error) {
	registry := NewRegistry()
	for _, manifest := range manifests {
		if err := registry.ApplyManifest(manifest); err != nil {
			return nil, err
		}
	}
	return registry, nil
}
