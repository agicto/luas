package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type bindingProbe struct {
	Username string   `json:"username" binding:"required,min=3,max=50,excludesrune=@"`
	Role     string   `json:"role" binding:"omitempty,oneof=admin member"`
	Tags     []string `json:"tags" binding:"max=2"`
}

type bindingQueryProbe struct {
	Status string `form:"status" binding:"omitempty,oneof=active disabled"`
}

func bindingEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/probe", func(c *gin.Context) {
		var request bindingProbe
		if !BindJSON(c, &request) {
			return
		}
		c.Status(http.StatusNoContent)
	})
	engine.GET("/probe", func(c *gin.Context) {
		var query bindingQueryProbe
		if !BindQuery(c, &query) {
			return
		}
		c.Status(http.StatusNoContent)
	})
	engine.POST("/limited", func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16)
		var request bindingProbe
		if !BindJSON(c, &request) {
			return
		}
		c.Status(http.StatusNoContent)
	})
	return engine
}

func send(engine *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	return recorder
}

func TestBindJSONReportsFieldFailuresAs422WithWireNames(t *testing.T) {
	recorder := send(bindingEngine(), http.MethodPost, "/probe", `{"username":"a@b","role":"owner","tags":["x","y","z"]}`)
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	var body struct {
		ErrorCode string              `json:"error_code"`
		Errors    map[string][]string `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "COMMON.VALIDATION_FAILED", body.ErrorCode)
	assert.Equal(t, []string{`must not contain "@"`}, body.Errors["username"])
	assert.Equal(t, []string{"must be one of: admin, member"}, body.Errors["role"])
	assert.Equal(t, []string{"must contain at most 2 items"}, body.Errors["tags"])

	missing := send(bindingEngine(), http.MethodPost, "/probe", `{}`)
	require.Equal(t, http.StatusUnprocessableEntity, missing.Code)
	assert.Contains(t, missing.Body.String(), `"username":["is required"]`)
}

func TestBindJSONKeepsMalformedAndOversizedBodiesApart(t *testing.T) {
	malformed := send(bindingEngine(), http.MethodPost, "/probe", `{"username":`)
	assert.Equal(t, http.StatusBadRequest, malformed.Code)
	assert.Contains(t, malformed.Body.String(), "COMMON.INVALID_INPUT")

	oversized := send(bindingEngine(), http.MethodPost, "/limited", `{"username":"a-very-long-name-indeed"}`)
	assert.Equal(t, http.StatusRequestEntityTooLarge, oversized.Code)
	assert.Contains(t, oversized.Body.String(), "COMMON.REQUEST_TOO_LARGE")
}

func TestBindQueryFailuresStayInvalidInput(t *testing.T) {
	recorder := send(bindingEngine(), http.MethodGet, "/probe?status=exploded", "")
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "COMMON.INVALID_INPUT")
}

func TestParseIDRejectsZero(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/items/:id", func(c *gin.Context) {
		if _, ok := ParseID(c, "id"); ok {
			c.Status(http.StatusNoContent)
		}
	})
	assert.Equal(t, http.StatusBadRequest, send(engine, http.MethodGet, "/items/0", "").Code)
	assert.Equal(t, http.StatusNoContent, send(engine, http.MethodGet, "/items/7", "").Code)
}
