package auditlog

import (
	"gin_auth_service/internal/domain/auditlog"
	"time"

	"uuid"
)

// ResponseDTO represents an audit log entry in a response.
type ResponseDTO struct {
	ID        uint      `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Action    string    `json:"action"`
	Entity    string    `json:"entity"`
	EntityID  uuid.UUID `json:"entity_id"`
	Data      string    `json:"description"`
	Status    int       `json:"status"`
	ClientIP  string    `json:"client_ip"`
	UserAgent string    `json:"user_agent"`
	CreatedAt string    `json:"created_at"`
}

// FromModel maps an audit log entity to a response DTO.
func (dto *ResponseDTO) FromModel(log auditlog.AuditLog) {
	dto.ID = log.ID
	dto.UserID = log.UserID
	dto.Action = log.Action
	dto.Entity = log.Entity
	dto.EntityID = log.EntityID
	dto.Data = log.Data
	dto.Status = log.Status
	dto.ClientIP = log.ClientIP
	dto.UserAgent = log.UserAgent
	dto.CreatedAt = log.CreatedAt.Format(time.RFC3339)
}

// AuditLogListResponse represents a paginated list of audit logs.
type AuditLogListResponse struct {
	Logs  []ResponseDTO `json:"logs"`
	Total int64         `json:"total"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
}
