package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/zgiai/zgo/pkg/pagination"
)

func TestPaginatorSmoke(t *testing.T) {
	p := pagination.NewPaginator([]string{"a", "b", "c"}, 10, 2, 3)
	p.SetPath("/api/users")

	assert.Equal(t, int64(10), p.Total())
	assert.Equal(t, 2, p.CurrentPage())
	assert.Equal(t, 3, p.PerPage())
	assert.Equal(t, 4, p.From())
	assert.Equal(t, 6, p.To())
	assert.True(t, p.HasMorePages())
	assert.Equal(t, "/api/users?page=3", *p.NextPageURL())
}

func TestFromContextReadsPaginationQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		req := pagination.FromContext(c)
		assert.Equal(t, 3, req.GetPage())
		assert.Equal(t, 25, req.GetPerPage())
		assert.Equal(t, "john", req.Keyword)
		assert.Equal(t, "created_at asc", req.GetOrderBy())
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test?page=3&per_page=25&keyword=john&sort=created_at&order=asc", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestRequestDefaultsAndBounds(t *testing.T) {
	req := pagination.NewRequest(-1, 1000).WithSort("created_at", "")

	assert.Equal(t, pagination.DefaultPage, req.GetPage())
	assert.Equal(t, pagination.MaxPerPage, req.GetPerPage())
	assert.Equal(t, "created_at desc", req.GetOrderBy())
}
