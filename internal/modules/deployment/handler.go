package deployment

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zgiai/zgo/internal/contracts"
	"github.com/zgiai/zgo/internal/infra/deploycontrol"
	httphandler "github.com/zgiai/zgo/pkg/handler"
	"github.com/zgiai/zgo/pkg/response"
)

type Handler struct {
	contracts.BaseModule
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Name() string {
	return "deployment"
}

func (h *Handler) ListTargets(c *gin.Context) {
	targets, err := h.service.ListTargets()
	if err != nil {
		response.HandleError(c, "Failed to list deployment targets", err)
		return
	}

	response.Success(c, targets)
}

func (h *Handler) ListDeployments(c *gin.Context) {
	limit := httphandler.QueryInt(c, "limit", 20)
	deployments, err := h.service.ListDeployments(limit)
	if err != nil {
		response.HandleError(c, "Failed to list deployments", err)
		return
	}

	response.Success(c, deployments)
}

func (h *Handler) GetDeployment(c *gin.Context) {
	deployment, err := h.service.GetDeployment(c.Param("id"))
	if err != nil {
		if errors.Is(err, deploycontrol.ErrDeploymentNotFound) {
			response.NotFound(c, "Deployment not found")
			return
		}
		response.HandleError(c, "Failed to load deployment", err)
		return
	}

	response.Success(c, deployment)
}

func (h *Handler) CreateDeployment(c *gin.Context) {
	var req RunDeploymentRequest
	if !httphandler.BindJSON(c, &req) {
		return
	}

	deployment, err := h.service.StartDeployment(c.Request.Context(), deploycontrol.RunRequest{
		Target:      req.Target,
		Branch:      req.Branch,
		Commit:      req.Commit,
		TriggeredBy: req.TriggeredBy,
		TriggerMode: "api",
		Environment: req.Environment,
	})
	if err != nil {
		response.HandleError(c, "Failed to start deployment", err)
		return
	}

	response.Accepted(c, RunDeploymentResponse{Deployment: deployment})
}

func (h *Handler) ListLogs(c *gin.Context) {
	tail := httphandler.QueryInt(c, "tail", 200)
	logs, err := h.service.ListLogs(c.Param("id"), tail)
	if err != nil {
		response.HandleError(c, "Failed to load deployment logs", err)
		return
	}

	response.Success(c, logs)
}

func (h *Handler) StreamLogs(c *gin.Context) {
	events, cancel, err := h.service.Watch(c.Param("id"))
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

			payload, err := json.Marshal(event)
			if err != nil {
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

func (h *Handler) GenerateCertificate(c *gin.Context) {
	var req GenerateCertificateRequest
	if !httphandler.BindJSON(c, &req) {
		return
	}

	certificate, err := h.service.GenerateCertificate(deploycontrol.CertificateRequest{
		Domain:    req.Domain,
		ValidDays: req.ValidDays,
	})
	if err != nil {
		response.HandleError(c, "Failed to generate certificate", err)
		return
	}

	response.Created(c, certificate)
}

func (h *Handler) TriggerWebhook(c *gin.Context) {
	var req TriggerWebhookRequest
	if !httphandler.BindJSON(c, &req) {
		return
	}

	deployment, err := h.service.HandleWebhook(
		c.Request.Context(),
		c.Param("target"),
		c.GetHeader("X-Deploy-Secret"),
		req,
	)
	if err != nil {
		response.HandleError(c, "Failed to process deployment webhook", err)
		return
	}

	response.Accepted(c, RunDeploymentResponse{Deployment: deployment})
}
