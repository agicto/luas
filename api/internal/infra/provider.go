package infra

import (
	"github.com/google/wire"

	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/database"
	"github.com/zgiai/luas/api/internal/infra/email"
	"github.com/zgiai/luas/api/internal/infra/events"
	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/infra/storage"
)

// ProviderSet aggregates all infrastructure providers for Wire DI.
// This is the single source of truth for infrastructure dependencies.
var ProviderSet = wire.NewSet(
	// Config - loaded from environment
	config.Load,
	ConfiguredProviderSet,
)

// ConfiguredProviderSet aggregates infrastructure providers that depend on an already-resolved config.
// Bootstraps and tests reuse this graph through wiring.InitApplicationWithConfig.
var ConfiguredProviderSet = wire.NewSet(
	// Database - depends on Config
	database.NewDB,

	// Email Service - depends on Config
	email.NewService,

	// Event Bus
	events.NewEventBus,

	// Object storage - disabled, local, or private R2 from the typed config.
	storage.ProviderSet,

	// Migration - depends on Database and EventBus
	migration.ProviderSet,
)
