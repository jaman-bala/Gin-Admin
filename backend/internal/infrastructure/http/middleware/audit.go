package middleware

import (
	"fmt"
	"gin_auth_service/internal/application/auditlog"
	domainAudit "gin_auth_service/internal/domain/auditlog"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuditMiddleware logs user actions for auditing purposes.
func AuditMiddleware(service auditlog.UseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")

		c.Next()

		if method == "GET" {
			return
		}

		status := c.Writer.Status()

		var userID = uuid.Nil
		if uid, exists := c.Get("id"); exists {
			if uuidVal, ok := uid.(uuid.UUID); ok {
				userID = uuidVal
			}
		}

		var entityID = uuid.Nil
		if idParam := c.Param("id"); idParam != "" {
			if parsedID, err := uuid.Parse(idParam); err == nil {
				entityID = parsedID
			}
		}

		entityType := "Unknown"
		if strings.Contains(path, "/users") || strings.Contains(path, "/dashboard") {
			entityType = "User"
		} else if strings.Contains(path, "/auth") {
			entityType = "Auth"
		} else if strings.Contains(path, "/audit") {
			entityType = "Audit"
		}

		var actionDescription string
		switch method {
		case "GET":
			actionDescription = "Просмотр"
		case "POST":
			actionDescription = "Создание"
		case "PUT", "PATCH":
			actionDescription = "Обновление"
		case "DELETE":
			actionDescription = "Удаление"
		default:
			actionDescription = method
		}

		logData := fmt.Sprintf("Действие: %s %s | Путь: %s", actionDescription, entityType, path)

		// Capture entityID from context if it was set by the handler (e.g. for POST requests)
		if eid, exists := c.Get("entity_id"); exists {
			if uuidVal, ok := eid.(uuid.UUID); ok {
				entityID = uuidVal
			}
		}

		log := &domainAudit.AuditLog{
			UserID:    userID,
			Action:    method,
			Entity:    entityType,
			EntityID:  entityID,
			ClientIP:  clientIP,
			UserAgent: userAgent,
			Data:      logData,
			Status:    status,
			CreatedAt: time.Now(),
		}

		if err := service.Create(c.Request.Context(), log); err != nil {
			slog.Error("failed to create audit log", "error", err, "userID", userID, "entity", entityType)
		}
	}
}
