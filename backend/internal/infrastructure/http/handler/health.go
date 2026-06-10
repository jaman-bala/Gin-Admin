package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Pinger is satisfied by any dependency that can report its own liveness.
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler checks liveness of all infrastructure dependencies.
type HealthHandler struct {
	db      *sqlx.DB
	redis   Pinger
	storage Pinger
}

func NewHealthHandler(db *sqlx.DB, redis, storage Pinger) *HealthHandler {
	return &HealthHandler{db: db, redis: redis, storage: storage}
}

// Check godoc
// @Summary Health check
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Router /health [get]
func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	checks := make(map[string]string, 3)
	healthy := true

	check := func(name string, ping func() error) {
		if err := ping(); err != nil {
			checks[name] = "error: " + err.Error()
			healthy = false
		} else {
			checks[name] = "ok"
		}
	}

	check("database", func() error { return h.db.PingContext(ctx) })
	check("redis", func() error { return h.redis.Ping(ctx) })
	check("storage", func() error { return h.storage.Ping(ctx) })

	status := "ok"
	httpStatus := http.StatusOK
	if !healthy {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{"status": status, "checks": checks})
}