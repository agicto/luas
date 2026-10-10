package audit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	testplatform "github.com/zgiai/luas/api/internal/infra/testing"
)

// Keyset pages must visit every matching record exactly once, newest first, including records
// that share a timestamp, and must agree with the offset pages they replace.
func TestRepositoryKeysetPagesVisitEveryRecordOncePostgres(t *testing.T) {
	db := testplatform.OpenPostgres(t, nil, &AuditLogPO{})
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	owner, other := uint(1), uint(2)
	for index := 0; index < 23; index++ {
		user := &owner
		if index%4 == 0 {
			user = &other
		}
		// Groups of three share a timestamp to exercise the id tie-breaker.
		row := AuditLogPO{
			CreatedAt: base.Add(time.Duration(index/3) * time.Minute), UserID: user, ActorType: "user",
			Action: "update", Resource: "users", Method: "POST", Path: "/p", StatusCode: 200,
		}
		require.NoError(t, db.Create(&row).Error)
	}
	repo := NewRepository(db)
	ctx := context.Background()

	collect := func(t *testing.T, page func(after *domain.AuditLogCursor) ([]*domain.AuditLog, *domain.AuditLogCursor, error)) []uint {
		t.Helper()
		var ids []uint
		var after *domain.AuditLogCursor
		for pages := 0; ; pages++ {
			require.Less(t, pages, 20, "pagination did not terminate")
			items, next, err := page(after)
			require.NoError(t, err)
			for _, item := range items {
				ids = append(ids, item.ID)
			}
			if next == nil {
				return ids
			}
			// Round-trip through the public token, as a client would.
			after, err = DecodeCursor(EncodeCursor(next))
			require.NoError(t, err)
		}
	}

	t.Run("platform-wide by time then id", func(t *testing.T) {
		keyset := collect(t, func(after *domain.AuditLogCursor) ([]*domain.AuditLog, *domain.AuditLogCursor, error) {
			return repo.FindAllAfter(ctx, domain.AuditLogFilter{}, after, 4)
		})
		offset, total, err := repo.FindAll(ctx, domain.AuditLogFilter{}, 1, 100)
		require.NoError(t, err)
		require.EqualValues(t, 23, total)
		expected := make([]uint, len(offset))
		for index, item := range offset {
			expected[index] = item.ID
		}
		assert.Equal(t, expected, keyset)
	})

	t.Run("one user by id", func(t *testing.T) {
		keyset := collect(t, func(after *domain.AuditLogCursor) ([]*domain.AuditLog, *domain.AuditLogCursor, error) {
			return repo.FindByUserIDAfter(ctx, owner, domain.AuditLogFilter{}, after, 5)
		})
		offset, total, err := repo.FindByUserID(ctx, owner, domain.AuditLogFilter{}, 1, 100)
		require.NoError(t, err)
		require.EqualValues(t, 17, total)
		expected := make([]uint, len(offset))
		for index, item := range offset {
			expected[index] = item.ID
		}
		assert.Equal(t, expected, keyset)
	})

	t.Run("an exact final page reports no next cursor", func(t *testing.T) {
		items, next, err := repo.FindByUserIDAfter(ctx, other, domain.AuditLogFilter{}, nil, 6)
		require.NoError(t, err)
		assert.Len(t, items, 6)
		assert.Nil(t, next)
	})
}

func TestAsyncRequestAuditRecordsAreWrittenByShutdownPostgres(t *testing.T) {
	db := testplatform.OpenPostgres(t, nil, &AuditLogPO{})
	svc := ProvideService(NewRepository(db), &config.Config{Audit: config.AuditConfig{WriteMode: config.AuditWriteModeAsync}})
	require.NotNil(t, svc.writer)
	user := uint(7)
	requestTime := time.Now().UTC()
	for index := 0; index < 450; index++ {
		require.NoError(t, svc.RecordRequest(context.Background(), &domain.AuditLog{
			UserID: &user, Method: "POST", Path: "/v1/things", RouteName: "things.store", StatusCode: 201,
		}))
	}
	require.NoError(t, svc.Shutdown(context.Background()))

	var stored []AuditLogPO
	require.NoError(t, db.Order("id").Find(&stored).Error)
	require.Len(t, stored, 450, "every accepted record is written by shutdown")
	assert.Equal(t, "things", stored[0].Resource)
	assert.Equal(t, domain.AuditActorUser, stored[0].ActorType)
	assert.WithinDuration(t, requestTime, stored[0].CreatedAt, time.Second,
		"created_at is the request time, not the flush time")
}
