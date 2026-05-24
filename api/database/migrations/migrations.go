// Package migrations provides database migration files for the ZGO scaffold.
// Migrations are registered using the new Migration interface from internal/infra/migration.
package migrations

import (
	"slices"

	"github.com/zgiai/zgo/internal/infra/migration"
)

// registry holds all registered migrations
var registry = make(map[string]migration.Migration)

var defaultExcluded = map[string]struct{}{
	"2025_12_26_000000_create_roles_table":            {},
	"2025_12_26_000001_create_permissions_table":      {},
	"2025_12_26_000002_create_role_permissions_table": {},
	"2025_12_26_000003_create_user_roles_table":       {},
	"2025_12_26_000004_seed_default_roles":            {},
}

// register adds a migration to the registry.
// This is called by init() functions in migration files.
func register(name string, m migration.Migration) {
	registry[name] = m
}

// All returns all registered migrations as a map.
// The key is the migration name (e.g., "2025_06_18_000000_create_users_table").
func All() map[string]migration.Migration {
	return registry
}

// Default returns the legacy default scaffold migration set.
// Optional example-module migrations remain available in All().
// Deprecated: starter.DefaultMigrations() is the canonical source for default starter assembly.
func Default() map[string]migration.Migration {
	filtered := make(map[string]migration.Migration)
	for name, m := range registry {
		if _, excluded := defaultExcluded[name]; excluded {
			continue
		}
		filtered[name] = m
	}
	return filtered
}

// Names returns all registered migration names in sorted order.
func Names() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
