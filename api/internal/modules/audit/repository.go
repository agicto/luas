package audit

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
)

type repository struct {
	db *gorm.DB
}

var _ domain.AuditLogRepository = (*repository)(nil)

// NewRepository creates a new audit repository.
func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) withContext(ctx context.Context) (*gorm.DB, error) {
	if r == nil || r.db == nil {
		return nil, domain.ErrServiceUnavailable
	}
	return r.db.WithContext(ctx), nil
}

func (r *repository) Create(ctx context.Context, entry *domain.AuditLog) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	po := newAuditLogPO(entry)
	if err := db.Create(po).Error; err != nil {
		return err
	}

	entry.ID = po.ID
	entry.CreatedAt = po.CreatedAt
	entry.UpdatedAt = po.UpdatedAt
	return nil
}

// CreateBatch inserts records in one multi-row statement.
func (r *repository) CreateBatch(ctx context.Context, entries []*domain.AuditLog) error {
	if len(entries) == 0 {
		return nil
	}
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	rows := make([]*AuditLogPO, len(entries))
	for index, entry := range entries {
		rows[index] = newAuditLogPO(entry)
	}
	if err := db.Create(&rows).Error; err != nil {
		return err
	}
	for index, row := range rows {
		entries[index].ID = row.ID
		entries[index].CreatedAt = row.CreatedAt
		entries[index].UpdatedAt = row.UpdatedAt
	}
	return nil
}

// FindByUserIDAfter pages one user's history newest first on the (user_id, id) index.
func (r *repository) FindByUserIDAfter(
	ctx context.Context,
	userID uint,
	filter domain.AuditLogFilter,
	after *domain.AuditLogCursor,
	limit int,
) ([]*domain.AuditLog, *domain.AuditLogCursor, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	query := applyAuditFilter(db.Model(&AuditLogPO{}).Where("user_id = ?", userID), filter)
	if after != nil {
		query = query.Where("id < ?", after.ID)
	}
	return findAuditSlice(query.Order("id DESC"), limit)
}

// FindAllAfter pages platform-wide history newest first on the (created_at, id) index.
func (r *repository) FindAllAfter(
	ctx context.Context,
	filter domain.AuditLogFilter,
	after *domain.AuditLogCursor,
	limit int,
) ([]*domain.AuditLog, *domain.AuditLogCursor, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	query := platformAuditQuery(db, filter)
	if after != nil {
		query = query.Where("(created_at, id) < (?, ?)", after.CreatedAt.UTC(), after.ID)
	}
	return findAuditSlice(query.Order("created_at DESC, id DESC"), limit)
}

func platformAuditQuery(db *gorm.DB, filter domain.AuditLogFilter) *gorm.DB {
	query := db.Model(&AuditLogPO{})
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if !filter.From.IsZero() {
		query = query.Where("created_at >= ?", filter.From.UTC())
	}
	if !filter.To.IsZero() {
		query = query.Where("created_at < ?", filter.To.UTC())
	}
	return applyAuditFilter(query, filter)
}

// findAuditSlice reads one extra row to learn whether an older page exists, so no count is needed.
func findAuditSlice(query *gorm.DB, limit int) ([]*domain.AuditLog, *domain.AuditLogCursor, error) {
	var rows []AuditLogPO
	if err := query.Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, nil, err
	}
	var next *domain.AuditLogCursor
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1]
		next = &domain.AuditLogCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	items := make([]*domain.AuditLog, len(rows))
	for i := range rows {
		items[i] = rows[i].toDomain()
	}
	return items, next, nil
}

func applyAuditFilter(query *gorm.DB, filter domain.AuditLogFilter) *gorm.DB {
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
	return query
}

// PruneBefore deletes one deterministic PostgreSQL batch without holding an unbounded transaction.
func (r *repository) PruneBefore(ctx context.Context, before time.Time, batch int) (int64, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return 0, err
	}

	result := db.Exec(`
		WITH candidates AS (
			SELECT id
			FROM audit_logs
			WHERE created_at < ?
			ORDER BY created_at ASC, id ASC
			LIMIT ?
			FOR UPDATE SKIP LOCKED
		)
		DELETE FROM audit_logs
		USING candidates
		WHERE audit_logs.id = candidates.id
	`, before.UTC(), batch)
	return result.RowsAffected, result.Error
}
