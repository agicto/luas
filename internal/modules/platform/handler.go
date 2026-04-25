package platform

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zgiai/zgo/internal/contracts"
	"github.com/zgiai/zgo/internal/infra/deploycontrol"
	httphandler "github.com/zgiai/zgo/pkg/handler"
	"github.com/zgiai/zgo/pkg/response"
)

type Handler struct {
	service PlatformService
}

var (
	_ contracts.Module      = (*Handler)(nil)
	_ contracts.RouteModule = (*Handler)(nil)
)

func NewHandler(service PlatformService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Name() string {
	return "platform"
}

func (h *Handler) Overview(c *gin.Context) {
	data, err := h.service.Overview(c.Request.Context())
	if err != nil {
		response.HandleError(c, "Failed to load platform overview", err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) ListDeployTargets(c *gin.Context) {
	targets, err := h.service.ListDeployTargets()
	if err != nil {
		response.HandleError(c, "Failed to load deploy targets", err)
		return
	}
	response.Success(c, targets)
}

func (h *Handler) ListGitHubConnections(c *gin.Context) {
	data, err := h.service.ListGitHubConnections(c.Request.Context())
	if err != nil {
		response.HandleError(c, "Failed to load GitHub connections", err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) ConnectGitHub(c *gin.Context) {
	var req ConnectGitHubRequest
	if !httphandler.BindJSON(c, &req) {
		return
	}

	connection, err := h.service.ConnectGitHub(c.Request.Context(), &req)
	if err != nil {
		response.HandleError(c, "Failed to connect GitHub", err)
		return
	}
	response.Created(c, connection)
}

func (h *Handler) ListGitHubRepositories(c *gin.Context) {
	connectionID, err := parseUintParam(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid connection id", err)
		return
	}

	query := c.Query("query")
	limit := httphandler.QueryInt(c, "limit", 50)

	repositories, err := h.service.ListGitHubRepositories(c.Request.Context(), connectionID, query, limit)
	if err != nil {
		response.HandleError(c, "Failed to load GitHub repositories", err)
		return
	}
	response.Success(c, repositories)
}

func (h *Handler) ListProjects(c *gin.Context) {
	projects, err := h.service.ListProjects(c.Request.Context())
	if err != nil {
		response.HandleError(c, "Failed to load projects", err)
		return
	}
	response.Success(c, projects)
}

func (h *Handler) CreateProject(c *gin.Context) {
	var req CreateProjectRequest
	if !httphandler.BindJSON(c, &req) {
		return
	}

	project, err := h.service.CreateProject(c.Request.Context(), &req)
	if err != nil {
		response.HandleError(c, "Failed to create project", err)
		return
	}
	response.Created(c, project)
}

func (h *Handler) ListServices(c *gin.Context) {
	services, err := h.service.ListServices(c.Request.Context())
	if err != nil {
		response.HandleError(c, "Failed to load services", err)
		return
	}
	response.Success(c, services)
}

func (h *Handler) GetService(c *gin.Context) {
	serviceID, err := parseUintParam(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid service id", err)
		return
	}

	service, err := h.service.GetService(c.Request.Context(), serviceID)
	if err != nil {
		response.HandleError(c, "Failed to load service", err)
		return
	}
	response.Success(c, service)
}

func (h *Handler) ImportService(c *gin.Context) {
	var req ImportServiceRequest
	if !httphandler.BindJSON(c, &req) {
		return
	}

	service, err := h.service.ImportService(c.Request.Context(), &req)
	if err != nil {
		response.HandleError(c, "Failed to import service", err)
		return
	}
	response.Created(c, ImportServiceResponse{Service: service})
}

func (h *Handler) UpdateService(c *gin.Context) {
	serviceID, err := parseUintParam(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid service id", err)
		return
	}

	var req UpdateServiceRequest
	if !httphandler.BindJSON(c, &req) {
		return
	}

	service, err := h.service.UpdateService(c.Request.Context(), serviceID, &req)
	if err != nil {
		response.HandleError(c, "Failed to update service", err)
		return
	}
	response.Success(c, service)
}

func (h *Handler) ReplaceEnvironment(c *gin.Context) {
	serviceID, err := parseUintParam(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid service id", err)
		return
	}

	var req ReplaceEnvironmentRequest
	if !httphandler.BindJSON(c, &req) {
		return
	}

	vars, err := h.service.ReplaceEnvironment(c.Request.Context(), serviceID, &req)
	if err != nil {
		response.HandleError(c, "Failed to update environment variables", err)
		return
	}
	response.Success(c, vars)
}

func (h *Handler) ListServiceDeployments(c *gin.Context) {
	serviceID, err := parseUintParam(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid service id", err)
		return
	}

	limit := httphandler.QueryInt(c, "limit", 20)
	deployments, err := h.service.ListServiceDeployments(c.Request.Context(), serviceID, limit)
	if err != nil {
		response.HandleError(c, "Failed to load service deployments", err)
		return
	}
	response.Success(c, deployments)
}

func (h *Handler) DeployService(c *gin.Context) {
	serviceID, err := parseUintParam(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid service id", err)
		return
	}

	var req TriggerServiceDeploymentRequest
	if c.Request.ContentLength > 0 {
		if !httphandler.BindJSON(c, &req) {
			return
		}
	}

	service, deployment, err := h.service.DeployService(c.Request.Context(), serviceID, &req)
	if err != nil {
		response.HandleError(c, "Failed to deploy service", err)
		return
	}

	response.Accepted(c, TriggerServiceDeploymentResponse{
		Service:    service,
		Deployment: deployment,
	})
}

func (h *Handler) HandleGitHubWebhook(c *gin.Context) {
	serviceID, err := parseUintParam(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid service id", err)
		return
	}

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.BadRequest(c, "Failed to read webhook payload", err)
		return
	}

	result, err := h.service.HandleGitHubWebhook(
		c.Request.Context(),
		serviceID,
		c.GetHeader("X-Hub-Signature-256"),
		payload,
	)
	if err != nil {
		response.HandleError(c, "Failed to process GitHub webhook", err)
		return
	}

	response.Accepted(c, result)
}

func (h *Handler) ListDeploymentLogs(c *gin.Context) {
	tail := httphandler.QueryInt(c, "tail", 300)
	logs, err := h.service.ListDeploymentLogs(c.Param("deploymentId"), tail)
	if err != nil {
		if errors.Is(err, deploycontrol.ErrDeploymentNotFound) {
			response.NotFound(c, "Deployment not found")
			return
		}
		response.HandleError(c, "Failed to load deployment logs", err)
		return
	}
	response.Success(c, logs)
}

func (h *Handler) StreamDeploymentLogs(c *gin.Context) {
	events, cancel, err := h.service.WatchDeployment(c.Param("deploymentId"))
	if err != nil {
		if errors.Is(err, deploycontrol.ErrDeploymentNotFound) {
			response.NotFound(c, "Deployment not found")
			return
		}
		response.HandleError(c, "Failed to stream deployment logs", err)
		return
	}
	defer cancel()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		response.InternalServerError(c, "Streaming is not supported", nil)
		return
	}

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			payload, marshalErr := json.Marshal(event)
			if marshalErr != nil {
				continue
			}
			_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
			flusher.Flush()
			if event.Done {
				return
			}
		}
	}
}

func parseUintParam(value string) (uint, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(parsed), nil
}
