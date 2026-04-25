package starter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgiai/zgo/internal/contracts"
	"github.com/zgiai/zgo/internal/infra/events"
	"github.com/zgiai/zgo/internal/infra/router"
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
	_ contracts.Module           = (*moduleOnly)(nil)
	_ contracts.RouteModule      = (*routeOnly)(nil)
	_ contracts.MiddlewareModule = (*middlewareOnly)(nil)
	_ contracts.EventModule      = (*eventOnly)(nil)
	_ contracts.RouteModule      = (*fullModule)(nil)
	_ contracts.MiddlewareModule = (*fullModule)(nil)
	_ contracts.EventModule      = (*fullModule)(nil)
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
