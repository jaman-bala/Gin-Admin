package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimitMiddleware provides a basic rate limit using a map.
// Note: In production, consider using Redis for distributed rate limiting.
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	// Simple in-memory storage for rate limiting
	clients := make(map[string][]time.Time)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		// Cleanup old entries
		if times, exists := clients[clientIP]; exists {
			var validTimes []time.Time
			for _, t := range times {
				if now.Sub(t) < time.Minute {
					validTimes = append(validTimes, t)
				}
			}
			clients[clientIP] = validTimes
		}

		// Check limit
		if len(clients[clientIP]) >= requestsPerMinute {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Превышен лимит запросов",
				"code":        "RATE_LIMIT_EXCEEDED",
				"retry_after": "60 seconds",
			})
			c.Abort()
			return
		}

		// Add current request
		clients[clientIP] = append(clients[clientIP], now)

		c.Next()
	}
}
