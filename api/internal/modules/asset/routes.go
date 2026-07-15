package asset

import "github.com/zgiai/luas/api/internal/infra/router"

// RegisterRoutes attaches private asset management and local transfer endpoints.
func (h *Handler) RegisterRoutes(r *router.Router) {
	r.Group("", func(auth *router.Router) {
		auth.WithMiddleware("auth")
		auth.GET("/assets", h.List).Name("assets.index")
		auth.POST("/assets/upload-intents", h.CreateUploadIntent).Name("assets.upload-intents.store")
		auth.POST("/assets/:id/complete", h.Complete).Name("assets.complete").WhereUUID("id")
		auth.POST("/assets/:id/download-grant", h.DownloadGrant).Name("assets.download-grant").WhereUUID("id")
		auth.DELETE("/assets/:id", h.Delete).Name("assets.destroy").WhereUUID("id")
	})

	r.PUT("/asset-transfers/:token", h.LocalUpload).Name("asset-transfers.upload")
	r.GET("/asset-transfers/:token", h.LocalDownload).Name("asset-transfers.download")
}
