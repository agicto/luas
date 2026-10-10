package assembly

import (
	"context"

	"github.com/zgiai/luas/api/internal/infra/events"
	"github.com/zgiai/luas/api/internal/infra/router"
	"github.com/zgiai/luas/api/pkg/response"
)

// Module is the minimal assembly seam shared by all Luas modules.
// Optional capabilities are expressed via narrower interfaces below.
type Module interface {
	// Name returns the unique name of the module.
	Name() string
}

// ActivationModule installs runtime hooks only when its owning starter is selected.
type ActivationModule interface {
	Module
	Activate() error
}

// ShutdownModule finishes background work after the HTTP server stops taking requests and before
// shared resources such as the database close.
type ShutdownModule interface {
	Module
	Shutdown(ctx context.Context) error
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

// ErrorModule maps the module's domain errors to public HTTP status codes and error codes, so a
// starter owns its error contract without editing core bootstrap.
type ErrorModule interface {
	Module
	RegisterErrorMappings(mapper *response.ErrorMapper)
}
