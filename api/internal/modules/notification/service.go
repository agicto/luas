package notification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/email"
	infraevents "github.com/zgiai/luas/api/internal/infra/events"
	auditstarter "github.com/zgiai/luas/api/internal/modules/audit"
	pkgevents "github.com/zgiai/luas/api/pkg/events"
)

const (
	notificationFilterAll    = "all"
	notificationFilterUnread = "unread"

	defaultDeliveryLease = time.Minute
	maxDeliveryAttempts  = uint8(5)
	maxDispatchBatch     = 100

	failureNotConfigured        = "EMAIL.NOT_CONFIGURED"
	failureRecipientUnavailable = "EMAIL.RECIPIENT_UNAVAILABLE"
	failureRouteChanged         = "EMAIL.ROUTE_CHANGED"
	failureInvalidMessage       = "EMAIL.INVALID_MESSAGE"
	failureProviderRejected     = "EMAIL.PROVIDER_REJECTED"
	failureProviderUnavailable  = "EMAIL.PROVIDER_UNAVAILABLE"
	failureRetryExhausted       = "EMAIL.RETRY_EXHAUSTED"
)

var (
	idempotencyKeyPattern   = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
	notificationKindPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$`)
)

type emailSender interface {
	IsConfigured() bool
	SendEmailIdempotent(context.Context, []string, string, string, string) error
}

type eventPublisher interface {
	Publish(context.Context, pkgevents.Event) error
}

// Service owns publication, recipient state, and durable delivery dispatch.
type Service interface {
	domain.NotificationPublisher
	domain.NotificationDispatcher
	ListForUser(context.Context, uint, string, int, int) ([]*domain.Notification, int64, error)
	UnreadCount(context.Context, uint) (int64, error)
	ReplaceReadState(context.Context, uint, uint, bool) (*domain.Notification, error)
	MarkReadThrough(context.Context, uint, uint) (int64, int64, error)
	Preference(context.Context, uint) (*domain.NotificationPreference, error)
	ReplacePreference(context.Context, uint, bool, bool) (*domain.NotificationPreference, error)
}

type service struct {
	enabled bool
	store   notificationStore
	mailer  emailSender
	events  eventPublisher
	now     func() time.Time
}

var (
	_ Service                       = (*service)(nil)
	_ domain.NotificationPublisher  = (*service)(nil)
	_ domain.NotificationDispatcher = (*service)(nil)
)

// NewService creates the optional notification starter service.
func NewService(
	cfg *config.Config,
	store notificationStore,
	mailer emailSender,
	events eventPublisher,
) *service {
	return &service{
		enabled: notificationStarterEnabled(cfg),
		store:   store,
		mailer:  mailer,
		events:  events,
		now:     func() time.Time { return time.Now().UTC() },
	}
}

func (s *service) Publish(
	ctx context.Context,
	publication domain.NotificationPublication,
) (*domain.Notification, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	normalized, err := normalizePublication(publication)
	if err != nil {
		return nil, err
	}
	fingerprint, err := publicationFingerprint(normalized)
	if err != nil {
		return nil, err
	}
	result, err := s.store.CreatePublication(ctx, normalized, fingerprint, s.now())
	if err != nil {
		return nil, fmt.Errorf("publish notification: %w", err)
	}
	if result == nil || result.Notification == nil {
		return nil, domain.ErrServiceUnavailable
	}
	if result.Created && s.events != nil {
		if eventErr := s.events.Publish(ctx, domain.NewNotificationCreatedEvent(result.Notification)); eventErr != nil {
			slog.WarnContext(ctx, "notification.created_event_failed",
				"notification_id", result.Notification.ID,
			)
		}
	}
	return result.Notification, nil
}

func (s *service) ListForUser(
	ctx context.Context,
	userID uint,
	status string,
	page int,
	pageSize int,
) ([]*domain.Notification, int64, error) {
	if err := s.available(); err != nil {
		return nil, 0, err
	}
	if userID == 0 || page < 1 || pageSize < 1 || pageSize > 100 || !validNotificationFilter(status) {
		return nil, 0, domain.ErrInvalidInput
	}
	return s.store.ListForUser(ctx, userID, status, page, pageSize)
}

func (s *service) UnreadCount(ctx context.Context, userID uint) (int64, error) {
	if err := s.available(); err != nil {
		return 0, err
	}
	if userID == 0 {
		return 0, domain.ErrInvalidInput
	}
	return s.store.UnreadCount(ctx, userID)
}

func (s *service) ReplaceReadState(
	ctx context.Context,
	userID uint,
	notificationID uint,
	isRead bool,
) (*domain.Notification, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	if userID == 0 || notificationID == 0 {
		return nil, domain.ErrInvalidInput
	}
	notification, err := s.store.ReplaceReadState(ctx, userID, notificationID, isRead, s.now())
	if err != nil {
		return nil, err
	}
	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     "update",
		Resource:   "notifications",
		TargetType: "notification",
		TargetID:   strconv.FormatUint(uint64(notificationID), 10),
		Result:     domain.AuditResultSuccess,
		Metadata: map[string]any{
			"is_read": isRead,
		},
	})
	return notification, nil
}

func (s *service) MarkReadThrough(
	ctx context.Context,
	userID uint,
	throughID uint,
) (int64, int64, error) {
	if err := s.available(); err != nil {
		return 0, 0, err
	}
	if userID == 0 || throughID == 0 {
		return 0, 0, domain.ErrInvalidInput
	}
	updated, unread, err := s.store.MarkReadThrough(ctx, userID, throughID, s.now())
	if err != nil {
		return 0, 0, err
	}
	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     "update",
		Resource:   "notification-read-state",
		TargetType: "user",
		TargetID:   strconv.FormatUint(uint64(userID), 10),
		Result:     domain.AuditResultSuccess,
		Metadata: map[string]any{
			"through_id":    throughID,
			"updated_count": updated,
		},
	})
	return updated, unread, nil
}

func (s *service) Preference(ctx context.Context, userID uint) (*domain.NotificationPreference, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	if userID == 0 {
		return nil, domain.ErrInvalidInput
	}
	return s.store.Preference(ctx, userID)
}

func (s *service) ReplacePreference(
	ctx context.Context,
	userID uint,
	inAppEnabled bool,
	emailEnabled bool,
) (*domain.NotificationPreference, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	if userID == 0 {
		return nil, domain.ErrInvalidInput
	}
	before, err := s.store.Preference(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	preference := &domain.NotificationPreference{
		UserID:       userID,
		InAppEnabled: inAppEnabled,
		EmailEnabled: emailEnabled,
		CreatedAt:    before.CreatedAt,
		UpdatedAt:    now,
	}
	if preference.CreatedAt.IsZero() {
		preference.CreatedAt = now
	}
	if err := s.store.ReplacePreference(ctx, preference); err != nil {
		return nil, err
	}
	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     "update",
		Resource:   "notification-preferences",
		TargetType: "user",
		TargetID:   strconv.FormatUint(uint64(userID), 10),
		Result:     domain.AuditResultSuccess,
		Changes: map[string]domain.AuditValueChange{
			"in_app_enabled": {Before: before.InAppEnabled, After: inAppEnabled},
			"email_enabled":  {Before: before.EmailEnabled, After: emailEnabled},
		},
	})
	return preference, nil
}

func (s *service) DispatchDue(ctx context.Context, limit int) (int, error) {
	if err := s.available(); err != nil {
		return 0, err
	}
	if limit < 1 || limit > maxDispatchBatch {
		return 0, domain.ErrInvalidInput
	}
	now := s.now()
	if _, err := s.store.FailExhausted(ctx, now, maxDeliveryAttempts); err != nil {
		return 0, err
	}
	deliveries, err := s.store.ClaimDueEmail(
		ctx,
		now,
		defaultDeliveryLease,
		maxDeliveryAttempts,
		limit,
	)
	if err != nil {
		return 0, err
	}

	completed := 0
	for index := range deliveries {
		if err := ctx.Err(); err != nil {
			return completed, err
		}
		ok, err := s.dispatchEmail(ctx, deliveries[index])
		if err != nil {
			return completed, err
		}
		if ok {
			completed++
		}
	}
	return completed, nil
}

func (s *service) dispatchEmail(ctx context.Context, delivery claimedEmailDelivery) (bool, error) {
	now := s.now()
	if delivery.RecipientDeletedAt != nil || strings.TrimSpace(delivery.RecipientEmail) == "" {
		return s.completePermanent(ctx, delivery, now, failureRecipientUnavailable)
	}
	destinationHash := hashDestination(delivery.RecipientEmail)
	if delivery.DestinationHash != "" && delivery.DestinationHash != destinationHash {
		return s.completePermanent(ctx, delivery, now, failureRouteChanged)
	}
	bound, err := s.store.BindEmailDestination(ctx, delivery.ID, delivery.LeaseToken, destinationHash)
	if err != nil {
		return false, err
	}
	if !bound {
		return false, nil
	}
	if s.mailer == nil || !s.mailer.IsConfigured() {
		return s.completePermanent(ctx, delivery, now, failureNotConfigured)
	}

	err = s.mailer.SendEmailIdempotent(
		ctx,
		[]string{delivery.RecipientEmail},
		delivery.Title,
		renderEmailBody(delivery),
		fmt.Sprintf("notification-email-%d", delivery.ID),
	)
	if err == nil {
		return s.store.CompleteEmail(ctx, delivery.ID, delivery.LeaseToken, deliveryCompletion{
			Status:      deliveryStatusDelivered,
			CompletedAt: s.now(),
		})
	}
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	failureCode, retryable := classifyEmailFailure(err)
	if !retryable || delivery.Attempts >= maxDeliveryAttempts {
		return s.completePermanent(ctx, delivery, s.now(), failureCode)
	}
	retryAt := s.now().Add(deliveryRetryDelay(delivery.Attempts))
	return s.store.CompleteEmail(ctx, delivery.ID, delivery.LeaseToken, deliveryCompletion{
		Status:      deliveryStatusPending,
		CompletedAt: s.now(),
		RetryAt:     &retryAt,
		FailureCode: failureCode,
	})
}

func (s *service) completePermanent(
	ctx context.Context,
	delivery claimedEmailDelivery,
	now time.Time,
	failureCode string,
) (bool, error) {
	return s.store.CompleteEmail(ctx, delivery.ID, delivery.LeaseToken, deliveryCompletion{
		Status:      deliveryStatusFailed,
		CompletedAt: now,
		FailureCode: failureCode,
	})
}

func (s *service) available() error {
	if s == nil || !s.enabled || s.store == nil || s.now == nil {
		return domain.ErrServiceUnavailable
	}
	return nil
}

func notificationStarterEnabled(cfg *config.Config) bool {
	return cfg != nil && slices.Contains(cfg.Starters.Optional, "notification")
}

func validNotificationFilter(status string) bool {
	return status == notificationFilterAll || status == notificationFilterUnread
}

func normalizePublication(publication domain.NotificationPublication) (domain.NotificationPublication, error) {
	if publication.UserID == 0 || !idempotencyKeyPattern.MatchString(publication.IdempotencyKey) {
		return domain.NotificationPublication{}, domain.ErrInvalidInput
	}
	if len(publication.Kind) > 100 || !notificationKindPattern.MatchString(publication.Kind) {
		return domain.NotificationPublication{}, domain.ErrInvalidInput
	}
	publication.Title = strings.TrimSpace(publication.Title)
	publication.Body = strings.TrimSpace(publication.Body)
	if !validBoundedText(publication.Title, 160) || !validBoundedText(publication.Body, 4_000) {
		return domain.NotificationPublication{}, domain.ErrInvalidInput
	}
	actionURL, err := normalizeActionURL(publication.ActionURL)
	if err != nil {
		return domain.NotificationPublication{}, err
	}
	publication.ActionURL = actionURL

	channels, err := canonicalChannels(publication.Channels)
	if err != nil || len(channels) == 0 {
		return domain.NotificationPublication{}, domain.ErrNotificationInvalidChannel
	}
	required, err := canonicalChannels(publication.RequiredChannels)
	if err != nil {
		return domain.NotificationPublication{}, err
	}
	for _, channel := range required {
		if !slices.Contains(channels, channel) {
			return domain.NotificationPublication{}, domain.ErrNotificationInvalidChannel
		}
	}
	publication.Channels = channels
	publication.RequiredChannels = required
	return publication, nil
}

func canonicalChannels(channels []domain.NotificationChannel) ([]domain.NotificationChannel, error) {
	result := append([]domain.NotificationChannel(nil), channels...)
	slices.Sort(result)
	for index, channel := range result {
		if channel != domain.NotificationChannelInApp && channel != domain.NotificationChannelEmail {
			return nil, domain.ErrNotificationInvalidChannel
		}
		if index > 0 && result[index-1] == channel {
			return nil, domain.ErrNotificationInvalidChannel
		}
	}
	return result, nil
}

func selectPublicationChannels(
	publication domain.NotificationPublication,
	preference *domain.NotificationPreference,
) ([]domain.NotificationChannel, error) {
	if preference == nil || preference.UserID != publication.UserID {
		return nil, domain.ErrServiceUnavailable
	}
	required := make(map[domain.NotificationChannel]struct{}, len(publication.RequiredChannels))
	for _, channel := range publication.RequiredChannels {
		required[channel] = struct{}{}
	}
	selected := make([]domain.NotificationChannel, 0, len(publication.Channels))
	for _, channel := range publication.Channels {
		_, mandatory := required[channel]
		enabled := mandatory
		switch channel {
		case domain.NotificationChannelInApp:
			enabled = enabled || preference.InAppEnabled
		case domain.NotificationChannelEmail:
			enabled = enabled || preference.EmailEnabled
		default:
			return nil, domain.ErrNotificationInvalidChannel
		}
		if enabled {
			selected = append(selected, channel)
		}
	}
	return selected, nil
}

func publicationFingerprint(publication domain.NotificationPublication) (string, error) {
	payload, err := json.Marshal(struct {
		UserID           uint                         `json:"user_id"`
		IdempotencyKey   string                       `json:"idempotency_key"`
		Kind             string                       `json:"kind"`
		Title            string                       `json:"title"`
		Body             string                       `json:"body"`
		ActionURL        string                       `json:"action_url"`
		Channels         []domain.NotificationChannel `json:"channels"`
		RequiredChannels []domain.NotificationChannel `json:"required_channels"`
	}{
		UserID:           publication.UserID,
		IdempotencyKey:   publication.IdempotencyKey,
		Kind:             publication.Kind,
		Title:            publication.Title,
		Body:             publication.Body,
		ActionURL:        publication.ActionURL,
		Channels:         publication.Channels,
		RequiredChannels: publication.RequiredChannels,
	})
	if err != nil {
		return "", fmt.Errorf("encode notification fingerprint: %w", err)
	}
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:]), nil
}

func normalizeActionURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if len(value) > 2_048 || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || containsControl(value) {
		return "", domain.ErrInvalidInput
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return "", domain.ErrInvalidInput
	}
	decodedPath, err := url.PathUnescape(parsed.EscapedPath())
	if err != nil || strings.HasPrefix(decodedPath, "//") || strings.Contains(decodedPath, `\`) || containsControl(decodedPath) {
		return "", domain.ErrInvalidInput
	}
	return value, nil
}

func validBoundedText(value string, maxRunes int) bool {
	return value != "" && utf8.ValidString(value) && utf8.RuneCountInString(value) <= maxRunes && !containsControlExceptWhitespace(value)
}

func containsControl(value string) bool {
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return true
		}
	}
	return false
}

func containsControlExceptWhitespace(value string) bool {
	for _, char := range value {
		if (char < 0x20 && char != '\n' && char != '\r' && char != '\t') || char == 0x7f {
			return true
		}
	}
	return false
}

func renderEmailBody(delivery claimedEmailDelivery) string {
	body := strings.ReplaceAll(html.EscapeString(delivery.Body), "\n", "<br>")
	var builder strings.Builder
	builder.WriteString("<p>")
	builder.WriteString(body)
	builder.WriteString("</p>")
	if delivery.ActionURL != "" {
		builder.WriteString(`<p><a href="`)
		builder.WriteString(html.EscapeString(delivery.ActionURL))
		builder.WriteString(`">View details</a></p>`)
	}
	return builder.String()
}

func classifyEmailFailure(err error) (string, bool) {
	switch {
	case errors.Is(err, email.ErrNotConfigured):
		return failureNotConfigured, false
	case errors.Is(err, email.ErrInvalidMessage):
		return failureInvalidMessage, false
	case errors.Is(err, email.ErrProviderResponseTooLarge), errors.Is(err, email.ErrInvalidProviderResponse):
		return failureProviderUnavailable, true
	}
	var providerError *email.ProviderError
	if errors.As(err, &providerError) {
		status := providerError.StatusCode
		if status == http.StatusRequestTimeout || status == http.StatusConflict || status == http.StatusTooEarly ||
			status == http.StatusTooManyRequests || status >= http.StatusInternalServerError {
			return failureProviderUnavailable, true
		}
		return failureProviderRejected, false
	}
	return failureProviderUnavailable, true
}

func deliveryRetryDelay(attempt uint8) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := 30 * time.Second * time.Duration(1<<min(int(attempt-1), 6))
	return min(delay, 30*time.Minute)
}

func hashDestination(value string) string {
	hash := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(value))))
	return hex.EncodeToString(hash[:])
}

// Ensure concrete infrastructure providers satisfy the private adapter seams.
var (
	_ emailSender    = (*email.Service)(nil)
	_ eventPublisher = (*infraevents.EventBus)(nil)
)
