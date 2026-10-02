package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

func TestOperatorHandlerDiagnosesAndReplaysWithinThePathOrganization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	receiver := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer receiver.Close()

	service, organizationID, actorID, _ := newWebhookServiceTest(t)
	(&Handler{}).RegisterErrorMappings(response.DefaultErrorMapper)
	ctx := context.Background()
	created, err := service.CreateEndpoint(ctx, organizationID, actorID, endpointInput{
		Name: "Receiver", URL: receiver.URL + "/hook", EventTypes: []string{"webhook.test"},
	})
	require.NoError(t, err)
	delivery, err := service.PublishWebhookTest(ctx, organizationID, created.Endpoint.ID, actorID, "operator-test-1")
	require.NoError(t, err)

	const operatorID = uint(4242)
	handler := NewOperatorHandler(service)
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("userID", operatorID) })
	engine.GET("/organizations/:id/webhook-endpoints", handler.ListEndpoints)
	engine.GET("/organizations/:id/webhook-deliveries", handler.ListDeliveries)
	engine.GET("/organizations/:id/webhook-deliveries/:delivery_id/attempts", handler.ListAttempts)
	engine.POST("/organizations/:id/webhook-deliveries/:delivery_id/replay", handler.ReplayDelivery)
	call := func(method, path string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest(method, path, nil))
		return recorder
	}
	own := fmt.Sprintf("/organizations/%d", organizationID)
	other := fmt.Sprintf("/organizations/%d", organizationID+1000)
	deliveryPath := fmt.Sprintf("/webhook-deliveries/%d", delivery.ID)

	endpoints := call(http.MethodGet, own+"/webhook-endpoints")
	require.Equal(t, http.StatusOK, endpoints.Code)
	assert.Equal(t, "private, no-store", endpoints.Header().Get("Cache-Control"))
	assert.Contains(t, endpoints.Body.String(), `"name":"Receiver"`)
	assert.NotContains(t, endpoints.Body.String(), created.SigningSecret)
	assert.NotContains(t, endpoints.Body.String(), "ciphertext")
	assert.NotContains(t, call(http.MethodGet, other+"/webhook-endpoints").Body.String(), "Receiver")

	pending := call(http.MethodPost, own+deliveryPath+"/replay")
	assert.Equal(t, http.StatusConflict, pending.Code, "a pending delivery is not terminal")
	assert.Contains(t, pending.Body.String(), domain.CodeWebhookReplayNotAllowed)

	processed, err := service.DispatchWebhooks(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, processed)

	deliveries := call(http.MethodGet, own+"/webhook-deliveries?status=delivered")
	require.Equal(t, http.StatusOK, deliveries.Code)
	assert.Contains(t, deliveries.Body.String(), delivery.MessageID)
	assert.NotContains(t, deliveries.Body.String(), "payload")
	assert.Equal(t, http.StatusBadRequest, call(http.MethodGet, own+"/webhook-deliveries?status=exploded").Code)

	attempts := call(http.MethodGet, own+deliveryPath+"/attempts")
	require.Equal(t, http.StatusOK, attempts.Code)
	assert.Contains(t, attempts.Body.String(), `"number":1`)

	for _, request := range []struct{ method, path string }{
		{http.MethodGet, other + deliveryPath + "/attempts"},
		{http.MethodPost, other + deliveryPath + "/replay"},
	} {
		crossTenant := call(request.method, request.path)
		assert.Equal(t, http.StatusNotFound, crossTenant.Code, request.path)
		assert.Contains(t, crossTenant.Body.String(), domain.CodeWebhookDeliveryNotFound)
	}
	assert.Equal(t, http.StatusBadRequest, call(http.MethodPost, own+"/webhook-deliveries/0/replay").Code)

	replayed := call(http.MethodPost, own+deliveryPath+"/replay")
	require.Equal(t, http.StatusOK, replayed.Code, replayed.Body.String())
	var body struct {
		Data DeliveryResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(replayed.Body.Bytes(), &body))
	assert.Equal(t, domain.WebhookDeliveryStatusPending, body.Data.Status)
	assert.EqualValues(t, 1, body.Data.ReplayCount)
	assert.Equal(t, delivery.MessageID, body.Data.MessageID, "replay keeps the original message ID")
}
