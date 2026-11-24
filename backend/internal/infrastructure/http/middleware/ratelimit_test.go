package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// memLimiter mimics the Redis counter semantics: Increment takes a slot,
// Release gives one back and drops the key at zero.
type memLimiter struct {
	mu     sync.Mutex
	n      map[string]int64
	incErr error
	relErr error
}

func newLimiter() *memLimiter { return &memLimiter{n: map[string]int64{}} }

func (m *memLimiter) Increment(_ context.Context, k string, _ time.Duration) (int64, error) {
	if m.incErr != nil {
		return 0, m.incErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.n[k]++
	return m.n[k], nil
}

func (m *memLimiter) Release(_ context.Context, k string) error {
	if m.relErr != nil {
		return m.relErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.n[k] > 0 {
		m.n[k]--
	}
	if m.n[k] == 0 {
		delete(m.n, k)
	}
	return nil
}

// loginServer returns a send func: send(ip, good) performs one login attempt
// and returns the HTTP status. A "good" attempt answers 200, a bad one 401.
func loginServer(l RateLimiter) func(ip string, good bool) int {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/login", LoginRateLimitMiddleware(l), func(c *gin.Context) {
		if c.GetHeader("X-Good") == "1" {
			c.Status(http.StatusOK)
		} else {
			c.Status(http.StatusUnauthorized)
		}
	})
	return func(ip string, good bool) int {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = ip + ":12345"
		if good {
			req.Header.Set("X-Good", "1")
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
}

func TestLoginRateLimit_BlocksAfterMaxFailures(t *testing.T) {
	send := loginServer(newLimiter())

	for i := int64(1); i <= loginIPMaxFailures; i++ {
		if code := send("203.0.113.1", false); code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i, code)
		}
	}
	if code := send("203.0.113.1", false); code != http.StatusTooManyRequests {
		t.Fatalf("attempt %d: status = %d, want 429", loginIPMaxFailures+1, code)
	}
}

// The original bypass. An attacker with a valid account of their own
// interleaves it with guesses; the counter must keep growing.
func TestLoginRateLimit_SuccessDoesNotResetTheCounter(t *testing.T) {
	send := loginServer(newLimiter())

	badGuesses, blocked := 0, 0
	for i := 0; i < 200; i++ {
		if i%5 == 4 {
			send("203.0.113.2", true) // own successful login
			continue
		}
		if send("203.0.113.2", false) == http.StatusTooManyRequests {
			blocked++
		} else {
			badGuesses++
		}
	}
	if blocked == 0 {
		t.Fatal("the attacker was never blocked: successful logins reset the counter")
	}
	if badGuesses > int(loginIPMaxFailures) {
		t.Fatalf("%d bad guesses got through, limit is %d", badGuesses, loginIPMaxFailures)
	}
}

func TestLoginRateLimit_SuccessesAloneNeverBlock(t *testing.T) {
	send := loginServer(newLimiter())

	for i := 0; i < int(loginIPMaxFailures)*5; i++ {
		if code := send("203.0.113.3", true); code != http.StatusOK {
			t.Fatalf("successful login %d: status = %d, want 200", i+1, code)
		}
	}
}

func TestLoginRateLimit_IsPerIP(t *testing.T) {
	send := loginServer(newLimiter())

	for i := int64(0); i <= loginIPMaxFailures; i++ {
		send("203.0.113.4", false)
	}
	if code := send("203.0.113.4", false); code != http.StatusTooManyRequests {
		t.Fatalf("offender: status = %d, want 429", code)
	}
	if code := send("203.0.113.99", true); code != http.StatusOK {
		t.Fatalf("another client: status = %d, want 200", code)
	}
}

func TestLoginRateLimit_RedisFailuresFailOpen(t *testing.T) {
	t.Run("increment error", func(t *testing.T) {
		l := newLimiter()
		l.incErr = errors.New("redis down")
		send := loginServer(l)
		for i := 0; i < 50; i++ {
			if code := send("203.0.113.5", false); code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401: an outage must not block anyone", code)
			}
		}
	})
	t.Run("release error does not break a successful login", func(t *testing.T) {
		l := newLimiter()
		l.relErr = errors.New("redis down")
		if code := loginServer(l)("203.0.113.6", true); code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
	})
}
