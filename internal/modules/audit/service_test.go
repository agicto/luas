package audit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zgiai/zgo/internal/domain"
)

type mockRepository struct {
	mock.Mock
}

var _ domain.AuditLogRepository = (*mockRepository)(nil)

func (m *mockRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *mockRepository) FindByUserID(ctx context.Context, userID uint, filter domain.AuditLogFilter, page, pageSize int) ([]*domain.AuditLog, int64, error) {
	args := m.Called(ctx, userID, filter, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.AuditLog), args.Get(1).(int64), args.Error(2)
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

func TestServiceListForUser(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()
	userID := uint(9)
	filter := domain.AuditLogFilter{Action: "delete"}
	expected := []*domain.AuditLog{{ID: 1, UserID: &userID, Action: "delete"}}

	repo.On("FindByUserID", ctx, userID, filter, 1, 15).Return(expected, int64(1), nil)

	items, total, err := svc.ListForUser(ctx, userID, filter, 1, 15)

	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, int64(1), total)
	repo.AssertExpectations(t)
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
