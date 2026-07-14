package starter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/infra/events"
	"github.com/zgiai/luas/api/internal/infra/router"
	"github.com/zgiai/luas/api/internal/starter/assembly"
)

type counters struct {
	routes     int
	middleware int
	events     int
}

type moduleOnly struct {
	name string
}

func (m *moduleOnly) Name() string {
	return m.name
}

type routeOnly struct {
	name     string
	counters *counters
}

func (m *routeOnly) Name() string {
	return m.name
}

func (m *routeOnly) RegisterRoutes(r *router.Router) {
	m.counters.routes++
}

type middlewareOnly struct {
	name     string
	counters *counters
}

func (m *middlewareOnly) Name() string {
	return m.name
}

func (m *middlewareOnly) RegisterMiddleware(r *router.Router) {
	m.counters.middleware++
}

type eventOnly struct {
	name     string
	counters *counters
}

func (m *eventOnly) Name() string {
	return m.name
}

func (m *eventOnly) RegisterEvents(bus *events.EventBus) {
	m.counters.events++
}

type fullModule struct {
	name     string
	counters *counters
}

type activationOnly struct {
	name     string
	counters *counters
	err      error
}

func (m *activationOnly) Name() string {
	return m.name
}

func (m *activationOnly) Activate() error {
	m.counters.events++
	return m.err
}

func (m *fullModule) Name() string {
	return m.name
}

func (m *fullModule) RegisterRoutes(r *router.Router) {
	m.counters.routes++
}

func (m *fullModule) RegisterMiddleware(r *router.Router) {
	m.counters.middleware++
}

func (m *fullModule) RegisterEvents(bus *events.EventBus) {
	m.counters.events++
}

var (
	_ assembly.Module           = (*moduleOnly)(nil)
	_ assembly.RouteModule      = (*routeOnly)(nil)
	_ assembly.MiddlewareModule = (*middlewareOnly)(nil)
	_ assembly.EventModule      = (*eventOnly)(nil)
	_ assembly.RouteModule      = (*fullModule)(nil)
	_ assembly.MiddlewareModule = (*fullModule)(nil)
	_ assembly.EventModule      = (*fullModule)(nil)
	_ assembly.ActivationModule = (*activationOnly)(nil)
)

func TestRegistryDispatchesOnlySupportedCapabilities(t *testing.T) {
	registry := NewRegistry()
	routeCounters := &counters{}
	middlewareCounters := &counters{}
	eventCounters := &counters{}
	fullCounters := &counters{}

	registry.RegisterModule(&moduleOnly{name: "module-only"})
	registry.RegisterModule(&routeOnly{name: "route", counters: routeCounters})
	registry.RegisterModule(&middlewareOnly{name: "middleware", counters: middlewareCounters})
	registry.RegisterModule(&eventOnly{name: "event", counters: eventCounters})
	registry.RegisterModule(&fullModule{name: "full", counters: fullCounters})

	registry.RegisterRoutes(nil)
	registry.RegisterMiddleware(nil)
	registry.RegisterEvents(nil)

	assert.Equal(t, counters{routes: 1}, *routeCounters)
	assert.Equal(t, counters{middleware: 1}, *middlewareCounters)
	assert.Equal(t, counters{events: 1}, *eventCounters)
	assert.Equal(t, counters{routes: 1, middleware: 1, events: 1}, *fullCounters)
}

func TestRegistryModulesReturnsClone(t *testing.T) {
	registry := NewRegistry()
	registry.RegisterModule(&moduleOnly{name: "module-only"})

	modules := registry.Modules()
	assert.Len(t, modules, 1)

	modules[0] = &moduleOnly{name: "mutated"}

	original := registry.Modules()
	assert.Len(t, original, 1)
	assert.Equal(t, "module-only", original[0].Name())
}

func TestRegistryActivatesOnlyAppliedManifestModules(t *testing.T) {
	registry := NewRegistry()
	activationCounters := &counters{}
	manifest := assembly.NewStaticStarterManifest(
		"organization",
		assembly.WithStarterModule(&activationOnly{name: "organization", counters: activationCounters}),
	)

	assert.NoError(t, registry.ApplyManifest(manifest))
	assert.Equal(t, 1, activationCounters.events)
	assert.Equal(t, []string{"organization"}, registry.StarterNames())

	err := registry.ApplyManifest(manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already applied")
	assert.Equal(t, 1, activationCounters.events)
}

func TestRegistryDoesNotActivateOrMutateWhenManifestAssetsAreInvalid(t *testing.T) {
	registry := NewRegistry()
	activationCounters := &counters{}
	manifest := assembly.NewStaticStarterManifest(
		"invalid-assets",
		assembly.WithStarterModule(&activationOnly{name: "invalid-assets", counters: activationCounters}),
		assembly.WithStarterMigrationNames("2099_01_01_000000_missing"),
	)

	err := registry.ApplyManifest(manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
	assert.Zero(t, activationCounters.events)
	assert.Empty(t, registry.Modules())
	assert.Empty(t, registry.StarterNames())
}

func TestRegistryRejectsMigrationOwnershipCollisionsBeforeActivation(t *testing.T) {
	registry := NewRegistry()
	const migrationName = "2025_06_18_000000_create_users_table"
	first := assembly.NewStaticStarterManifest(
		"first",
		assembly.WithStarterMigrationNames(migrationName),
	)
	assert.NoError(t, registry.ApplyManifest(first))

	activationCounters := &counters{}
	second := assembly.NewStaticStarterManifest(
		"second",
		assembly.WithStarterModule(&activationOnly{name: "second", counters: activationCounters}),
		assembly.WithStarterMigrationNames(migrationName),
	)

	err := registry.ApplyManifest(second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already owned")
	assert.Zero(t, activationCounters.events)
	assert.Equal(t, []string{"first"}, registry.StarterNames())
	assert.Empty(t, registry.Modules())
}
