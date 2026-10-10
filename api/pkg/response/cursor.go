package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CursorMeta describes one keyset page. Unlike offset pagination it carries no total, so a page
// costs the same however many records match.
type CursorMeta struct {
	PerPage    int     `json:"per_page"`
	HasMore    bool    `json:"has_more"`
	NextCursor *string `json:"next_cursor"`
}

// CursorPageResponse is the envelope for a keyset page.
type CursorPageResponse struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    any        `json:"data"`
	Meta    CursorMeta `json:"meta"`
}

// SuccessCursorPage writes one keyset page. An empty nextCursor means there is no further page.
func SuccessCursorPage(c *gin.Context, items any, perPage int, nextCursor string) {
	meta := CursorMeta{PerPage: perPage}
	if nextCursor != "" {
		meta.HasMore = true
		meta.NextCursor = &nextCursor
	}
	c.JSON(http.StatusOK, CursorPageResponse{Code: 0, Message: "success", Data: items, Meta: meta})
}
