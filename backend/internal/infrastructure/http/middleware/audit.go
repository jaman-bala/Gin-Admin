package middleware

import (
	"fmt"
	"gin_auth_service/internal/application/auditlog"
	domainAudit "gin_auth_service/internal/domain/auditlog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"uuid"
)

// AuditMiddleware records user actions for auditing purposes. Entries are
// enqueued into an async recorder, so the request path never waits on the DB.
func AuditMiddleware(recorder auditlog.Recorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")

		c.Next()

		status := c.Writer.Status()

		var userID = uuid.Nil()
		if uid, exists := c.Get("id"); exists {
			if uuidVal, ok := uid.(uuid.UUID); ok {
				userID = uuidVal
			}
		}

		var entityID = uuid.Nil()
		if idParam := c.Param("id"); idParam != "" {
			if parsedID, err := uuid.Parse(idParam); err == nil {
				entityID = parsedID
			}
		} else if strings.HasSuffix(path, "/me") {
			// /users/me — entity is the authenticated user themselves
			entityID = userID
		}

		entityType := "Unknown"
		if strings.Contains(path, "/users") {
			entityType = "User"
		} else if strings.Contains(path, "/auth") {
			entityType = "Auth"
		} else if strings.Contains(path, "/audit") {
			entityType = "Audit"
		}

		var actionDescription string
		switch method {
		case "GET":
			actionDescription = "View"
		case "POST":
			actionDescription = "Create"
		case "PUT", "PATCH":
			actionDescription = "Update"
		case "DELETE":
			actionDescription = "Delete"
		default:
			actionDescription = method
		}

		requestID, _ := c.Get(RequestIDKey)
		logData := fmt.Sprintf("Action: %s %s | Path: %s | RequestID: %s", actionDescription, entityType, path, requestID)

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

		recorder.Record(log)
	}
}
