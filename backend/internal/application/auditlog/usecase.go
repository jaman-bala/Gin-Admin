package auditlog

import (
	"context"
	"gin_auth_service/internal/domain/auditlog"

	"uuid"
)


type usecase struct {
	repo auditlog.Repository
}

// NewUseCase creates a new instance of UseCase.
func NewUseCase(repo auditlog.Repository) UseCase {
	return &usecase{
		repo: repo,
	}
}

func (uc *usecase) GetAll(ctx context.Context, page, limit int, entityID *uuid.UUID) (AuditLogListResponse, error) {
	logs, total, err := uc.repo.GetAllPaginated(ctx, page, limit, entityID)
	if err != nil {
		return AuditLogListResponse{}, err
	}

	var dtos []ResponseDTO
	for _, l := range logs {
		var dto ResponseDTO
		dto.FromModel(l)
		dtos = append(dtos, dto)
	}

	return AuditLogListResponse{
		Logs:  dtos,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (uc *usecase) GetByUserID(ctx context.Context, userID uuid.UUID) ([]ResponseDTO, error) {
	logs, err := uc.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var response []ResponseDTO
	for _, log := range logs {
		var dto ResponseDTO
		dto.FromModel(log)
		response = append(response, dto)
	}
	return response, nil
}
