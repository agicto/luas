// Package migrations provides database migration files for the Luas scaffold.
// Migrations are registered using the new Migration interface from internal/infra/migration.
package migrations

import (
	"slices"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

// registry holds all registered migrations
var registry = make(map[string]migration.Migration)

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
// Deprecated: starter.DefaultMigrations() is the canonical source for default starter assembly.
func Default() map[string]migration.Migration {
	out := make(map[string]migration.Migration, len(registry))
	for name, m := range registry {
		out[name] = m
	}
	return out
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
