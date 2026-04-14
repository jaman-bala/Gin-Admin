package handler

import (
	"gin_auth_service/internal/application/auditlog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuditHandler handles audit log HTTP requests.
type AuditHandler struct {
	usecase auditlog.UseCase
}

// NewAuditHandler creates a new instance of AuditHandler.
func NewAuditHandler(usecase auditlog.UseCase) *AuditHandler {
	return &AuditHandler{
		usecase: usecase,
	}
}

// GetAllLogs godoc
// @Summary Get all user actions
// @Tags audit
// @Security BearerAuth
// @Produce json
// @Router /api/v1/audit [get]
func (h *AuditHandler) GetAllLogs(c *gin.Context) {
	ctx := c.Request.Context()

	page := 1
	limit := 10
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	var entityID *uuid.UUID
	if eID := c.Query("entity_id"); eID != "" {
		if parsedID, err := uuid.Parse(eID); err == nil {
			entityID = &parsedID
		}
	}

	result, err := h.usecase.GetAll(ctx, page, limit, entityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
