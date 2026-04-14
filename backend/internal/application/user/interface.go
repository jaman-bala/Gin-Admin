package user

import (
	"context"

	"github.com/google/uuid"
)

// UseCase handles business scenarios for user management.
type UseCase interface {
	GetAll(ctx context.Context, page, limit int, search string, isActive *bool) (*UserListResponse, error)
	Create(ctx context.Context, req UserRequestDTO, photoFile *FileUpload) (*UserResponseDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (*UserResponseDTO, error)
	GetByPhone(ctx context.Context, phone string) (*UserResponseDTO, error)
	Patch(ctx context.Context, id uuid.UUID, request UserUpdateDTO, photoFile *FileUpload) (*UserResponseDTO, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
