package auditlog

import (
	"context"
	"gin_auth_service/internal/domain/auditlog"

	"github.com/google/uuid"
)

// UseCase handles business scenarios for audit logging.
type UseCase interface {
	Create(ctx context.Context, log *auditlog.AuditLog) error
	GetAll(ctx context.Context, page, limit int, entityID *uuid.UUID) (AuditLogListResponse, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]ResponseDTO, error)
}
