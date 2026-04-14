package middleware

import (
	"bytes"
	"encoding/json"
	"gin_auth_service/pkg/errors"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	// SQL injection patterns
	sqlInjectionPattern = regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute|;|--|/\*|\*/|')`)
	
	// XSS patterns
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
				if sqlInjectionPattern.Match(body) || xssPattern.Match(body) {
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

// isMalicious checks if string contains SQLi or XSS patterns
func isMalicious(input string) bool {
	if input == "" {
		return false
	}
	decoded := html.UnescapeString(input)
	return sqlInjectionPattern.MatchString(decoded) || xssPattern.MatchString(decoded)
}

// SanitizeString escapes HTML entities to prevent XSS
func SanitizeString(input string) string {
	if input == "" {
		return input
	}
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	// Escape HTML
	return html.EscapeString(input)
}

// SanitizeStruct recursively sanitizes string fields in a struct
func SanitizeStruct(data interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Simple string replacement for JSON content
	str := string(bytes)
	str = strings.ReplaceAll(str, "<", "\\u003c")
	str = strings.ReplaceAll(str, ">", "\\u003e")
	str = strings.ReplaceAll(str, "&", "\\u0026")

	return json.Unmarshal([]byte(str), data)
}

// ValidatePhone validates phone number format (E.164)
func ValidatePhone(phone string) error {
	pattern := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	if !pattern.MatchString(phone) {
		return errors.ErrInvalidUserID // or create specific error
	}
	return nil
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.ErrInvalidCredentials
	}
	// Check for at least one uppercase, one lowercase, one digit
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`\d`).MatchString(password)

	if !hasUpper || !hasLower || !hasDigit {
		return errors.ErrInvalidCredentials
	}
	return nil
}
