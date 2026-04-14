package handler

import (
	"gin_auth_service/internal/application/analytics"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AnalyticsHandler handles analytics and reporting HTTP requests.
type AnalyticsHandler struct {
	usecase analytics.UseCase
}

// NewAnalyticsHandler creates a new instance of AnalyticsHandler.
func NewAnalyticsHandler(usecase analytics.UseCase) *AnalyticsHandler {
	return &AnalyticsHandler{
		usecase: usecase,
	}
}

// GetUserStats godoc
// @Summary Get user statistics
// @Tags analytics
// @Security BearerAuth
// @Produce json
// @Success 200 {object} analytics.UserStats
// @Router /api/v1/users/stats [get]
func (h *AnalyticsHandler) GetUserStats(c *gin.Context) {
	ctx := c.Request.Context()
	stats, err := h.usecase.GetUserStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
