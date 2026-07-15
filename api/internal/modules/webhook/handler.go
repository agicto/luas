package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/starter/assembly"
	httphandler "github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/pagination"
	"github.com/zgiai/luas/api/pkg/response"
)

const maxWebhookManagementBodyBytes = 16 * 1024

var webhookEndpointETagPattern = regexp.MustCompile(`^"webhook-endpoint-v([1-9][0-9]{0,19})"$`)

// Handler owns the organization-scoped webhook management HTTP boundary.
type Handler struct {
	service Service
}

var (
	_ assembly.Module      = (*Handler)(nil)
	_ assembly.RouteModule = (*Handler)(nil)
)

// NewHandler creates the optional webhook HTTP boundary.
func NewHandler(service *service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Name() string { return "webhook" }

// EventTypes returns the finite code-owned subscription catalog.
// luas:bounded-list max=128 reason=finite-code-owned-catalog
func (h *Handler) EventTypes(c *gin.Context) {
	organization, _, ok := webhookManager(c)
	if !ok {
		return
	}
	_ = organization
	values, err := h.service.EventTypes(c.Request.Context())
	if err != nil {
		response.HandleError(c, "Failed to load webhook event types", err)
		return
	}
	response.Success(c, values)
}

// ListEndpoints returns paginated endpoints for the active organization.
func (h *Handler) ListEndpoints(c *gin.Context) {
	organization, _, ok := webhookManager(c)
	if !ok {
		return
	}
	page := pagination.FromContext(c)
	values, total, err := h.service.ListEndpoints(
		c.Request.Context(),
		organization.OrganizationID,
		page.GetPage(),
		page.GetPerPage(),
	)
	if err != nil {
		response.HandleError(c, "Failed to load webhook endpoints", err)
		return
	}
	paginator := pagination.NewPaginator(toEndpointResponses(values), total, page.GetPage(), page.GetPerPage())
	paginator.SetPath(c.Request.URL.Path)
	response.Success(c, paginator)
}

// CreateEndpoint creates an endpoint and returns its secret exactly once.
func (h *Handler) CreateEndpoint(c *gin.Context) {
	organization, actorID, ok := webhookManager(c)
	if !ok {
		return
	}
	request, ok := bindWebhookJSON[endpointRequest](c)
	if !ok {
		return
	}
	value, err := h.service.CreateEndpoint(c.Request.Context(), organization.OrganizationID, actorID, request.input())
	if err != nil {
		response.HandleError(c, "Failed to create webhook endpoint", err)
		return
	}
	c.Header("ETag", webhookEndpointETag(value.Endpoint.Version))
	response.Created(c, toEndpointSecretResponse(value))
}

// UpdateEndpoint replaces one endpoint's mutable configuration using CAS.
func (h *Handler) UpdateEndpoint(c *gin.Context) {
	organization, actorID, endpointID, version, ok := webhookEndpointMutationContext(c)
	if !ok {
		return
	}
	request, ok := bindWebhookJSON[endpointRequest](c)
	if !ok {
		return
	}
	value, err := h.service.UpdateEndpoint(
		c.Request.Context(),
		organization.OrganizationID,
		endpointID,
		actorID,
		version,
		request.input(),
	)
	if err != nil {
		response.HandleError(c, "Failed to update webhook endpoint", err)
		return
	}
	c.Header("ETag", webhookEndpointETag(value.Version))
	response.Success(c, toEndpointResponse(value))
}

// ReplaceEndpointStatus explicitly enables or disables an endpoint using CAS.
func (h *Handler) ReplaceEndpointStatus(c *gin.Context) {
	organization, actorID, endpointID, version, ok := webhookEndpointMutationContext(c)
	if !ok {
		return
	}
	request, ok := bindWebhookJSON[endpointStatusRequest](c)
	if !ok {
		return
	}
	if request.Enabled == nil {
		response.BadRequest(c, "Invalid webhook endpoint status")
		return
	}
	value, err := h.service.ReplaceEndpointStatus(
		c.Request.Context(),
		organization.OrganizationID,
		endpointID,
		actorID,
		version,
		*request.Enabled,
	)
	if err != nil {
		response.HandleError(c, "Failed to replace webhook endpoint status", err)
		return
	}
	c.Header("ETag", webhookEndpointETag(value.Version))
	response.Success(c, toEndpointResponse(value))
}

// DeleteEndpoint scrubs signing material and cancels open work using CAS.
func (h *Handler) DeleteEndpoint(c *gin.Context) {
	organization, actorID, endpointID, version, ok := webhookEndpointMutationContext(c)
	if !ok {
		return
	}
	if err := h.service.DeleteEndpoint(
		c.Request.Context(),
		organization.OrganizationID,
		endpointID,
		actorID,
		version,
	); err != nil {
		response.HandleError(c, "Failed to delete webhook endpoint", err)
		return
	}
	response.NoContent(c)
}

// RotateEndpointSecret replaces signing material and returns plaintext once.
func (h *Handler) RotateEndpointSecret(c *gin.Context) {
	organization, actorID, endpointID, version, ok := webhookEndpointMutationContext(c)
	if !ok {
		return
	}
	value, err := h.service.RotateEndpointSecret(
		c.Request.Context(),
		organization.OrganizationID,
		endpointID,
		actorID,
		version,
	)
	if err != nil {
		response.HandleError(c, "Failed to rotate webhook endpoint secret", err)
		return
	}
	c.Header("ETag", webhookEndpointETag(value.Endpoint.Version))
	response.Created(c, toEndpointSecretResponse(value))
}

// TestEndpoint queues only the starter-owned test event for one endpoint.
func (h *Handler) TestEndpoint(c *gin.Context) {
	organization, actorID, ok := webhookManager(c)
	if !ok {
		return
	}
	endpointID, ok := httphandler.ParseID(c, "id")
	if !ok {
		return
	}
	idempotencyKey, ok := webhookIdempotencyKey(c)
	if !ok {
		return
	}
	value, err := h.service.PublishWebhookTest(
		c.Request.Context(),
		organization.OrganizationID,
		endpointID,
		actorID,
		idempotencyKey,
	)
	if err != nil {
		response.HandleError(c, "Failed to queue webhook endpoint test", err)
		return
	}
	response.Accepted(c, toDeliveryResponse(value))
}

// ListDeliveries returns a privacy-minimized delivery ledger.
func (h *Handler) ListDeliveries(c *gin.Context) {
	organization, _, ok := webhookManager(c)
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
		organization.OrganizationID,
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

// ListAttempts returns minimized attempts for one organization-owned delivery.
func (h *Handler) ListAttempts(c *gin.Context) {
	organization, _, ok := webhookManager(c)
	if !ok {
		return
	}
	deliveryID, ok := parseWebhookDeliveryID(c)
	if !ok {
		return
	}
	page := pagination.FromContext(c)
	values, total, err := h.service.ListAttempts(
		c.Request.Context(),
		organization.OrganizationID,
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

func webhookManager(c *gin.Context) (domain.OrganizationContext, uint, bool) {
	setPrivateWebhookResponse(c)
	actorID, ok := httphandler.GetUserID(c)
	if !ok {
		return domain.OrganizationContext{}, 0, false
	}
	organization, ok := domain.OrganizationContextFromContext(c.Request.Context())
	if !ok {
		response.HandleError(c, "Organization context required", domain.ErrOrganizationContextRequired)
		return domain.OrganizationContext{}, 0, false
	}
	if organization.UserID != actorID || !organization.Role.CanManageOrganization() {
		response.HandleError(c, "Webhook management forbidden", domain.ErrPermissionDenied)
		return domain.OrganizationContext{}, 0, false
	}
	return organization, actorID, true
}

func webhookEndpointMutationContext(
	c *gin.Context,
) (domain.OrganizationContext, uint, uint, uint64, bool) {
	organization, actorID, ok := webhookManager(c)
	if !ok {
		return domain.OrganizationContext{}, 0, 0, 0, false
	}
	endpointID, ok := httphandler.ParseID(c, "id")
	if !ok {
		return domain.OrganizationContext{}, 0, 0, 0, false
	}
	version, ok := expectedWebhookEndpointVersion(c)
	if !ok {
		return domain.OrganizationContext{}, 0, 0, 0, false
	}
	return organization, actorID, endpointID, version, true
}

func expectedWebhookEndpointVersion(c *gin.Context) (uint64, bool) {
	values := c.Request.Header.Values("If-Match")
	if len(values) == 0 {
		response.HandleError(c, "Webhook endpoint version precondition required", domain.ErrWebhookPreconditionRequired)
		return 0, false
	}
	if len(values) != 1 || strings.Contains(values[0], ",") {
		response.ErrorWithCode(c, http.StatusBadRequest, domain.CodeInvalidInput, "Invalid If-Match header")
		return 0, false
	}
	match := webhookEndpointETagPattern.FindStringSubmatch(values[0])
	if len(match) != 2 {
		response.ErrorWithCode(c, http.StatusBadRequest, domain.CodeInvalidInput, "Invalid If-Match header")
		return 0, false
	}
	version, err := strconv.ParseUint(match[1], 10, 64)
	if err != nil || version == 0 {
		response.ErrorWithCode(c, http.StatusBadRequest, domain.CodeInvalidInput, "Invalid If-Match header")
		return 0, false
	}
	return version, true
}

func webhookEndpointETag(version uint64) string {
	return `"webhook-endpoint-v` + strconv.FormatUint(version, 10) + `"`
}

func webhookIdempotencyKey(c *gin.Context) (string, bool) {
	values := c.Request.Header.Values("Idempotency-Key")
	if len(values) != 1 || values[0] == "" || values[0] != strings.TrimSpace(values[0]) || strings.Contains(values[0], ",") {
		response.ErrorWithCode(c, http.StatusBadRequest, domain.CodeInvalidInput, "Valid Idempotency-Key header required")
		return "", false
	}
	return values[0], true
}

func webhookDeliveryFilter(c *gin.Context) (deliveryFilter, bool) {
	filter := deliveryFilter{}
	if value := c.Query("endpoint_id"); value != "" {
		parsed, err := strconv.ParseUint(value, 10, strconv.IntSize)
		if err != nil || parsed == 0 || strconv.FormatUint(parsed, 10) != value {
			response.ErrorWithCode(c, http.StatusBadRequest, domain.CodeInvalidInput, "Invalid endpoint_id filter")
			return deliveryFilter{}, false
		}
		filter.EndpointID = uint(parsed)
	}
	if value := c.Query("status"); value != "" {
		filter.Status = domain.WebhookDeliveryStatus(value)
		if !validDeliveryStatus(filter.Status) {
			response.ErrorWithCode(c, http.StatusBadRequest, domain.CodeInvalidInput, "Invalid status filter")
			return deliveryFilter{}, false
		}
	}
	return filter, true
}

func parseWebhookDeliveryID(c *gin.Context) (uint64, bool) {
	value := c.Param("id")
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 || strconv.FormatUint(parsed, 10) != value {
		response.BadRequest(c, "Invalid delivery ID")
		return 0, false
	}
	return parsed, true
}

func bindWebhookJSON[T any](c *gin.Context) (T, bool) {
	var request T
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxWebhookManagementBodyBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		response.BadRequest(c, "Invalid webhook request", err)
		return request, false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		response.BadRequest(c, "Invalid webhook request", fmt.Errorf("request must contain exactly one JSON object"))
		return request, false
	}
	return request, true
}

func setPrivateWebhookResponse(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Vary", "Authorization, Organization-Id")
}
