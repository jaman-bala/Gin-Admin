package handler

import (
	"gin_auth_service/internal/application/auditlog"
	"net/http"

	"github.com/gin-gonic/gin"
	"uuid"
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
// @ID getAuditLogs
// @Tags audit
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param entity_id query string false "Filter by entity ID"
// @Success 200 {object} auditlog.AuditLogListResponse
// @Router /api/v1/audit [get]
func (h *AuditHandler) GetAllLogs(c *gin.Context) {
	ctx := c.Request.Context()

	page, limit := parsePagination(c)

	var entityID *uuid.UUID
	if eID := c.Query("entity_id"); eID != "" {
		if parsedID, err := uuid.Parse(eID); err == nil {
			entityID = &parsedID
		}
	}

	result, err := h.usecase.GetAll(ctx, page, limit, entityID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
