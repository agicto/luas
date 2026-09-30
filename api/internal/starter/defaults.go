package starter

import (
	"fmt"

	"github.com/google/wire"

	"github.com/zgiai/luas/api/database/seeders"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/modules/apikey"
	"github.com/zgiai/luas/api/internal/modules/asset"
	"github.com/zgiai/luas/api/internal/modules/audit"
	"github.com/zgiai/luas/api/internal/modules/notification"
	"github.com/zgiai/luas/api/internal/modules/organization"
	permissionstarter "github.com/zgiai/luas/api/internal/modules/permission"
	"github.com/zgiai/luas/api/internal/modules/setting"
	"github.com/zgiai/luas/api/internal/modules/usage"
	"github.com/zgiai/luas/api/internal/modules/user"
	"github.com/zgiai/luas/api/internal/modules/webhook"
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
	asset.ProviderSet,
	setting.ProviderSet,
	usage.ProviderSet,
	webhook.ProviderSet,
	wire.Struct(new(Handlers), "*"),
	NewConfiguredRegistry,
)

// Handlers carries each starter's runtime HTTP module. Wire fills every field, so adding a starter
// adds one field here instead of a positional parameter to every assembly function. A nil
// *Handlers describes starters without runtime modules: catalog, validation, migration, and seeder
// resolution only need that static metadata.
type Handlers struct {
	Audit        *audit.Handler
	APIKey       *apikey.Handler
	User         *user.Handler
	Organization *organization.Handler
	Permission   *permissionstarter.Handler
	Notification *notification.Handler
	Asset        *asset.Handler
	Setting      *setting.Handler
	Usage        *usage.Handler
	Webhook      *webhook.Handler
}

func (h *Handlers) orMetadataOnly() *Handlers {
	if h == nil {
		return &Handlers{}
	}
	return h
}

// NewConfiguredRegistry assembles defaults plus configured optional starters and registers their
// migrations on the application migrator from the same selection snapshot.
func NewConfiguredRegistry(cfg *config.Config, migrator *migration.Migrator, handlers *Handlers) (*Registry, error) {
	manifests, err := ConfiguredManifests(cfg, handlers)
	if err != nil {
		return nil, err
	}
	registry, err := registryFromManifests(manifests)
	if err != nil {
		return nil, err
	}
	if migrator != nil {
		if err := registerCoreMigrations(registry); err != nil {
			return nil, err
		}
		migrator.RegisterMany(registry.Migrations())
	}
	return registry, nil
}

// NewDefaultRegistry creates the default scaffold starter registry.
func NewDefaultRegistry(handlers *Handlers) (*Registry, error) {
	return registryFromManifests(DefaultManifests(handlers))
}

// OptionalManifests returns the starters available for additive activation.
func OptionalManifests(handlers *Handlers) []assembly.StarterManifest {
	h := handlers.orMetadataOnly()
	return []assembly.StarterManifest{
		organization.NewStarterManifest(h.Organization),
		permissionstarter.NewStarterManifest(h.Permission),
		notification.NewStarterManifest(h.Notification),
		asset.NewStarterManifest(h.Asset),
		setting.NewStarterManifest(h.Setting),
		usage.NewStarterManifest(h.Usage),
		webhook.NewStarterManifest(h.Webhook),
	}
}

// ConfiguredManifests resolves one validated configuration snapshot into active manifests.
func ConfiguredManifests(cfg *config.Config, handlers *Handlers) ([]assembly.StarterManifest, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required for starter selection")
	}
	catalog, err := NewCatalog(DefaultManifests(handlers), OptionalManifests(handlers))
	if err != nil {
		return nil, err
	}
	return catalog.Select(cfg.Starters.Optional)
}

// ValidateConfig fails before infrastructure work when starter selection is ambiguous.
func ValidateConfig(cfg *config.Config) error {
	_, err := ConfiguredManifests(cfg, nil)
	return err
}

// AvailableCatalog returns the complete starter catalog without requiring
// handlers, infrastructure, or a running application.
func AvailableCatalog() (*Catalog, error) {
	return NewCatalog(DefaultManifests(nil), OptionalManifests(nil))
}

// DefaultManifests returns the starter manifests enabled in the default scaffold.
func DefaultManifests(handlers *Handlers) []assembly.StarterManifest {
	h := handlers.orMetadataOnly()
	return []assembly.StarterManifest{
		audit.NewStarterManifest(h.Audit),
		apikey.NewStarterManifest(h.APIKey),
		user.NewStarterManifest(h.User),
	}
}

// DefaultMigrations returns the migrations enabled by the default starters.
func DefaultMigrations() (map[string]migration.Migration, error) {
	registry, err := registryFromManifests(DefaultManifests(nil))
	if err != nil {
		return nil, err
	}
	if err := registerCoreMigrations(registry); err != nil {
		return nil, err
	}
	return registry.Migrations(), nil
}

// ConfiguredMigrations returns migrations from the same additive starter selection as HTTP.
func ConfiguredMigrations(cfg *config.Config) (map[string]migration.Migration, error) {
	manifests, err := ConfiguredManifests(cfg, nil)
	if err != nil {
		return nil, err
	}
	registry, err := registryFromManifests(manifests)
	if err != nil {
		return nil, err
	}
	if err := registerCoreMigrations(registry); err != nil {
		return nil, err
	}
	return registry.Migrations(), nil
}

func registerCoreMigrations(registry *Registry) error {
	return registry.RegisterMigrationByName("2026_07_31_000000_create_workflow_tasks_table")
}

// DefaultSeeders returns the seeders enabled by the default starters.
func DefaultSeeders() ([]seeders.Seeder, error) {
	registry, err := registryFromManifests(DefaultManifests(nil))
	if err != nil {
		return nil, err
	}
	return registry.Seeders(), nil
}

// ConfiguredSeeders returns seeders from the same additive starter selection as HTTP.
func ConfiguredSeeders(cfg *config.Config) ([]seeders.Seeder, error) {
	manifests, err := ConfiguredManifests(cfg, nil)
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
