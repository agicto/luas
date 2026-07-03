package trend

import "github.com/zgiai/luas/api/internal/infra/router"

// RegisterRoutes registers the trend pipeline routes.
func (h *Handler) RegisterRoutes(r *router.Router) {
	r.GET("/trends", h.List).Name("trends.index")
	r.GET("/trend-stats", h.Stats).Name("trend_stats.show")
	r.POST("/trend-sync-runs", h.CreateSyncRun).Name("trend_sync_runs.store")
}
