package audit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

type mockRepository struct {
	mock.Mock
}

var _ domain.AuditLogRepository = (*mockRepository)(nil)

func (m *mockRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *mockRepository) CreateBatch(ctx context.Context, logs []*domain.AuditLog) error {
	args := m.Called(ctx, logs)
	return args.Error(0)
}

func (m *mockRepository) FindByUserIDAfter(ctx context.Context, userID uint, filter domain.AuditLogFilter, after *domain.AuditLogCursor, limit int) ([]*domain.AuditLog, *domain.AuditLogCursor, error) {
	args := m.Called(ctx, userID, filter, after, limit)
	next, _ := args.Get(1).(*domain.AuditLogCursor)
	items, _ := args.Get(0).([]*domain.AuditLog)
	return items, next, args.Error(2)
}

func (m *mockRepository) FindAllAfter(ctx context.Context, filter domain.AuditLogFilter, after *domain.AuditLogCursor, limit int) ([]*domain.AuditLog, *domain.AuditLogCursor, error) {
	args := m.Called(ctx, filter, after, limit)
	next, _ := args.Get(1).(*domain.AuditLogCursor)
	items, _ := args.Get(0).([]*domain.AuditLog)
	return items, next, args.Error(2)
}

func (m *mockRepository) PruneBefore(ctx context.Context, before time.Time, batch int) (int64, error) {
	args := m.Called(ctx, before, batch)
	return args.Get(0).(int64), args.Error(1)
}

func TestServiceRecordDerivesActionAndActor(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	userID := uint(7)

	repo.On("Create", ctx, mock.MatchedBy(func(entry *domain.AuditLog) bool {
		return entry.ActorType == domain.AuditActorUser &&
			entry.ActorID != nil &&
			*entry.ActorID == userID &&
			entry.Action == "update" &&
			entry.Resource == "users.profile" &&
			entry.Method == "PUT"
	})).Return(nil)

	err := svc.Record(ctx, &domain.AuditLog{
		UserID:    &userID,
		Method:    "put",
		Path:      "/v1/users/profile",
		RouteName: "users.profile.update",
	})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestServiceListForUserAfter(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	userID := uint(9)
	filter := domain.AuditLogFilter{Action: "delete"}
	after := &domain.AuditLogCursor{ID: 40}
	expected := []*domain.AuditLog{{ID: 39, UserID: &userID, Action: "delete"}}

	repo.On("FindByUserIDAfter", ctx, userID, filter, after, 15).Return(expected, (*domain.AuditLogCursor)(nil), nil)

	items, next, err := svc.ListForUserAfter(ctx, userID, filter, after, 15)

	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Nil(t, next)
	_, _, err = svc.ListForUserAfter(ctx, 0, filter, nil, 15)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	_, _, err = svc.ListForUserAfter(ctx, userID, filter, nil, 101)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	repo.AssertExpectations(t)
}

func TestServicePruneAuditLogsUsesBoundedPastCutoff(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	before := time.Now().UTC().Add(-90 * 24 * time.Hour).Truncate(time.Second)

	repo.On("PruneBefore", ctx, before, 500).Return(int64(37), nil)

	count, err := svc.PruneAuditLogs(ctx, before, 500)

	assert.NoError(t, err)
	assert.Equal(t, int64(37), count)
	repo.AssertExpectations(t)

	for _, invalid := range []struct {
		before time.Time
		batch  int
	}{
		{before: time.Time{}, batch: 500},
		{before: time.Now().UTC().Add(time.Hour), batch: 500},
		{before: before, batch: 0},
		{before: before, batch: maxAuditPruneBatch + 1},
	} {
		_, err = svc.PruneAuditLogs(ctx, invalid.before, invalid.batch)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	}
}

func TestServiceRecordMergesBusinessChangeFromContext(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)

	ctx := withChangeCollector(context.Background())
	RecordChange(ctx, Change{
		TargetType: "user",
		TargetID:   "42",
		Result:     domain.AuditResultSuccess,
		Changes: map[string]domain.AuditValueChange{
			"nickname": {Before: "old", After: "new"},
		},
	})

	repo.On("Create", ctx, mock.MatchedBy(func(entry *domain.AuditLog) bool {
		return entry.TargetType == "user" &&
			entry.TargetID == "42" &&
			entry.Result == domain.AuditResultSuccess &&
			entry.Changes["nickname"].Before == "old" &&
			entry.Changes["nickname"].After == "new"
	})).Return(nil)

	err := svc.Record(ctx, &domain.AuditLog{
		Method: "PATCH",
		Path:   "/v1/users/profile",
	})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestServiceRecordRedactsSensitiveBusinessMetadata(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := withChangeCollector(context.Background())
	RecordChange(ctx, Change{
		Changes: map[string]domain.AuditValueChange{
			"password":      {Before: "old-password", After: "new-password"},
			"client_secret": {Before: nil, After: "new-client-secret"},
		},
		Metadata: map[string]any{
			"access_token": "audit-access-token",
			"nested": map[string]any{
				"client_secret": "audit-client-secret",
				"outcome":       "success",
			},
		},
	})

	repo.On("Create", ctx, mock.MatchedBy(func(entry *domain.AuditLog) bool {
		password := entry.Changes["password"]
		clientSecret := entry.Changes["client_secret"]
		nested, ok := entry.Metadata["nested"].(map[string]any)
		return password.Before == "[REDACTED]" &&
			password.After == "[REDACTED]" &&
			clientSecret.Before == nil &&
			clientSecret.After == "[REDACTED]" &&
			entry.Metadata["access_token"] == "[REDACTED]" &&
			ok &&
			nested["client_secret"] == "[REDACTED]" &&
			nested["outcome"] == "success"
	})).Return(nil)

	err := svc.Record(ctx, &domain.AuditLog{Method: "POST", Path: "/v1/example"})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestServiceListAuditLogsAfterBoundsTheTimeRange(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	repo.On("FindAllAfter", mock.Anything, mock.MatchedBy(func(filter domain.AuditLogFilter) bool {
		return filter.To.Sub(filter.From) == 30*24*time.Hour
	}), (*domain.AuditLogCursor)(nil), 50).Return([]*domain.AuditLog{}, (*domain.AuditLogCursor)(nil), nil).Once()
	_, _, err := svc.ListAuditLogsAfter(context.Background(), domain.AuditLogFilter{To: now}, nil, 50)
	require.NoError(t, err, "a missing start defaults to 30 days before the end")

	for name, filter := range map[string]domain.AuditLogFilter{
		"reversed range": {From: now, To: now.Add(-time.Hour)},
		"empty range":    {From: now, To: now},
		"range too long": {From: now.Add(-domain.MaxAuditQueryRange - time.Hour), To: now},
	} {
		_, _, rangeErr := svc.ListAuditLogsAfter(context.Background(), filter, nil, 50)
		require.ErrorIs(t, rangeErr, domain.ErrInvalidInput, name)
	}
	_, _, err = svc.ListAuditLogsAfter(context.Background(), domain.AuditLogFilter{}, nil, 101)
	require.ErrorIs(t, err, domain.ErrInvalidInput)
	repo.AssertExpectations(t)
}
