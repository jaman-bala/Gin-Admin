package middleware

import (
	"fmt"
	"gin_auth_service/internal/domain/token"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// LoginRateLimitMiddleware provides brute-force protection using Redis.
// Limits: 5 attempts per 15 minutes per IP.
func LoginRateLimitMiddleware(repo token.Repository) gin.HandlerFunc {
	const (
		maxAttempts = 5
		window      = 15 * time.Minute
	)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		key := fmt.Sprintf("rate_limit:login:%s", clientIP)

		ctx := c.Request.Context()

		// Get current attempts count
		val, err := repo.Get(ctx, key)
		attempts := 0
		if err == nil && val != "" {
			attempts, _ = strconv.Atoi(val)
		}

		// Check if limit exceeded
		if attempts >= maxAttempts {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Слишком много попыток входа. Попробуйте позже.",
				"code":        "RATE_LIMIT_EXCEEDED",
				"retry_after": window.Seconds(),
			})
			c.Abort()
			return
		}

		// Increment attempts
		newAttempts := attempts + 1
		repo.Set(ctx, key, newAttempts, window)

		c.Next()
	}
}
