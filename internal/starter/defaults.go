package starter

import (
	"fmt"

	"github.com/google/wire"
	"github.com/zgiai/zgo/database/migrations"
	"github.com/zgiai/zgo/database/seeders"
	"github.com/zgiai/zgo/internal/infra/migration"
	"github.com/zgiai/zgo/internal/modules/apikey"
	"github.com/zgiai/zgo/internal/modules/deployment"
	"github.com/zgiai/zgo/internal/modules/platform"
	"github.com/zgiai/zgo/internal/modules/user"
)

var (
	defaultMigrationNames = []string{
		"2025_06_18_000000_create_users_table",
		"2025_06_18_000001_seed_default_users",
		"2026_04_06_000000_create_api_keys_table",
	}
	defaultSeederNames = []string{
		"users",
	}
)

// ProviderSet wires the default scaffold starters and their registry.
var ProviderSet = wire.NewSet(
	apikey.ProviderSet,
	deployment.ProviderSet,
	platform.ProviderSet,
	user.ProviderSet,
	NewDefaultRegistry,
)

// NewDefaultRegistry creates the default scaffold starter registry.
func NewDefaultRegistry(
	apiKeyHandler *apikey.Handler,
	deploymentHandler *deployment.Handler,
	platformHandler *platform.Handler,
	userHandler *user.Handler,
) (*Registry, error) {
	registry := NewRegistry()
	registry.RegisterModule(apiKeyHandler)
	registry.RegisterModule(deploymentHandler)
	registry.RegisterModule(platformHandler)
	registry.RegisterModule(userHandler)

	defaultMigrations, err := DefaultMigrations()
	if err != nil {
		return nil, err
	}
	for name, m := range defaultMigrations {
		registry.RegisterMigration(name, m)
	}

	defaultSeeders, err := DefaultSeeders()
	if err != nil {
		return nil, err
	}
	for _, seeder := range defaultSeeders {
		registry.RegisterSeeder(seeder)
	}

	return registry, nil
}

// DefaultMigrations returns the migrations enabled by the default starters.
func DefaultMigrations() (map[string]migration.Migration, error) {
	all := migrations.All()
	filtered := make(map[string]migration.Migration, len(defaultMigrationNames))
	for _, name := range defaultMigrationNames {
		m, ok := all[name]
		if !ok {
			return nil, fmt.Errorf("starter migration %q not registered", name)
		}
		filtered[name] = m
	}
	return filtered, nil
}

// DefaultSeeders returns the seeders enabled by the default starters.
func DefaultSeeders() ([]seeders.Seeder, error) {
	index := make(map[string]seeders.Seeder)
	for _, seeder := range seeders.All() {
		index[seeder.Name()] = seeder
	}

	filtered := make([]seeders.Seeder, 0, len(defaultSeederNames))
	for _, name := range defaultSeederNames {
		seeder, ok := index[name]
		if !ok {
			return nil, fmt.Errorf("starter seeder %q not registered", name)
		}
		filtered = append(filtered, seeder)
	}
	return filtered, nil
}
