package notification

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

func TestOperatorDeliveryLedgerFiltersWithoutExposingContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, db, _, _ := newNotificationTestService(t)
	ctx := context.Background()
	firstUser := createNotificationTestUser(t, db, "first")
	secondUser := createNotificationTestUser(t, db, "second")
	first, err := service.Publish(ctx, testPublication(firstUser, "ledger-1"))
	require.NoError(t, err)
	_, err = service.Publish(ctx, testPublication(secondUser, "ledger-2"))
	require.NoError(t, err)
	require.NoError(t, db.Model(&NotificationDeliveryPO{}).
		Where("notification_id = ? AND channel = ?", first.ID, "email").
		Updates(map[string]any{"status": string(deliveryStatusFailed), "attempts": 5, "last_failure_code": "provider_rejected"}).Error)

	repo := NewRepository(db)
	all, total, err := repo.operatorDeliveries(ctx, operatorDeliveryFilter{}, 1, 10)
	require.NoError(t, err)
	require.EqualValues(t, 4, total, "two channels for each of two notifications")
	require.Len(t, all, 4)
	for index := 1; index < len(all); index++ {
		assert.Greater(t, all[index-1].ID, all[index].ID, "newest first")
	}

	failed, total, err := repo.operatorDeliveries(ctx, operatorDeliveryFilter{Status: "failed", Channel: "email"}, 1, 10)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	assert.Equal(t, firstUser, failed[0].UserID)
	assert.Equal(t, first.ID, failed[0].NotificationID)
	assert.Equal(t, "billing.invoice_paid", failed[0].Kind)
	assert.Equal(t, "provider_rejected", failed[0].LastFailureCode)
	assert.EqualValues(t, 5, failed[0].Attempts)

	byUser, total, err := repo.operatorDeliveries(ctx, operatorDeliveryFilter{UserID: secondUser}, 1, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	for _, delivery := range byUser {
		assert.Equal(t, secondUser, delivery.UserID)
	}

	empty, total, err := repo.operatorDeliveries(ctx, operatorDeliveryFilter{UserID: secondUser, Status: "failed"}, 1, 10)
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.NotNil(t, empty)

	_, _, err = repo.operatorDeliveries(ctx, operatorDeliveryFilter{}, 1, 101)
	require.ErrorIs(t, err, domain.ErrInvalidInput)

	engine := gin.New()
	engine.GET("/notification-deliveries", NewOperatorHandler(repo).ListDeliveries)
	get := func(path string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		return recorder
	}
	page := get("/notification-deliveries?status=failed")
	require.Equal(t, http.StatusOK, page.Code)
	assert.Equal(t, "private, no-store", page.Header().Get("Cache-Control"))
	assert.Contains(t, page.Body.String(), `"last_failure_code":"provider_rejected"`)
	for _, private := range []string{"Invoice paid", "Invoice 1042", "/console/invoices", "destination", "lease"} {
		assert.NotContains(t, get("/notification-deliveries").Body.String(), private)
	}
	assert.Equal(t, http.StatusBadRequest, get("/notification-deliveries?status=exploded").Code)
	assert.Equal(t, http.StatusBadRequest, get("/notification-deliveries?channel=sms").Code)
}
