package app

import (
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/email"
	"github.com/zgiai/luas/api/internal/infra/events"
	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/starter"
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
