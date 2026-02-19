package routes

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/zgiai/zgo/internal/app"
	"github.com/zgiai/zgo/internal/infra/middleware"
	"github.com/zgiai/zgo/internal/infra/monitor"
	"github.com/zgiai/zgo/internal/infra/router"
)

// Setup configures all application routes using the fluent router API
func Setup(engine *gin.Engine, handlers *app.Handlers) *router.Router {
	r := router.New(engine)

	// Register middleware groups
	r.MiddlewareGroup("web", gin.Logger(), gin.Recovery())
	r.MiddlewareGroup("api", gin.Logger(), gin.Recovery())
	r.MiddlewareGroup("auth", middleware.JWTAuth())

	// Register middleware aliases
	r.AliasMiddleware("jwt", middleware.JWTAuth())

	// Apply global middleware
	r.Use(gin.Logger(), gin.Recovery())

	// Swagger documentation
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Root endpoint - Welcome page
	RegisterWelcome(engine)

	// Register V1 API Routes
	r.Group("/v1", func(api *router.Router) {
		RegisterAPI(api, handlers)
	})

	// Register Monitor
	monitor.RegisterRoutes(engine)

	return r
}
