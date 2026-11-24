package middleware

import (
	"github.com/gin-gonic/gin"
	"uuid"
)

const (
	RequestIDHeader = "X-Request-ID"
	RequestIDKey    = "request_id"
)

const maxRequestIDLen = 64

// validRequestID reports whether a client-supplied ID is safe to propagate
// into logs, response headers and the audit trail: short and made only of
// [A-Za-z0-9._-]. Anything else (spaces, newlines, non-ASCII, megabyte-long
// values) would allow log forging and unbounded storage by an anonymous client.
func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for i := 0; i < len(id); i++ {
		ch := id[i]
		switch {
		case ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9',
			ch == '-', ch == '_', ch == '.':
		default:
			return false
		}
	}
	return true
}

// RequestIDMiddleware generates a unique request ID for each request and adds it to the context.
// A client-supplied ID is reused only if it passes validRequestID; otherwise a new one is generated.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if !validRequestID(requestID) {
			requestID = uuid.NewV7().String()
		}

		c.Set(RequestIDKey, requestID)
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}
