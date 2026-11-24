package middleware

import (
	"context"
	"fmt"
	"log/slog"
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
	// Release gives back one slot taken by Increment (called on success).
	Release(ctx context.Context, key string) error
}

const (
	// loginIPMaxFailures is deliberately generous: it is the broad net against
	// one address spraying many accounts, and must not punish an office behind
	// a shared NAT. The strict per-account limit lives in the auth use case.
	loginIPMaxFailures int64 = 20
	loginIPWindow            = 15 * time.Minute
)

// LoginRateLimitMiddleware limits FAILED login attempts per client IP:
// loginIPMaxFailures per loginIPWindow.
//
// Flow: INCR (atomic, reserves a slot) -> over the limit? 429 -> c.Next() ->
// 2xx? give that one slot back. Only failures therefore accumulate. A success
// must never reset the counter: otherwise an attacker interleaves a login to
// their own account with their guesses and is never blocked.
//
// On Redis failure the middleware fails open so a Redis outage never locks out users.
func LoginRateLimitMiddleware(limiter RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("rate_limit:login:ip:%s", c.ClientIP())
		ctx := c.Request.Context()

		count, err := limiter.Increment(ctx, key, loginIPWindow)
		if err != nil {
			slog.Warn("login rate limiter unavailable, failing open", "error", err)
			c.Next()
			return
		}

		if count > loginIPMaxFailures {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Too many login attempts. Please try again later.",
				"code":        "RATE_LIMIT_EXCEEDED",
				"retry_after": loginIPWindow.Seconds(),
			})
			c.Abort()
			return
		}

		c.Next()

		if status := c.Writer.Status(); status >= 200 && status < 300 {
			if err := limiter.Release(ctx, key); err != nil {
				// Not fatal: the slot simply stays used until the window ends.
				slog.Warn("rate limit release failed", "error", err)
			}
		}
	}
}
