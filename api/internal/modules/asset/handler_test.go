package asset

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerDeleteReturnsNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, ownerID := newAssetServiceTest(t)
	intent, err := service.CreateUploadIntent(
		t.Context(),
		ownerID,
		"asset-handler-delete",
		"private.txt",
		"text/plain",
		7,
	)
	require.NoError(t, err)
	handler := &Handler{service: service}

	for range 2 {
		response := deleteAssetRequest(handler, intent.Asset.ID, &ownerID)
		assert.Equal(t, http.StatusNoContent, response.Code)
		assert.Empty(t, response.Body.String())
		assert.Equal(t, "private, no-store", response.Header().Get("Cache-Control"))
	}
}

func TestHandlerDeleteRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, _ := newAssetServiceTest(t)
	response := deleteAssetRequest(&Handler{service: service}, "019bf6d8-17c5-7a98-a084-6d45793f5f0c", nil)

	assert.Equal(t, http.StatusUnauthorized, response.Code)
	var body struct {
		ErrorCode string `json:"error_code"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, "AUTH.UNAUTHORIZED", body.ErrorCode)
}

func deleteAssetRequest(handler *Handler, assetID string, userID *uint) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	router := gin.New()
	router.DELETE("/assets/:id", func(c *gin.Context) {
		if userID != nil {
			c.Set("userID", *userID)
		}
		handler.Delete(c)
	})
	request := httptest.NewRequest(http.MethodDelete, "/assets/"+assetID, nil)
	router.ServeHTTP(recorder, request)
	return recorder
}
