package webhook

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

func TestHandlerCreateReturnsSecretOnceAndListsSecretFreeEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, organizationID, actorID, _ := newWebhookServiceTest(t)
	handler := NewHandler(service)
	body := `{"name":"Receiver","url":"http://127.0.0.1:8080/hook","event_types":["webhook.test"]}`
	recorder, context := newWebhookHandlerContext(http.MethodPost, "/v1/webhook-endpoints", body, organizationID, actorID, domain.OrganizationRoleOwner)

	handler.CreateEndpoint(context)
	require.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, `"webhook-endpoint-v1"`, recorder.Header().Get("ETag"))
	assert.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	var created struct {
		Data EndpointSecretResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &created))
	require.NotNil(t, created.Data.Endpoint)
	assert.True(t, strings.HasPrefix(created.Data.SigningSecret, webhookSecretPrefix))
	plaintext := created.Data.SigningSecret

	recorder, context = newWebhookHandlerContext(http.MethodGet, "/v1/webhook-endpoints", "", organizationID, actorID, domain.OrganizationRoleOwner)
	handler.ListEndpoints(context)
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), plaintext)
	assert.NotContains(t, recorder.Body.String(), "ciphertext")
	assert.Contains(t, recorder.Body.String(), created.Data.Endpoint.SecretHint)
}

func TestHandlerRejectsUnknownFieldsAndMembers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registerWebhookHandlerTestMappings()
	service, organizationID, actorID, _ := newWebhookServiceTest(t)
	handler := NewHandler(service)
	body := `{"name":"Receiver","url":"http://127.0.0.1:8080/hook","event_types":["webhook.test"],"headers":{}}`
	recorder, context := newWebhookHandlerContext(http.MethodPost, "/v1/webhook-endpoints", body, organizationID, actorID, domain.OrganizationRoleOwner)

	handler.CreateEndpoint(context)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), domain.CodeInvalidInput)

	recorder, context = newWebhookHandlerContext(http.MethodGet, "/v1/webhook-event-types", "", organizationID, actorID, domain.OrganizationRoleMember)
	handler.EventTypes(context)
	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), domain.CodePermissionDenied)
}

func TestHandlerRequiresEndpointVersionPrecondition(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registerWebhookHandlerTestMappings()
	service, organizationID, actorID, _ := newWebhookServiceTest(t)
	handler := NewHandler(service)
	body := `{"name":"Receiver","url":"http://127.0.0.1:8080/hook","event_types":["webhook.test"]}`
	recorder, context := newWebhookHandlerContext(http.MethodPatch, "/v1/webhook-endpoints/1", body, organizationID, actorID, domain.OrganizationRoleOwner)
	context.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.UpdateEndpoint(context)
	assert.Equal(t, http.StatusPreconditionRequired, recorder.Code)
	assert.Contains(t, recorder.Body.String(), domain.CodeWebhookPreconditionRequired)
}

func registerWebhookHandlerTestMappings() {
	response.DefaultErrorMapper.Register(domain.ErrPermissionDenied, http.StatusForbidden, domain.CodePermissionDenied)
	response.DefaultErrorMapper.Register(
		domain.ErrWebhookPreconditionRequired,
		http.StatusPreconditionRequired,
		domain.CodeWebhookPreconditionRequired,
	)
}

func TestHandlerRejectsAmbiguousIdempotencyHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, organizationID, actorID, _ := newWebhookServiceTest(t)
	handler := NewHandler(service)
	recorder, context := newWebhookHandlerContext(http.MethodPost, "/v1/webhook-endpoints/1/tests", "", organizationID, actorID, domain.OrganizationRoleOwner)
	context.Params = gin.Params{{Key: "id", Value: "1"}}
	context.Request.Header.Add("Idempotency-Key", "first")
	context.Request.Header.Add("Idempotency-Key", "second")

	handler.TestEndpoint(context)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), domain.CodeInvalidInput)
}

func newWebhookHandlerContext(
	method string,
	path string,
	body string,
	organizationID uint,
	actorID uint,
	role domain.OrganizationRole,
) (*httptest.ResponseRecorder, *gin.Context) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	requestContext := domain.WithOrganizationContext(request.Context(), domain.OrganizationContext{
		OrganizationID:   organizationID,
		OrganizationName: "Webhook Test",
		OrganizationSlug: "webhook-test",
		MembershipID:     1,
		UserID:           actorID,
		Role:             role,
	})
	context.Request = request.WithContext(requestContext)
	context.Set("userID", actorID)
	return recorder, context
}
