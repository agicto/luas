package platform

import "github.com/zgiai/zgo/internal/infra/router"

func (h *Handler) RegisterRoutes(r *router.Router) {
	r.GET("/platform/overview", h.Overview).Name("platform.overview")
	r.GET("/platform/deploy-targets", h.ListDeployTargets).Name("platform.deploy-targets")
	r.GET("/platform/github/connections", h.ListGitHubConnections).Name("platform.github.connections.index")
	r.POST("/platform/github/connections", h.ConnectGitHub).Name("platform.github.connections.store")
	r.GET("/platform/github/connections/:id/repositories", h.ListGitHubRepositories).Name("platform.github.repositories.index")
	r.GET("/platform/projects", h.ListProjects).Name("platform.projects.index")
	r.POST("/platform/projects", h.CreateProject).Name("platform.projects.store")
	r.GET("/platform/services", h.ListServices).Name("platform.services.index")
	r.POST("/platform/services/import", h.ImportService).Name("platform.services.import")
	r.GET("/platform/services/:id", h.GetService).Name("platform.services.show")
	r.PUT("/platform/services/:id", h.UpdateService).Name("platform.services.update")
	r.PUT("/platform/services/:id/environment", h.ReplaceEnvironment).Name("platform.services.environment.update")
	r.GET("/platform/services/:id/deployments", h.ListServiceDeployments).Name("platform.services.deployments.index")
	r.POST("/platform/services/:id/deploy", h.DeployService).Name("platform.services.deploy")
	r.POST("/platform/services/:id/webhooks/github", h.HandleGitHubWebhook).Name("platform.services.webhooks.github")
	r.GET("/platform/deployments/:deploymentId/logs", h.ListDeploymentLogs).Name("platform.deployments.logs")
	r.GET("/platform/deployments/:deploymentId/stream", h.StreamDeploymentLogs).Name("platform.deployments.stream")
}
