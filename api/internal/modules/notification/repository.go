package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/modules/user"
)

type publicationResult struct {
	Notification *domain.Notification
	Created      bool
}

type claimedEmailDelivery struct {
	ID                 uint
	NotificationID     uint
	LeaseToken         string
	Attempts           uint8
	UserID             uint
	RecipientEmail     string
	RecipientDeletedAt *time.Time
	DestinationHash    string
	Title              string
	Body               string
	ActionURL          string
}

type notificationStore interface {
	CreatePublication(context.Context, domain.NotificationPublication, string, time.Time) (*publicationResult, error)
	ListForUser(context.Context, uint, string, int, int) ([]*domain.Notification, int64, error)
	UnreadCount(context.Context, uint) (int64, error)
	ReplaceReadState(context.Context, uint, uint, bool, time.Time) (*domain.Notification, error)
	MarkReadThrough(context.Context, uint, uint, time.Time) (int64, int64, error)
	Preference(context.Context, uint) (*domain.NotificationPreference, error)
	ReplacePreference(context.Context, *domain.NotificationPreference) error
	FailExhausted(context.Context, time.Time, uint8) (int64, error)
	ClaimDueEmail(context.Context, time.Time, time.Duration, uint8, int) ([]claimedEmailDelivery, error)
	BindEmailDestination(context.Context, uint, string, string) (bool, error)
	CompleteEmail(context.Context, uint, string, deliveryCompletion) (bool, error)
}

type deliveryCompletion struct {
	Status      deliveryStatus
	CompletedAt time.Time
	RetryAt     *time.Time
	FailureCode string
}

type repository struct {
	db *gorm.DB
}

var _ notificationStore = (*repository)(nil)

// NewRepository creates the notification persistence adapter.
func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) CreatePublication(
	ctx context.Context,
	publication domain.NotificationPublication,
	publicationHash string,
	now time.Time,
) (*publicationResult, error) {
	if r == nil || r.db == nil {
		return nil, domain.ErrServiceUnavailable
	}

	var result publicationResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var recipient user.UserPO
		if err := tx.Select("id").First(&recipient, publication.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrUserNotFound
			}
			return fmt.Errorf("find notification recipient: %w", err)
		}

		preference, err := findPreference(tx, publication.UserID)
		if err != nil {
			return err
		}
		channels, err := selectPublicationChannels(publication, preference)
		if err != nil {
			return err
		}

		po := NotificationPO{
			UserID:          publication.UserID,
			IdempotencyKey:  publication.IdempotencyKey,
			PublicationHash: publicationHash,
			Kind:            publication.Kind,
			Title:           publication.Title,
			Body:            publication.Body,
			ActionURL:       publication.ActionURL,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		create := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "idempotency_key"}},
			DoNothing: true,
		}).Create(&po)
		if create.Error != nil {
			return fmt.Errorf("create notification: %w", create.Error)
		}
		if create.RowsAffected == 0 {
			var existing NotificationPO
			if err := tx.Where("user_id = ? AND idempotency_key = ?", publication.UserID, publication.IdempotencyKey).
				First(&existing).Error; err != nil {
				return fmt.Errorf("find idempotent notification: %w", err)
			}
			if existing.PublicationHash != publicationHash {
				return domain.ErrNotificationIdempotencyConflict
			}
			result.Notification = notificationFromPO(&existing)
			return nil
		}

		deliveries := make([]NotificationDeliveryPO, 0, len(channels))
		for _, channel := range channels {
			delivery := NotificationDeliveryPO{
				NotificationID: po.ID,
				Channel:        string(channel),
				Status:         string(deliveryStatusPending),
				AvailableAt:    now,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if channel == domain.NotificationChannelInApp {
				delivery.Status = string(deliveryStatusDelivered)
				delivery.DeliveredAt = cloneTime(&now)
			}
			deliveries = append(deliveries, delivery)
		}
		if len(deliveries) > 0 {
			if err := tx.Create(&deliveries).Error; err != nil {
				return fmt.Errorf("create notification deliveries: %w", err)
			}
		}

		result.Notification = notificationFromPO(&po)
		result.Created = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *repository) ListForUser(
	ctx context.Context,
	userID uint,
	status string,
	page int,
	pageSize int,
) ([]*domain.Notification, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, domain.ErrServiceUnavailable
	}

	query := visibleNotifications(r.db.WithContext(ctx), userID)
	if status == notificationFilterUnread {
		query = query.Where("notifications.read_at IS NULL")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	var rows []NotificationPO
	offset := (page - 1) * pageSize
	if err := query.Select("notifications.*").
		Order("notifications.id DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}

	items := make([]*domain.Notification, len(rows))
	for index := range rows {
		items[index] = notificationFromPO(&rows[index])
	}
	return items, total, nil
}

func (r *repository) UnreadCount(ctx context.Context, userID uint) (int64, error) {
	if r == nil || r.db == nil {
		return 0, domain.ErrServiceUnavailable
	}
	var count int64
	if err := visibleNotifications(r.db.WithContext(ctx), userID).
		Where("notifications.read_at IS NULL").
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

func (r *repository) ReplaceReadState(
	ctx context.Context,
	userID uint,
	notificationID uint,
	isRead bool,
	now time.Time,
) (*domain.Notification, error) {
	if r == nil || r.db == nil {
		return nil, domain.ErrServiceUnavailable
	}
	var readAt any
	if isRead {
		readAt = now
	}
	visibility := inAppVisibilitySubquery(r.db, userID)
	update := r.db.WithContext(ctx).Model(&NotificationPO{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Where("EXISTS (?)", visibility).
		Updates(map[string]any{"read_at": readAt, "updated_at": now})
	if update.Error != nil {
		return nil, fmt.Errorf("replace notification read state: %w", update.Error)
	}
	if update.RowsAffected == 0 {
		return nil, domain.ErrNotificationNotFound
	}

	var po NotificationPO
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", notificationID, userID).First(&po).Error; err != nil {
		return nil, fmt.Errorf("reload notification read state: %w", err)
	}
	return notificationFromPO(&po), nil
}

func (r *repository) MarkReadThrough(
	ctx context.Context,
	userID uint,
	throughID uint,
	now time.Time,
) (int64, int64, error) {
	if r == nil || r.db == nil {
		return 0, 0, domain.ErrServiceUnavailable
	}

	var updated int64
	var unread int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		visibility := inAppVisibilitySubquery(tx, userID)
		result := tx.Model(&NotificationPO{}).
			Where("user_id = ? AND id <= ? AND read_at IS NULL", userID, throughID).
			Where("EXISTS (?)", visibility).
			Updates(map[string]any{"read_at": now, "updated_at": now})
		if result.Error != nil {
			return fmt.Errorf("mark notifications read: %w", result.Error)
		}
		updated = result.RowsAffected
		if err := visibleNotifications(tx, userID).
			Where("notifications.read_at IS NULL").
			Count(&unread).Error; err != nil {
			return fmt.Errorf("count unread notifications: %w", err)
		}
		return nil
	})
	return updated, unread, err
}

func (r *repository) Preference(ctx context.Context, userID uint) (*domain.NotificationPreference, error) {
	if r == nil || r.db == nil {
		return nil, domain.ErrServiceUnavailable
	}
	preference, err := findPreference(r.db.WithContext(ctx), userID)
	if err != nil {
		return nil, err
	}
	return preference, nil
}

func (r *repository) ReplacePreference(ctx context.Context, preference *domain.NotificationPreference) error {
	if r == nil || r.db == nil {
		return domain.ErrServiceUnavailable
	}
	if preference == nil {
		return domain.ErrInvalidInput
	}

	po := NotificationPreferencePO{
		UserID:       preference.UserID,
		InAppEnabled: preference.InAppEnabled,
		EmailEnabled: preference.EmailEnabled,
		CreatedAt:    preference.CreatedAt,
		UpdatedAt:    preference.UpdatedAt,
	}
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"in_app_enabled",
				"email_enabled",
				"updated_at",
			}),
		}).
		Select("user_id", "in_app_enabled", "email_enabled", "created_at", "updated_at").
		Create(&po).Error
	if err != nil {
		return fmt.Errorf("replace notification preference: %w", err)
	}
	return nil
}

func (r *repository) FailExhausted(ctx context.Context, now time.Time, maxAttempts uint8) (int64, error) {
	if r == nil || r.db == nil {
		return 0, domain.ErrServiceUnavailable
	}
	result := r.db.WithContext(ctx).Model(&NotificationDeliveryPO{}).
		Where("channel = ? AND attempts >= ?", domain.NotificationChannelEmail, maxAttempts).
		Where(
			"status = ? OR (status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?)",
			deliveryStatusPending,
			deliveryStatusProcessing,
			now,
		).
		Updates(map[string]any{
			"status":            deliveryStatusFailed,
			"lease_token":       "",
			"lease_expires_at":  nil,
			"last_failure_code": failureRetryExhausted,
			"updated_at":        now,
		})
	if result.Error != nil {
		return 0, fmt.Errorf("fail exhausted notification deliveries: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *repository) ClaimDueEmail(
	ctx context.Context,
	now time.Time,
	leaseDuration time.Duration,
	maxAttempts uint8,
	limit int,
) ([]claimedEmailDelivery, error) {
	if r == nil || r.db == nil {
		return nil, domain.ErrServiceUnavailable
	}

	claimed := make([]claimedEmailDelivery, 0, limit)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Where("channel = ? AND attempts < ?", domain.NotificationChannelEmail, maxAttempts).
			Where(
				"(status = ? AND available_at <= ?) OR (status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?)",
				deliveryStatusPending,
				now,
				deliveryStatusProcessing,
				now,
			).
			Order("available_at ASC, id ASC").
			Limit(limit)
		if supportsSkipLocked(tx) {
			query = query.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
		}

		var rows []NotificationDeliveryPO
		if err := query.Find(&rows).Error; err != nil {
			return fmt.Errorf("find due notification deliveries: %w", err)
		}
		for index := range rows {
			row := &rows[index]
			leaseToken := uuid.NewString()
			leaseExpiresAt := now.Add(leaseDuration)
			update := tx.Model(&NotificationDeliveryPO{}).
				Where("id = ? AND channel = ? AND attempts = ? AND attempts < ?", row.ID, domain.NotificationChannelEmail, row.Attempts, maxAttempts).
				Where(
					"(status = ? AND available_at <= ?) OR (status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?)",
					deliveryStatusPending,
					now,
					deliveryStatusProcessing,
					now,
				).
				Updates(map[string]any{
					"status":           deliveryStatusProcessing,
					"attempts":         gorm.Expr("attempts + 1"),
					"lease_token":      leaseToken,
					"lease_expires_at": leaseExpiresAt,
					"updated_at":       now,
				})
			if update.Error != nil {
				return fmt.Errorf("claim notification delivery: %w", update.Error)
			}
			if update.RowsAffected != 1 {
				continue
			}

			var envelope struct {
				UserID             uint
				RecipientEmail     string
				RecipientDeletedAt *time.Time
				Title              string
				Body               string
				ActionURL          string
				DestinationHash    string
			}
			if err := tx.Table("notifications").
				Select(
					"notifications.user_id, users.email AS recipient_email, users.deleted_at AS recipient_deleted_at, "+
						"notifications.title, notifications.body, notifications.action_url, "+
						"notification_deliveries.destination_hash",
				).
				Joins("JOIN notification_deliveries ON notification_deliveries.notification_id = notifications.id").
				Joins("JOIN users ON users.id = notifications.user_id").
				Where("notifications.id = ? AND notification_deliveries.id = ?", row.NotificationID, row.ID).
				Take(&envelope).Error; err != nil {
				return fmt.Errorf("load notification delivery envelope: %w", err)
			}
			claimed = append(claimed, claimedEmailDelivery{
				ID:                 row.ID,
				NotificationID:     row.NotificationID,
				LeaseToken:         leaseToken,
				Attempts:           row.Attempts + 1,
				UserID:             envelope.UserID,
				RecipientEmail:     envelope.RecipientEmail,
				RecipientDeletedAt: envelope.RecipientDeletedAt,
				DestinationHash:    envelope.DestinationHash,
				Title:              envelope.Title,
				Body:               envelope.Body,
				ActionURL:          envelope.ActionURL,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

func (r *repository) BindEmailDestination(
	ctx context.Context,
	deliveryID uint,
	leaseToken string,
	destinationHash string,
) (bool, error) {
	if r == nil || r.db == nil {
		return false, domain.ErrServiceUnavailable
	}
	result := r.db.WithContext(ctx).Model(&NotificationDeliveryPO{}).
		Where(
			"id = ? AND status = ? AND lease_token = ? AND destination_hash = ''",
			deliveryID,
			deliveryStatusProcessing,
			leaseToken,
		).
		Update("destination_hash", destinationHash)
	if result.Error != nil {
		return false, fmt.Errorf("bind notification email destination: %w", result.Error)
	}
	if result.RowsAffected == 1 {
		return true, nil
	}

	// MySQL reports zero affected rows for an idempotent same-value update. Recheck the
	// current lease and route so a retry can proceed without accepting a stale worker.
	var count int64
	if err := r.db.WithContext(ctx).Model(&NotificationDeliveryPO{}).
		Where(
			"id = ? AND status = ? AND lease_token = ? AND destination_hash = ?",
			deliveryID,
			deliveryStatusProcessing,
			leaseToken,
			destinationHash,
		).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("verify notification email destination: %w", err)
	}
	return count == 1, nil
}

func (r *repository) CompleteEmail(
	ctx context.Context,
	deliveryID uint,
	leaseToken string,
	completion deliveryCompletion,
) (bool, error) {
	if r == nil || r.db == nil {
		return false, domain.ErrServiceUnavailable
	}
	updates := map[string]any{
		"status":            completion.Status,
		"lease_token":       "",
		"lease_expires_at":  nil,
		"last_failure_code": completion.FailureCode,
		"updated_at":        completion.CompletedAt,
	}
	if completion.Status == deliveryStatusDelivered {
		updates["delivered_at"] = completion.CompletedAt
	}
	if completion.RetryAt != nil {
		updates["available_at"] = *completion.RetryAt
	}
	result := r.db.WithContext(ctx).Model(&NotificationDeliveryPO{}).
		Where("id = ? AND status = ? AND lease_token = ?", deliveryID, deliveryStatusProcessing, leaseToken).
		Updates(updates)
	if result.Error != nil {
		return false, fmt.Errorf("complete notification delivery: %w", result.Error)
	}
	return result.RowsAffected == 1, nil
}

func findPreference(db *gorm.DB, userID uint) (*domain.NotificationPreference, error) {
	var po NotificationPreferencePO
	if err := db.Where("user_id = ?", userID).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &domain.NotificationPreference{
				UserID:       userID,
				InAppEnabled: true,
				EmailEnabled: true,
			}, nil
		}
		return nil, fmt.Errorf("find notification preference: %w", err)
	}
	return preferenceFromPO(&po), nil
}

func visibleNotifications(db *gorm.DB, userID uint) *gorm.DB {
	return db.Model(&NotificationPO{}).
		Joins(
			"JOIN notification_deliveries AS in_app_delivery "+
				"ON in_app_delivery.notification_id = notifications.id "+
				"AND in_app_delivery.channel = ? AND in_app_delivery.status = ?",
			domain.NotificationChannelInApp,
			deliveryStatusDelivered,
		).
		Where("notifications.user_id = ?", userID)
}

func inAppVisibilitySubquery(db *gorm.DB, userID uint) *gorm.DB {
	return db.Model(&NotificationDeliveryPO{}).
		Select("1").
		Where("notification_deliveries.notification_id = notifications.id").
		Where("notifications.user_id = ?", userID).
		Where("notification_deliveries.channel = ?", domain.NotificationChannelInApp).
		Where("notification_deliveries.status = ?", deliveryStatusDelivered)
}

func supportsSkipLocked(db *gorm.DB) bool {
	if db == nil || db.Dialector == nil {
		return false
	}
	switch db.Name() {
	case "postgres", "mysql":
		return true
	default:
		return false
	}
}
