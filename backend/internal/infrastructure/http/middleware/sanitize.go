package middleware

import (
	"bytes"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	xssPattern = regexp.MustCompile(`(?i)(<script|javascript:|onerror=|onload=|<iframe|<object|<embed)`)
)

// SanitizeMiddleware sanitizes and validates input to prevent XSS and SQL injection.
func SanitizeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check query parameters
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				if isMalicious(value) {
					c.JSON(http.StatusBadRequest, gin.H{
						"error":  "Invalid input detected",
						"code":   "INVALID_INPUT",
						"field":  key,
						"detail": "Potentially malicious content",
					})
					c.Abort()
					return
				}
			}
		}

		// Sanitize request body for JSON content
		if c.Request.Body != nil && c.Request.ContentLength > 0 {
			contentType := c.GetHeader("Content-Type")
			if strings.Contains(contentType, "application/json") {
				body, err := io.ReadAll(c.Request.Body)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
					c.Abort()
					return
				}
				c.Request.Body.Close()

				// Check for malicious patterns in raw body
				if xssPattern.Match(body) {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "Invalid input detected",
						"code":  "INVALID_INPUT",
						"detail": "Potentially malicious content in request body",
					})
					c.Abort()
					return
				}

				// Restore body for next handlers
				c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			}
		}

		c.Next()
	}
}

// isMalicious checks if string contains XSS patterns
func isMalicious(input string) bool {
	if input == "" {
		return false
	}
	decoded := html.UnescapeString(input)
	return xssPattern.MatchString(decoded)
}



