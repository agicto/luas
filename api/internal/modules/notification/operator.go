package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/pagination"
	"github.com/zgiai/luas/api/pkg/response"
)

// OperatorDeliveryResponse is the platform-operator view of one channel delivery. It carries state
// and stable failure codes only: never the notification title, body, action URL, or destination.
type OperatorDeliveryResponse struct {
	ID              uint       `json:"id"`
	NotificationID  uint       `json:"notification_id"`
	UserID          uint       `json:"user_id"`
	Kind            string     `json:"kind"`
	Channel         string     `json:"channel"`
	Status          string     `json:"status"`
	Attempts        uint8      `json:"attempts"`
	LastFailureCode string     `json:"last_failure_code"`
	AvailableAt     time.Time  `json:"available_at"`
	DeliveredAt     *time.Time `json:"delivered_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type operatorDeliveryFilter struct {
	Status  string
	Channel string
	UserID  uint
}

type operatorStore interface {
	operatorDeliveries(context.Context, operatorDeliveryFilter, int, int) ([]OperatorDeliveryResponse, int64, error)
}

// OperatorHandler serves the delivery ledger to the starter that owns operator authorization. It
// applies no recipient check and must only be mounted behind that authorization.
type OperatorHandler struct {
	store operatorStore
}

// NewOperatorHandler creates the operator-facing notification delivery handler.
func NewOperatorHandler(repo *repository) *OperatorHandler {
	return &OperatorHandler{store: repo}
}

type operatorDeliveriesQuery struct {
	Status  string `form:"status" binding:"omitempty,oneof=pending processing delivered failed"`
	Channel string `form:"channel" binding:"omitempty,oneof=in_app email"`
	UserID  uint   `form:"user_id" binding:"omitempty,min=1"`
}

// ListDeliveries returns channel deliveries across all users, newest first.
func (h *OperatorHandler) ListDeliveries(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	var query operatorDeliveriesQuery
	if !handler.BindQuery(c, &query) {
		return
	}
	page := pagination.FromContext(c)
	items, total, err := h.store.operatorDeliveries(
		c.Request.Context(),
		operatorDeliveryFilter(query),
		page.GetPage(),
		page.GetPerPage(),
	)
	if err != nil {
		response.HandleError(c, "Failed to list notification deliveries", err)
		return
	}
	paginator := pagination.NewPaginator(items, total, page.GetPage(), page.GetPerPage())
	paginator.SetPath(c.Request.URL.Path).WithQuery(c.Request.URL.Query())
	response.Success(c, paginator)
}

func (r *repository) operatorDeliveries(
	ctx context.Context,
	filter operatorDeliveryFilter,
	page int,
	pageSize int,
) ([]OperatorDeliveryResponse, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, domain.ErrServiceUnavailable
	}
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return nil, 0, domain.ErrInvalidInput
	}
	query := r.db.WithContext(ctx).
		Table("notification_deliveries").
		Joins("JOIN notifications ON notifications.id = notification_deliveries.notification_id")
	if filter.Status != "" {
		query = query.Where("notification_deliveries.status = ?", filter.Status)
	}
	if filter.Channel != "" {
		query = query.Where("notification_deliveries.channel = ?", filter.Channel)
	}
	if filter.UserID != 0 {
		query = query.Where("notifications.user_id = ?", filter.UserID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count notification deliveries: %w", err)
	}
	var rows []OperatorDeliveryResponse
	if err := query.
		Select(`notification_deliveries.id, notification_deliveries.notification_id, notifications.user_id,
			notifications.kind, notification_deliveries.channel, notification_deliveries.status,
			notification_deliveries.attempts, notification_deliveries.last_failure_code,
			notification_deliveries.available_at, notification_deliveries.delivered_at,
			notification_deliveries.created_at, notification_deliveries.updated_at`).
		Order("notification_deliveries.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list notification deliveries: %w", err)
	}
	if rows == nil {
		rows = []OperatorDeliveryResponse{}
	}
	return rows, total, nil
}
