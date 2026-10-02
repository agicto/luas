package webhook

import (
	"strconv"

	"github.com/gin-gonic/gin"

	httphandler "github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/pagination"
	"github.com/zgiai/luas/api/pkg/response"
)

// OperatorHandler serves secret-free webhook diagnosis and delivery replay for one organization
// named in the path. The starter that mounts it owns operator authorization and the organization
// lookup; this handler applies no membership check.
type OperatorHandler struct {
	service *service
}

// NewOperatorHandler creates the operator-facing webhook handler.
func NewOperatorHandler(service *service) *OperatorHandler {
	return &OperatorHandler{service: service}
}

// ListEndpoints returns the endpoints of the path organization.
func (h *OperatorHandler) ListEndpoints(c *gin.Context) {
	organizationID, ok := operatorOrganizationID(c)
	if !ok {
		return
	}
	page := pagination.FromContext(c)
	values, total, err := h.service.ListEndpoints(c.Request.Context(), organizationID, page.GetPage(), page.GetPerPage())
	if err != nil {
		response.HandleError(c, "Failed to load webhook endpoints", err)
		return
	}
	paginator := pagination.NewPaginator(toEndpointResponses(values), total, page.GetPage(), page.GetPerPage())
	paginator.SetPath(c.Request.URL.Path)
	response.Success(c, paginator)
}

// ListDeliveries returns the deliveries of the path organization, filtered by endpoint or status.
func (h *OperatorHandler) ListDeliveries(c *gin.Context) {
	organizationID, ok := operatorOrganizationID(c)
	if !ok {
		return
	}
	filter, ok := webhookDeliveryFilter(c)
	if !ok {
		return
	}
	page := pagination.FromContext(c)
	values, total, err := h.service.ListDeliveries(
		c.Request.Context(),
		organizationID,
		filter,
		page.GetPage(),
		page.GetPerPage(),
	)
	if err != nil {
		response.HandleError(c, "Failed to load webhook deliveries", err)
		return
	}
	paginator := pagination.NewPaginator(toDeliveryResponses(values), total, page.GetPage(), page.GetPerPage())
	paginator.SetPath(c.Request.URL.Path).WithQuery(c.Request.URL.Query())
	response.Success(c, paginator)
}

// ListAttempts returns the attempts of one delivery owned by the path organization.
func (h *OperatorHandler) ListAttempts(c *gin.Context) {
	organizationID, ok := operatorOrganizationID(c)
	if !ok {
		return
	}
	deliveryID, ok := operatorDeliveryID(c)
	if !ok {
		return
	}
	page := pagination.FromContext(c)
	values, total, err := h.service.ListAttempts(
		c.Request.Context(),
		organizationID,
		deliveryID,
		page.GetPage(),
		page.GetPerPage(),
	)
	if err != nil {
		response.HandleError(c, "Failed to load webhook delivery attempts", err)
		return
	}
	paginator := pagination.NewPaginator(toAttemptResponses(values), total, page.GetPage(), page.GetPerPage())
	paginator.SetPath(c.Request.URL.Path)
	response.Success(c, paginator)
}

// ReplayDelivery queues one terminal delivery again with the operator as the audited actor.
func (h *OperatorHandler) ReplayDelivery(c *gin.Context) {
	organizationID, ok := operatorOrganizationID(c)
	if !ok {
		return
	}
	deliveryID, ok := operatorDeliveryID(c)
	if !ok {
		return
	}
	actorID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	delivery, err := h.service.ReplayWebhookDelivery(c.Request.Context(), organizationID, deliveryID, actorID)
	if err != nil {
		response.HandleError(c, "Failed to replay webhook delivery", err)
		return
	}
	response.Success(c, toDeliveryResponse(delivery))
}

func operatorOrganizationID(c *gin.Context) (uint, bool) {
	setPrivateWebhookResponse(c)
	return httphandler.ParseID(c, "id")
}

func operatorDeliveryID(c *gin.Context) (uint64, bool) {
	value := c.Param("delivery_id")
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 || strconv.FormatUint(parsed, 10) != value {
		response.BadRequest(c, "Invalid delivery ID")
		return 0, false
	}
	return parsed, true
}
