package deployment

import "github.com/zgiai/zgo/internal/infra/router"

func (h *Handler) RegisterRoutes(r *router.Router) {
	r.GET("/deploy/targets", h.ListTargets).Name("deploy.targets")
	r.GET("/deployments", h.ListDeployments).Name("deployments.index")
	r.POST("/deployments", h.CreateDeployment).Name("deployments.store")
	r.GET("/deployments/:id", h.GetDeployment).Name("deployments.show")
	r.GET("/deployments/:id/logs", h.ListLogs).Name("deployments.logs")
	r.GET("/deployments/:id/stream", h.StreamLogs).Name("deployments.stream")
	r.POST("/deployments/certificates", h.GenerateCertificate).Name("deployments.certificates.store")
	r.POST("/deployments/webhooks/:target", h.TriggerWebhook).Name("deployments.webhooks.store")
}
