package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is the interface the login rate limiter requires from its storage backend.
// Defined here (consumer side) so the infrastructure layer depends on domain, not vice versa.
type RateLimiter interface {
	// Increment atomically increments the counter for key, setting window TTL on first call.
	// Returns the counter value after increment.
	Increment(ctx context.Context, key string, window time.Duration) (int64, error)
	// Reset deletes the counter (called on successful login).
	Reset(ctx context.Context, key string) error
}

// LoginRateLimitMiddleware provides brute-force protection.
// Limits: 5 failed attempts per 15 minutes per IP.
//
// Flow: INCR (atomic) → if over limit, block immediately → c.Next() → 200? Reset counter.
// On Redis failure the middleware fails open so a Redis outage never locks out users.
func LoginRateLimitMiddleware(limiter RateLimiter) gin.HandlerFunc {
	const (
		maxAttempts int64 = 5
		window            = 15 * time.Minute
	)

	return func(c *gin.Context) {
		key := fmt.Sprintf("rate_limit:login:%s", c.ClientIP())
		ctx := c.Request.Context()

		count, err := limiter.Increment(ctx, key, window)
		if err != nil {
			// Redis unavailable — fail open rather than blocking legitimate users.
			c.Next()
			return
		}

		if count > maxAttempts {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Too many login attempts. Please try again later.",
				"code":        "RATE_LIMIT_EXCEEDED",
				"retry_after": window.Seconds(),
			})
			c.Abort()
			return
		}

		c.Next()

		if c.Writer.Status() == http.StatusOK {
			limiter.Reset(ctx, key)
		}
	}
}
