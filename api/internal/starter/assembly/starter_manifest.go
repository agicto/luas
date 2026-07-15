package assembly

import "reflect"

// StarterManifest describes how a starter contributes modules and bootstrap assets.
type StarterManifest interface {
	Name() string
	Dependencies() []string
	Modules() []Module
	MigrationNames() []string
	SeederNames() []string
}

// StaticStarterManifest is a small immutable manifest for starters with fixed assets.
type StaticStarterManifest struct {
	name           string
	dependencies   []string
	modules        []Module
	migrationNames []string
	seederNames    []string
}

// Dependencies returns starter names that must be selected before this manifest.
func (m *StaticStarterManifest) Dependencies() []string {
	return append([]string(nil), m.dependencies...)
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

// MigrationNames returns the migration names required by this starter.
func (m *StaticStarterManifest) MigrationNames() []string {
	return append([]string(nil), m.migrationNames...)
}

// SeederNames returns the seeder names required by this starter.
func (m *StaticStarterManifest) SeederNames() []string {
	return append([]string(nil), m.seederNames...)
}

// WithStarterModule adds a module to a static starter manifest.
func WithStarterModule(module Module) StarterManifestOption {
	return func(manifest *StaticStarterManifest) {
		if isNilModule(module) {
			return
		}
		manifest.modules = append(manifest.modules, module)
	}
}

// WithStarterDependencies declares starter assembly prerequisites.
func WithStarterDependencies(names ...string) StarterManifestOption {
	return func(manifest *StaticStarterManifest) {
		manifest.dependencies = append(manifest.dependencies, names...)
	}
}

func isNilModule(module Module) bool {
	if module == nil {
		return true
	}
	value := reflect.ValueOf(module)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// WithStarterMigrationNames adds migration names resolved via the global migration registry.
func WithStarterMigrationNames(names ...string) StarterManifestOption {
	return func(manifest *StaticStarterManifest) {
		manifest.migrationNames = append(manifest.migrationNames, names...)
	}
}

// WithStarterSeederNames adds seeder names resolved via the global seeder registry.
func WithStarterSeederNames(names ...string) StarterManifestOption {
	return func(manifest *StaticStarterManifest) {
		manifest.seederNames = append(manifest.seederNames, names...)
	}
}
