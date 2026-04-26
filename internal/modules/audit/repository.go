package audit

import (
	"context"
	"strings"

	"github.com/zgiai/zgo/internal/domain"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

var _ domain.AuditLogRepository = (*repository)(nil)

// NewRepository creates a new audit repository.
func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, entry *domain.AuditLog) error {
	po := newAuditLogPO(entry)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}

	entry.ID = po.ID
	entry.CreatedAt = po.CreatedAt
	entry.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *repository) FindByUserID(ctx context.Context, userID uint, filter domain.AuditLogFilter, page, pageSize int) ([]*domain.AuditLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 15
	}

	var (
		rows  []AuditLogPO
		total int64
	)

	query := r.db.WithContext(ctx).Model(&AuditLogPO{}).Where("user_id = ?", userID)

	if action := strings.TrimSpace(filter.Action); action != "" {
		query = query.Where("action = ?", action)
	}
	if resource := strings.TrimSpace(filter.Resource); resource != "" {
		query = query.Where("resource = ?", resource)
	}
	if method := strings.TrimSpace(filter.Method); method != "" {
		query = query.Where("method = ?", strings.ToUpper(method))
	}
	if requestID := strings.TrimSpace(filter.RequestID); requestID != "" {
		query = query.Where("request_id = ?", requestID)
	}
	if filter.StatusCode > 0 {
		query = query.Where("status_code = ?", filter.StatusCode)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	items := make([]*domain.AuditLog, len(rows))
	for i := range rows {
		items[i] = rows[i].toDomain()
	}
	return items, total, nil
}
