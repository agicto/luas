package notification

import (
	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/starter/assembly"
	httphandler "github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/pagination"
	"github.com/zgiai/luas/api/pkg/response"
)

// Handler owns the authenticated notification-center HTTP boundary.
type Handler struct {
	service Service
}

var (
	_ assembly.Module      = (*Handler)(nil)
	_ assembly.RouteModule = (*Handler)(nil)
)

// NewHandler creates the notification handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Name() string { return "notification" }

func (h *Handler) List(c *gin.Context) {
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	status := c.DefaultQuery("status", notificationFilterAll)
	request := pagination.FromContext(c)
	items, total, err := h.service.ListForUser(
		c.Request.Context(),
		userID,
		status,
		request.GetPage(),
		request.GetPerPage(),
	)
	if err != nil {
		response.HandleError(c, "Failed to list notifications", err)
		return
	}

	responses := make([]*NotificationResponse, len(items))
	for index := range items {
		responses[index] = toNotificationResponse(items[index])
	}
	paginator := pagination.NewPaginator(responses, total, request.GetPage(), request.GetPerPage())
	paginator.SetPath(c.Request.URL.Path)
	paginator.Append("status", status)
	response.Success(c, paginator)
}

func (h *Handler) Status(c *gin.Context) {
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	count, err := h.service.UnreadCount(c.Request.Context(), userID)
	if err != nil {
		response.HandleError(c, "Failed to read notification status", err)
		return
	}
	response.Success(c, &NotificationStatusResponse{UnreadCount: count})
}

func (h *Handler) ReplaceReadState(c *gin.Context) {
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	notificationID, ok := httphandler.ParseID(c, "id")
	if !ok {
		return
	}
	var request replaceNotificationReadStateRequest
	if !httphandler.BindJSON(c, &request) {
		return
	}
	notification, err := h.service.ReplaceReadState(
		c.Request.Context(),
		userID,
		notificationID,
		*request.IsRead,
	)
	if err != nil {
		response.HandleError(c, "Failed to update notification read state", err)
		return
	}
	response.Success(c, toNotificationResponse(notification))
}

func (h *Handler) MarkReadThrough(c *gin.Context) {
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	var request replaceNotificationReadStateThroughRequest
	if !httphandler.BindJSON(c, &request) {
		return
	}
	updated, unread, err := h.service.MarkReadThrough(
		c.Request.Context(),
		userID,
		request.ThroughID,
	)
	if err != nil {
		response.HandleError(c, "Failed to update notification read state", err)
		return
	}
	response.Success(c, &NotificationReadStateResponse{
		UpdatedCount: updated,
		UnreadCount:  unread,
	})
}

func (h *Handler) GetPreference(c *gin.Context) {
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	preference, err := h.service.Preference(c.Request.Context(), userID)
	if err != nil {
		response.HandleError(c, "Failed to read notification preferences", err)
		return
	}
	response.Success(c, toNotificationPreferenceResponse(preference))
}

func (h *Handler) ReplacePreference(c *gin.Context) {
	userID, ok := httphandler.GetUserID(c)
	if !ok {
		return
	}
	var request replaceNotificationPreferenceRequest
	if !httphandler.BindJSON(c, &request) {
		return
	}
	preference, err := h.service.ReplacePreference(
		c.Request.Context(),
		userID,
		*request.InAppEnabled,
		*request.EmailEnabled,
	)
	if err != nil {
		response.HandleError(c, "Failed to update notification preferences", err)
		return
	}
	response.Success(c, toNotificationPreferenceResponse(preference))
}
