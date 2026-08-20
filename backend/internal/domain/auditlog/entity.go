package auditlog

import (
	"time"

	"uuid"
)

// AuditLog represents a single entry in the audit trail.
type AuditLog struct {
	ID        uint
	UserID    uuid.UUID
	Action    string
	Entity    string
	EntityID  uuid.UUID
	ClientIP  string
	UserAgent string
	Data      string
	Status    int
	CreatedAt time.Time
}
