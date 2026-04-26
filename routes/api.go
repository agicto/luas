package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zgiai/zgo/internal/infra/router"
	"github.com/zgiai/zgo/internal/starter"
)

// RegisterAPI registers all API routes using fluent router
func RegisterAPI(r *router.Router, starters *starter.Registry) {
	// 1. Health Checks
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "version": "v1"})
	}).Name("health")

	// 2. Register Module Routes
	r.WithMiddleware("audit")
	starters.RegisterRoutes(r)
}
