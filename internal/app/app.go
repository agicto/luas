package app

import (
	"github.com/zgiai/zgo/internal/contracts"
	"github.com/zgiai/zgo/internal/infra/config"
	"github.com/zgiai/zgo/internal/infra/email"
	"github.com/zgiai/zgo/internal/infra/events"
	"github.com/zgiai/zgo/internal/infra/jwt"
	"github.com/zgiai/zgo/internal/infra/migration"
	"github.com/zgiai/zgo/internal/modules/permission"
	"github.com/zgiai/zgo/internal/modules/user"
	"gorm.io/gorm"
)

// Application holds all application dependencies injected via Wire.
// This is the root container for the entire application.
type Application struct {
	Config       *config.Config
	DB           *gorm.DB
	JWTService   *jwt.Service
	EmailService *email.Service
	EventBus     *events.EventBus
	Migrator     *migration.Migrator
	Handlers     *Handlers
}

// Handlers holds all HTTP handlers for modules.
type Handlers struct {
	User       *user.Handler
	Permission *permission.Handler
}

// Modules returns a list of all active modules
func (h *Handlers) Modules() []contracts.Module {
	return []contracts.Module{
		h.User,
		h.Permission,
	}
}
