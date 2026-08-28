package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// parsePagination reads page/limit query params shared by every paginated
// list endpoint, defaulting to 1/10 and ignoring invalid or non-positive
// values. Upper-bound clamping (e.g. max 100 per page) is enforced
// downstream in the repository layer, not here.
func parsePagination(c *gin.Context) (page, limit int) {
	page, limit = 1, 10
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	return page, limit
}
