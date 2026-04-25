package contracts

import (
	"github.com/zgiai/zgo/internal/infra/events"
	"github.com/zgiai/zgo/internal/infra/router"
)

// Module is the minimal contract shared by all ZGO modules.
// Optional capabilities are expressed via narrower interfaces below.
type Module interface {
	// Name returns the unique name of the module.
	Name() string
}

// RouteModule registers HTTP routes for a module.
type RouteModule interface {
	Module
	RegisterRoutes(r *router.Router)
}

// MiddlewareModule registers middleware aliases or groups for a module.
type MiddlewareModule interface {
	Module
	RegisterMiddleware(r *router.Router)
}

// EventModule registers event subscribers for a module.
type EventModule interface {
	Module
	RegisterEvents(bus *events.EventBus)
}
