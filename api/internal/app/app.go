package app

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/email"
	"github.com/zgiai/luas/api/internal/infra/events"
	"github.com/zgiai/luas/api/internal/infra/migration"
	"github.com/zgiai/luas/api/internal/starter"
)

// Application holds all application dependencies injected via Wire.
// This is the root container for the entire application.
type Application struct {
	Config                 *config.Config
	DB                     *gorm.DB
	EmailService           *email.Service
	EventBus               *events.EventBus
	Migrator               *migration.Migrator
	Starters               *starter.Registry
	NotificationPublisher  domain.NotificationPublisher
	NotificationDispatcher domain.NotificationDispatcher
	AssetReader            domain.AssetReader
	AssetMaintainer        domain.AssetMaintainer
}
