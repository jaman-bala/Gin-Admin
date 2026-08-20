package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodySizeLimit caps the request body size. Oversized bodies make handlers'
// reads fail with http.MaxBytesError, which Gin surfaces as a 4xx instead of
// letting a client exhaust server memory.
func BodySizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
