package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/pkg/response"
)

func TestRouteConstraintFailureUsesUnifiedErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	r := New(engine)
	r.GET("/items/:id", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	}).WhereNumber("id")

	req := httptest.NewRequest(http.MethodGet, "/items/not-a-number", nil)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)

	var body response.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, http.StatusNotFound, body.Code)
	assert.Equal(t, response.ErrorCodeNotFound, body.ErrorCode)
	assert.Equal(t, "Invalid parameter: id", body.Message)
}
