package infra

import (
	"github.com/google/wire"
	"github.com/zgiai/zgo/internal/infra/config"
	"github.com/zgiai/zgo/internal/infra/database"
	"github.com/zgiai/zgo/internal/infra/email"
	"github.com/zgiai/zgo/internal/infra/events"
	"github.com/zgiai/zgo/internal/infra/jwt"
	"github.com/zgiai/zgo/internal/infra/migration"
	"github.com/zgiai/zgo/internal/infra/storage"
)

// ProviderSet aggregates all infrastructure providers for Wire DI.
// This is the single source of truth for infrastructure dependencies.
var ProviderSet = wire.NewSet(
	// Config - loaded from environment
	config.Load,

	// ClickHouse Client
	storage.NewCHClient,

	// Database - depends on Config
	database.NewDB,

	// JWT Service - depends on Config
	jwt.NewService,

	// Email Service - depends on Config
	email.NewService,

	// Event Bus
	events.NewEventBus,

	// Migration - depends on Database and EventBus
	migration.ProviderSet,
)
