package starter

import (
	"github.com/google/wire"
	"github.com/zgiai/zgo/database/seeders"
	"github.com/zgiai/zgo/internal/contracts"
	"github.com/zgiai/zgo/internal/infra/migration"
	"github.com/zgiai/zgo/internal/modules/apikey"
	"github.com/zgiai/zgo/internal/modules/audit"
	"github.com/zgiai/zgo/internal/modules/user"
)

// ProviderSet wires the default scaffold starters and their registry.
var ProviderSet = wire.NewSet(
	audit.ProviderSet,
	apikey.ProviderSet,
	user.ProviderSet,
	NewDefaultRegistry,
)

// NewDefaultRegistry creates the default scaffold starter registry.
func NewDefaultRegistry(
	auditHandler *audit.Handler,
	apiKeyHandler *apikey.Handler,
	userHandler *user.Handler,
) (*Registry, error) {
	registry := NewRegistry()
	for _, manifest := range DefaultManifests(auditHandler, apiKeyHandler, userHandler) {
		if err := registry.ApplyManifest(manifest); err != nil {
			return nil, err
		}
	}

	return registry, nil
}

// DefaultManifests returns the starter manifests enabled in the default scaffold.
func DefaultManifests(auditHandler *audit.Handler, apiKeyHandler *apikey.Handler, userHandler *user.Handler) []contracts.StarterManifest {
	return []contracts.StarterManifest{
		audit.NewStarterManifest(auditHandler),
		apikey.NewStarterManifest(apiKeyHandler),
		user.NewStarterManifest(userHandler),
	}
}

// DefaultMigrations returns the migrations enabled by the default starters.
func DefaultMigrations() (map[string]migration.Migration, error) {
	registry := NewRegistry()
	for _, manifest := range DefaultManifests(nil, nil, nil) {
		if err := registry.ApplyManifest(manifest); err != nil {
			return nil, err
		}
	}
	return registry.Migrations(), nil
}

// DefaultSeeders returns the seeders enabled by the default starters.
func DefaultSeeders() ([]seeders.Seeder, error) {
	registry := NewRegistry()
	for _, manifest := range DefaultManifests(nil, nil, nil) {
		if err := registry.ApplyManifest(manifest); err != nil {
			return nil, err
		}
	}
	return registry.Seeders(), nil
}
