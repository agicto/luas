package app

import (
	"github.com/zgiai/zgo/internal/infra/config"
	"github.com/zgiai/zgo/internal/infra/email"
	"github.com/zgiai/zgo/internal/infra/events"
	"github.com/zgiai/zgo/internal/infra/migration"
	"github.com/zgiai/zgo/internal/starter"
	"gorm.io/gorm"
)

// Application holds all application dependencies injected via Wire.
// This is the root container for the entire application.
type Application struct {
	Config       *config.Config
	DB           *gorm.DB
	EmailService *email.Service
	EventBus     *events.EventBus
	Migrator     *migration.Migrator
	Starters     *starter.Registry
}
