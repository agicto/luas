package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/infra/router"
	"github.com/zgiai/luas/api/internal/starter"
)

// Setup configures all application routes using the fluent router API
func Setup(engine *gin.Engine, starters *starter.Registry) *router.Router {
	r := router.New(engine)

	// Let modules extend router middleware without editing the core route setup.
	starters.RegisterMiddleware(r)

	// Root endpoint - Welcome page
	RegisterWelcome(engine)

	// Register V1 API Routes
	r.Group("/v1", func(api *router.Router) {
		RegisterAPI(api, starters)
	})

	return r
}
