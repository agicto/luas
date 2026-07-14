package starter

import (
	"fmt"
	"slices"

	"github.com/zgiai/luas/api/database/migrations"
	"github.com/zgiai/luas/api/database/seeders"
	"github.com/zgiai/luas/api/internal/infra/events"
	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/infra/router"
	"github.com/zgiai/luas/api/internal/starter/assembly"
)

// Registry is the single assembly point for active default and optional starters.
// It keeps module, migration, seeder, and activation ownership in one place.
type Registry struct {
	starters        []string
	starterSet      map[string]struct{}
	modules         []assembly.Module
	moduleOwners    map[string]string
	migrations      map[string]migration.Migration
	migrationOwners map[string]string
	seeders         []seeders.Seeder
	seederOwners    map[string]string
}

// NewRegistry creates an empty starter registry.
func NewRegistry() *Registry {
	return &Registry{
		starterSet:      make(map[string]struct{}),
		moduleOwners:    make(map[string]string),
		migrations:      make(map[string]migration.Migration),
		migrationOwners: make(map[string]string),
		seederOwners:    make(map[string]string),
	}
}

// RegisterModule adds a starter module to the registry.
func (r *Registry) RegisterModule(module assembly.Module) {
	if isNilValue(module) {
		return
	}
	name := module.Name()
	if name == "" {
		return
	}
	if _, exists := r.moduleOwners[name]; exists {
		return
	}
	r.moduleOwners[name] = "direct registration"
	r.modules = append(r.modules, module)
}

// RegisterMigration adds a starter migration to the registry.
func (r *Registry) RegisterMigration(name string, m migration.Migration) {
	if name == "" || isNilValue(m) {
		return
	}
	if _, exists := r.migrationOwners[name]; exists {
		return
	}
	r.migrationOwners[name] = "direct registration"
	r.migrations[name] = m
}

// RegisterSeeder adds a starter seeder to the registry.
func (r *Registry) RegisterSeeder(seeder seeders.Seeder) {
	if isNilValue(seeder) {
		return
	}
	name := seeder.Name()
	if name == "" {
		return
	}
	if _, exists := r.seederOwners[name]; exists {
		return
	}
	r.seederOwners[name] = "direct registration"
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
	if owner, exists := r.migrationOwners[name]; exists {
		return fmt.Errorf("starter migration %q is already owned by %s", name, owner)
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
			if owner, exists := r.seederOwners[name]; exists {
				return fmt.Errorf("starter seeder %q is already owned by %s", name, owner)
			}
			r.RegisterSeeder(seeder)
			return nil
		}
	}

	return fmt.Errorf("starter seeder %q not registered", name)
}

// ApplyManifest lets a starter manifest register its modules and bootstrap assets.
func (r *Registry) ApplyManifest(manifest assembly.StarterManifest) error {
	name, err := validateCatalogManifest(manifest)
	if err != nil {
		return err
	}
	if _, exists := r.starterSet[name]; exists {
		return fmt.Errorf("starter manifest %q already applied", name)
	}

	modules := manifest.Modules()
	moduleNames := make(map[string]struct{}, len(modules))
	for _, module := range modules {
		if isNilValue(module) {
			return fmt.Errorf("starter module for %s is required", name)
		}
		moduleName := module.Name()
		if !starterNamePattern.MatchString(moduleName) {
			return fmt.Errorf("starter module name %q must be canonical lowercase", moduleName)
		}
		if _, duplicate := moduleNames[moduleName]; duplicate {
			return fmt.Errorf("starter module %q is listed more than once by %s", moduleName, name)
		}
		moduleNames[moduleName] = struct{}{}
		if owner, exists := r.moduleOwners[moduleName]; exists {
			return fmt.Errorf("starter module %q is already owned by %s", moduleName, owner)
		}
	}

	migrationNames := manifest.MigrationNames()
	resolvedMigrations := make(map[string]migration.Migration, len(migrationNames))
	allMigrations := migrations.All()
	for _, migrationName := range migrationNames {
		if migrationName == "" {
			return fmt.Errorf("starter migration name for %s is required", name)
		}
		if _, duplicate := resolvedMigrations[migrationName]; duplicate {
			return fmt.Errorf("starter migration %q is listed more than once by %s", migrationName, name)
		}
		if owner, exists := r.migrationOwners[migrationName]; exists {
			return fmt.Errorf("starter migration %q is already owned by %s", migrationName, owner)
		}
		resolved, exists := allMigrations[migrationName]
		if !exists || isNilValue(resolved) {
			return fmt.Errorf("starter migration %q not registered", migrationName)
		}
		resolvedMigrations[migrationName] = resolved
	}

	seederNames := manifest.SeederNames()
	resolvedSeeders := make(map[string]seeders.Seeder, len(seederNames))
	allSeeders := make(map[string]seeders.Seeder)
	for _, seeder := range seeders.All() {
		if !isNilValue(seeder) {
			allSeeders[seeder.Name()] = seeder
		}
	}
	for _, seederName := range seederNames {
		if seederName == "" {
			return fmt.Errorf("starter seeder name for %s is required", name)
		}
		if _, duplicate := resolvedSeeders[seederName]; duplicate {
			return fmt.Errorf("starter seeder %q is listed more than once by %s", seederName, name)
		}
		if owner, exists := r.seederOwners[seederName]; exists {
			return fmt.Errorf("starter seeder %q is already owned by %s", seederName, owner)
		}
		resolved, exists := allSeeders[seederName]
		if !exists {
			return fmt.Errorf("starter seeder %q not registered", seederName)
		}
		resolvedSeeders[seederName] = resolved
	}

	// Activation happens only after every manifest contribution passes preflight.
	for _, module := range modules {
		if activatable, ok := module.(assembly.ActivationModule); ok {
			if err := activatable.Activate(); err != nil {
				return fmt.Errorf("activate starter module %s: %w", module.Name(), err)
			}
		}
	}

	for _, module := range modules {
		moduleName := module.Name()
		r.moduleOwners[moduleName] = name
		r.modules = append(r.modules, module)
	}
	for migrationName, resolved := range resolvedMigrations {
		r.migrationOwners[migrationName] = name
		r.migrations[migrationName] = resolved
	}
	for _, seederName := range seederNames {
		r.seederOwners[seederName] = name
		r.seeders = append(r.seeders, resolvedSeeders[seederName])
	}
	r.starterSet[name] = struct{}{}
	r.starters = append(r.starters, name)
	return nil
}

// StarterNames returns active starter names in deterministic assembly order.
func (r *Registry) StarterNames() []string {
	if r == nil {
		return nil
	}
	return slices.Clone(r.starters)
}

// Modules returns the registered starter modules in registration order.
func (r *Registry) Modules() []assembly.Module {
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
		routeModule, ok := module.(assembly.RouteModule)
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
		middlewareModule, ok := module.(assembly.MiddlewareModule)
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
		eventModule, ok := module.(assembly.EventModule)
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
