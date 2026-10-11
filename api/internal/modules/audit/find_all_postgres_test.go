package audit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	testplatform "github.com/zgiai/luas/api/internal/infra/testing"
)

func TestRepositoryFindAllAfterPostgres(t *testing.T) {
	db := testplatform.OpenPostgres(t, nil, &AuditLogPO{})
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	userA, userB := uint(1), uint(2)
	rows := []AuditLogPO{
		{CreatedAt: base, UserID: &userA, ActorType: "user", Action: "update", Resource: "users", Method: "POST", Path: "/a", StatusCode: 200},
		{CreatedAt: base, UserID: &userB, ActorType: "user", Action: "update", Resource: "users", Method: "POST", Path: "/b", StatusCode: 200},
		{CreatedAt: base.Add(time.Hour), UserID: &userA, ActorType: "user", Action: "delete", Resource: "users", Method: "DELETE", Path: "/c", StatusCode: 204},
		{CreatedAt: base.Add(48 * time.Hour), ActorType: "system", Action: "grant", Resource: "platform_operators", Method: "CLI", Path: "operator:grant", StatusCode: 200},
	}
	for index := range rows {
		require.NoError(t, db.Create(&rows[index]).Error)
	}
	repo := NewRepository(db)
	ctx := context.Background()

	t.Run("orders newest first with id as tie-breaker", func(t *testing.T) {
		items, next, err := repo.FindAllAfter(ctx, domain.AuditLogFilter{}, nil, 10)
		require.NoError(t, err)
		assert.Nil(t, next)
		paths := make([]string, len(items))
		for index, item := range items {
			paths[index] = item.Path
		}
		assert.Equal(t, []string{"operator:grant", "/c", "/b", "/a"}, paths)
	})

	t.Run("treats From as inclusive and To as exclusive", func(t *testing.T) {
		items, _, err := repo.FindAllAfter(ctx, domain.AuditLogFilter{From: base, To: base.Add(time.Hour)}, nil, 10)
		require.NoError(t, err)
		assert.Len(t, items, 2)
	})

	t.Run("filters by user and action", func(t *testing.T) {
		items, _, err := repo.FindAllAfter(ctx, domain.AuditLogFilter{UserID: &userA, Action: "delete"}, nil, 10)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "/c", items[0].Path)
	})

	t.Run("pages deterministically", func(t *testing.T) {
		first, next, err := repo.FindAllAfter(ctx, domain.AuditLogFilter{}, nil, 3)
		require.NoError(t, err)
		require.NotNil(t, next)
		second, last, err := repo.FindAllAfter(ctx, domain.AuditLogFilter{}, next, 3)
		require.NoError(t, err)
		require.Len(t, first, 3)
		require.Len(t, second, 1)
		assert.Nil(t, last)
		assert.Equal(t, "/a", second[0].Path)
	})
}
