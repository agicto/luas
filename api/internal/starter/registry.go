package starter

import (
	"fmt"
	"slices"

	"github.com/zgiai/zgo/database/migrations"
	"github.com/zgiai/zgo/database/seeders"
	"github.com/zgiai/zgo/internal/contracts"
	"github.com/zgiai/zgo/internal/infra/events"
	"github.com/zgiai/zgo/internal/infra/migration"
	"github.com/zgiai/zgo/internal/infra/router"
)

// Registry is the single assembly point for the default scaffold starters.
// It keeps module, migration, and seeder registration in one place.
type Registry struct {
	modules    []contracts.Module
	migrations map[string]migration.Migration
	seeders    []seeders.Seeder
}

// NewRegistry creates an empty starter registry.
func NewRegistry() *Registry {
	return &Registry{
		migrations: make(map[string]migration.Migration),
	}
}

// RegisterModule adds a starter module to the registry.
func (r *Registry) RegisterModule(module contracts.Module) {
	if module == nil {
		return
	}
	r.modules = append(r.modules, module)
}

// RegisterMigration adds a starter migration to the registry.
func (r *Registry) RegisterMigration(name string, m migration.Migration) {
	if name == "" || m == nil {
		return
	}
	r.migrations[name] = m
}

// RegisterSeeder adds a starter seeder to the registry.
func (r *Registry) RegisterSeeder(seeder seeders.Seeder) {
	if seeder == nil {
		return
	}
	r.seeders = append(r.seeders, seeder)
}

// RegisterMigrationByName resolves and registers a migration from the global catalog.
func (r *Registry) RegisterMigrationByName(name string) error {
	if name == "" {
		return nil
	}

	m, ok := migrations.All()[name]
	if !ok {
		return fmt.Errorf("starter migration %q not registered", name)
	}

	r.RegisterMigration(name, m)
	return nil
}

// RegisterSeederByName resolves and registers a seeder from the global catalog.
func (r *Registry) RegisterSeederByName(name string) error {
	if name == "" {
		return nil
	}

	for _, seeder := range seeders.All() {
		if seeder.Name() == name {
			r.RegisterSeeder(seeder)
			return nil
		}
	}

	return fmt.Errorf("starter seeder %q not registered", name)
}

// ApplyManifest lets a starter manifest register its modules and bootstrap assets.
func (r *Registry) ApplyManifest(manifest contracts.StarterManifest) error {
	if manifest == nil {
		return nil
	}
	for _, module := range manifest.Modules() {
		r.RegisterModule(module)
	}
	for _, name := range manifest.MigrationNames() {
		if err := r.RegisterMigrationByName(name); err != nil {
			return fmt.Errorf("register starter migration for %s: %w", manifest.Name(), err)
		}
	}
	for _, name := range manifest.SeederNames() {
		if err := r.RegisterSeederByName(name); err != nil {
			return fmt.Errorf("register starter seeder for %s: %w", manifest.Name(), err)
		}
	}
	return nil
}

// Modules returns the registered starter modules in registration order.
func (r *Registry) Modules() []contracts.Module {
	if r == nil {
		return nil
	}
	return slices.Clone(r.modules)
}

// RegisterRoutes lets route-aware modules attach their HTTP routes.
func (r *Registry) RegisterRoutes(routes *router.Router) {
	if r == nil {
		return
	}
	for _, module := range r.modules {
		routeModule, ok := module.(contracts.RouteModule)
		if !ok {
			continue
		}
		routeModule.RegisterRoutes(routes)
	}
}

// RegisterMiddleware lets middleware-aware modules attach aliases or groups.
func (r *Registry) RegisterMiddleware(routes *router.Router) {
	if r == nil {
		return
	}
	for _, module := range r.modules {
		middlewareModule, ok := module.(contracts.MiddlewareModule)
		if !ok {
			continue
		}
		middlewareModule.RegisterMiddleware(routes)
	}
}

// RegisterEvents lets event-aware modules attach subscribers to the event bus.
func (r *Registry) RegisterEvents(bus *events.EventBus) {
	if r == nil {
		return
	}
	for _, module := range r.modules {
		eventModule, ok := module.(contracts.EventModule)
		if !ok {
			continue
		}
		eventModule.RegisterEvents(bus)
	}
}

// Migrations returns the registered starter migrations.
func (r *Registry) Migrations() map[string]migration.Migration {
	if r == nil {
		return nil
	}
	cloned := make(map[string]migration.Migration, len(r.migrations))
	for name, m := range r.migrations {
		cloned[name] = m
	}
	return cloned
}

// Seeders returns the registered starter seeders in registration order.
func (r *Registry) Seeders() []seeders.Seeder {
	if r == nil {
		return nil
	}
	return slices.Clone(r.seeders)
}
