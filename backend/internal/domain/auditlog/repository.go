package auditlog

import (
	"context"

	"uuid"
)

// Repository defines the interface for audit log persistence.
type Repository interface {
	Create(ctx context.Context, log *AuditLog) error
	CreateBatch(ctx context.Context, logs []AuditLog) error
	GetAll(ctx context.Context) ([]AuditLog, error)
	GetAllPaginated(ctx context.Context, page, limit int, entityID *uuid.UUID) ([]AuditLog, int64, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]AuditLog, error)
	GetByEntity(ctx context.Context, entity string) ([]AuditLog, error)
}
