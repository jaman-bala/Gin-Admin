package auditlog

import (
	"context"

	"uuid"
)

// UseCase handles business scenarios for audit logging.
type UseCase interface {
	GetAll(ctx context.Context, page, limit int, entityID *uuid.UUID) (AuditLogListResponse, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]ResponseDTO, error)
}
