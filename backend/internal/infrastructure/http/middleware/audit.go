package middleware

import (
	"fmt"
	"gin_auth_service/internal/application/auditlog"
	domainAudit "gin_auth_service/internal/domain/auditlog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"uuid"
)

const (
	// maxUserAgentRunes matches audit_logs.user_agent VARCHAR(255), which
	// PostgreSQL counts in characters, not bytes.
	maxUserAgentRunes = 255
	// maxAuditDataRunes bounds the free-text description (path + request id).
	maxAuditDataRunes = 1024
)

// sanitizeText makes client-controlled text safe to store in PostgreSQL and
// bounds its size: invalid UTF-8 and NUL bytes are rejected by the database,
// and an over-long value fails the whole multi-row INSERT. Truncation is by
// characters so a multibyte rune is never cut in half.
func sanitizeText(s string, maxRunes int) string {
	s = strings.ToValidUTF8(s, "\uFFFD")
	s = strings.ReplaceAll(s, "\x00", "")
	if utf8.RuneCountInString(s) > maxRunes {
		s = string([]rune(s)[:maxRunes])
	}
	return s
}

// AuditMiddleware records user actions for auditing purposes. Entries are
// enqueued into an async recorder, so the request path never waits on the DB.
func AuditMiddleware(recorder auditlog.Recorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := sanitizeText(c.Request.URL.Path, maxAuditDataRunes)
		method := c.Request.Method
		clientIP := c.ClientIP()
		userAgent := sanitizeText(c.GetHeader("User-Agent"), maxUserAgentRunes)

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
			Data:      sanitizeText(logData, maxAuditDataRunes),
			Status:    status,
			CreatedAt: time.Now(),
		}

		recorder.Record(log)
	}
}
