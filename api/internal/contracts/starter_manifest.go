package contracts

// StarterMigration is a typed migration contribution from a starter.
type StarterMigration struct {
	Name string
}

// StarterSeeder is a typed seeder contribution from a starter.
type StarterSeeder struct {
	Name string
}

// StarterManifest describes how a starter contributes modules and bootstrap assets.
type StarterManifest interface {
	Name() string
	Modules() []Module
	Migrations() []StarterMigration
	Seeders() []StarterSeeder
	MigrationNames() []string
	SeederNames() []string
}

// StaticStarterManifest is a small immutable manifest for starters with fixed assets.
type StaticStarterManifest struct {
	name       string
	modules    []Module
	migrations []StarterMigration
	seeders    []StarterSeeder
}

// StarterManifestOption mutates a StaticStarterManifest during construction.
type StarterManifestOption func(*StaticStarterManifest)

// NewStaticStarterManifest creates a starter manifest with fixed modules and bootstrap assets.
func NewStaticStarterManifest(name string, opts ...StarterManifestOption) *StaticStarterManifest {
	manifest := &StaticStarterManifest{name: name}
	for _, opt := range opts {
		if opt != nil {
			opt(manifest)
		}
	}
	return manifest
}

// Name returns the starter name.
func (m *StaticStarterManifest) Name() string {
	return m.name
}

// Modules returns the modules registered by this starter.
func (m *StaticStarterManifest) Modules() []Module {
	return append([]Module(nil), m.modules...)
}

// Migrations returns the typed migrations required by this starter.
func (m *StaticStarterManifest) Migrations() []StarterMigration {
	return append([]StarterMigration(nil), m.migrations...)
}

// Seeders returns the typed seeders required by this starter.
func (m *StaticStarterManifest) Seeders() []StarterSeeder {
	return append([]StarterSeeder(nil), m.seeders...)
}

// MigrationNames returns the migration names required by this starter.
func (m *StaticStarterManifest) MigrationNames() []string {
	names := make([]string, 0, len(m.migrations))
	for _, item := range m.migrations {
		names = append(names, item.Name)
	}
	return names
}

// SeederNames returns the seeder names required by this starter.
func (m *StaticStarterManifest) SeederNames() []string {
	names := make([]string, 0, len(m.seeders))
	for _, item := range m.seeders {
		names = append(names, item.Name)
	}
	return names
}

// WithStarterModule adds a module to a static starter manifest.
func WithStarterModule(module Module) StarterManifestOption {
	return func(manifest *StaticStarterManifest) {
		if module == nil {
			return
		}
		manifest.modules = append(manifest.modules, module)
	}
}

// WithStarterMigrationNames adds migration names resolved via the global migration registry.
func WithStarterMigrationNames(names ...string) StarterManifestOption {
	return func(manifest *StaticStarterManifest) {
		for _, name := range names {
			manifest.migrations = append(manifest.migrations, StarterMigration{Name: name})
		}
	}
}

// WithStarterSeederNames adds seeder names resolved via the global seeder registry.
func WithStarterSeederNames(names ...string) StarterManifestOption {
	return func(manifest *StaticStarterManifest) {
		for _, name := range names {
			manifest.seeders = append(manifest.seeders, StarterSeeder{Name: name})
		}
	}
}

// WithStarterMigrations adds typed migrations owned by a starter manifest.
func WithStarterMigrations(items ...StarterMigration) StarterManifestOption {
	return func(manifest *StaticStarterManifest) {
		for _, item := range items {
			if item.Name == "" {
				continue
			}
			manifest.migrations = append(manifest.migrations, item)
		}
	}
}

// WithStarterSeeders adds typed seeders owned by a starter manifest.
func WithStarterSeeders(items ...StarterSeeder) StarterManifestOption {
	return func(manifest *StaticStarterManifest) {
		for _, item := range items {
			if item.Name == "" {
				continue
			}
			manifest.seeders = append(manifest.seeders, item)
		}
	}
}
