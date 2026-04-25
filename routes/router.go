package routes

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/zgiai/zgo/internal/infra/monitor"
	"github.com/zgiai/zgo/internal/infra/router"
	"github.com/zgiai/zgo/internal/starter"
)

// Setup configures all application routes using the fluent router API
func Setup(engine *gin.Engine, starters *starter.Registry) *router.Router {
	r := router.New(engine)

	// Let modules extend router middleware without editing the core route setup.
	starters.RegisterMiddleware(r)

	// Swagger documentation
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Root endpoint - Welcome page
	RegisterWelcome(engine)

	// Register V1 API Routes
	r.Group("/v1", func(api *router.Router) {
		RegisterAPI(api, starters)
	})

	// Register Monitor
	monitor.RegisterRoutes(engine)

	return r
}
