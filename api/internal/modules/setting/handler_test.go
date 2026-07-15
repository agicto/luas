package setting

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/pkg/response"
)

func TestPublicSettingsUseAggregateETagAndRevalidation(t *testing.T) {
	fixture := newSettingTestFixture(t)
	handler := &Handler{service: fixture.service}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/settings/public", handler.PublicApp)

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/settings/public", nil))
	require.Equal(t, http.StatusOK, first.Code)
	assert.Equal(t, "public, max-age=60, stale-while-revalidate=300", first.Header().Get("Cache-Control"))
	etag := first.Header().Get("ETag")
	require.NotEmpty(t, etag)
	var payload struct {
		Data []SettingResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &payload))
	require.Len(t, payload.Data, 2)
	assert.Equal(t, "branding.display_name", payload.Data[0].Key)

	second := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/settings/public", nil)
	request.Header.Set("If-None-Match", etag)
	router.ServeHTTP(second, request)
	assert.Equal(t, http.StatusNotModified, second.Code)
	assert.Empty(t, second.Body.String())
	assert.Equal(t, etag, second.Header().Get("ETag"))
}

func TestUserSettingMutationRequiresCanonicalVersionAndRejectsStaleWriter(t *testing.T) {
	fixture := newSettingTestFixture(t)
	handler := &Handler{service: fixture.service}
	registerSettingTestErrors()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PATCH("/settings/user/:key", func(c *gin.Context) {
		c.Set("userID", fixture.user.ID)
		handler.UserSet(c)
	})
	router.DELETE("/settings/user/:key", func(c *gin.Context) {
		c.Set("userID", fixture.user.ID)
		handler.UserReset(c)
	})

	missing := settingMutationRequest(http.MethodPatch, "/settings/user/localization.locale", `{"value":"zh-Hans"}`, "")
	missingResponse := httptest.NewRecorder()
	router.ServeHTTP(missingResponse, missing)
	assert.Equal(t, http.StatusPreconditionRequired, missingResponse.Code)
	assert.Equal(t, domain.CodeSettingPreconditionRequired, responseErrorCode(t, missingResponse))

	created := settingMutationRequest(http.MethodPatch, "/settings/user/localization.locale", `{"value":"zh-Hans"}`, `"setting-v0"`)
	createdResponse := httptest.NewRecorder()
	router.ServeHTTP(createdResponse, created)
	require.Equal(t, http.StatusOK, createdResponse.Code)
	assert.Equal(t, `"setting-v1"`, createdResponse.Header().Get("ETag"))
	assert.Equal(t, "private, no-store", createdResponse.Header().Get("Cache-Control"))

	stale := settingMutationRequest(http.MethodPatch, "/settings/user/localization.locale", `{"value":"en-US"}`, `"setting-v0"`)
	staleResponse := httptest.NewRecorder()
	router.ServeHTTP(staleResponse, stale)
	assert.Equal(t, http.StatusPreconditionFailed, staleResponse.Code)
	assert.Equal(t, domain.CodeSettingVersionConflict, responseErrorCode(t, staleResponse))

	unknownField := settingMutationRequest(http.MethodPatch, "/settings/user/localization.locale", `{"value":"en-US","extra":true}`, `"setting-v1"`)
	unknownFieldResponse := httptest.NewRecorder()
	router.ServeHTTP(unknownFieldResponse, unknownField)
	assert.Equal(t, http.StatusUnprocessableEntity, unknownFieldResponse.Code)
	assert.Equal(t, domain.CodeSettingInvalidValue, responseErrorCode(t, unknownFieldResponse))

	reset := settingMutationRequest(http.MethodDelete, "/settings/user/localization.locale", "", `"setting-v1"`)
	resetResponse := httptest.NewRecorder()
	router.ServeHTTP(resetResponse, reset)
	assert.Equal(t, http.StatusNoContent, resetResponse.Code)
	assert.Equal(t, `"setting-v2"`, resetResponse.Header().Get("ETag"))
}

func TestOrganizationMemberCanReadButCannotMutateSettings(t *testing.T) {
	fixture := newSettingTestFixture(t)
	handler := &Handler{service: fixture.service}
	registerSettingTestErrors()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	memberContext := domain.OrganizationContext{
		OrganizationID:   fixture.organization.ID,
		OrganizationName: fixture.organization.Name,
		OrganizationSlug: fixture.organization.Slug,
		MembershipID:     9,
		UserID:           fixture.user.ID,
		Role:             domain.OrganizationRoleMember,
	}
	router.GET("/organization-settings", func(c *gin.Context) {
		c.Request = c.Request.WithContext(domain.WithOrganizationContext(context.Background(), memberContext))
		handler.OrganizationList(c)
	})
	router.PATCH("/organization-settings/:key", func(c *gin.Context) {
		c.Request = c.Request.WithContext(domain.WithOrganizationContext(context.Background(), memberContext))
		handler.OrganizationSet(c)
	})

	read := httptest.NewRecorder()
	router.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/organization-settings", nil))
	assert.Equal(t, http.StatusOK, read.Code)

	write := httptest.NewRecorder()
	request := settingMutationRequest(http.MethodPatch, "/organization-settings/localization.locale", `{"value":"zh-Hans"}`, `"setting-v0"`)
	router.ServeHTTP(write, request)
	assert.Equal(t, http.StatusForbidden, write.Code)
	assert.Equal(t, domain.CodePermissionDenied, responseErrorCode(t, write))
}

func settingMutationRequest(method string, path string, body string, etag string) *http.Request {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if etag != "" {
		request.Header.Set("If-Match", etag)
	}
	return request
}

func registerSettingTestErrors() {
	response.DefaultErrorMapper.Register(domain.ErrSettingNotFound, http.StatusNotFound, domain.CodeSettingNotFound)
	response.DefaultErrorMapper.Register(domain.ErrSettingInvalidValue, http.StatusUnprocessableEntity, domain.CodeSettingInvalidValue)
	response.DefaultErrorMapper.Register(domain.ErrSettingVersionConflict, http.StatusPreconditionFailed, domain.CodeSettingVersionConflict)
	response.DefaultErrorMapper.Register(domain.ErrSettingPreconditionRequired, http.StatusPreconditionRequired, domain.CodeSettingPreconditionRequired)
	response.DefaultErrorMapper.Register(domain.ErrPermissionDenied, http.StatusForbidden, domain.CodePermissionDenied)
}

func responseErrorCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		ErrorCode string `json:"error_code"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body.ErrorCode
}
