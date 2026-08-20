package user

import (
	"context"
	"gin_auth_service/internal/application/file"

	"uuid"
)

// UseCase handles business scenarios for user management.
type UseCase interface {
	GetAll(ctx context.Context, page, limit int, search string, isActive *bool) (*UserListResponse, error)
	Create(ctx context.Context, req UserRequestDTO, photo *file.FileUpload) (*UserResponseDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (*UserResponseDTO, error)
	GetByPhone(ctx context.Context, phone string) (*UserResponseDTO, error)
	GetMe(ctx context.Context, id uuid.UUID) (*UserResponseDTO, error)
	// PatchSelf is used by users updating their own profile. Role and IsActive
	// cannot be changed through this method — use Patch (admin) for that.
	PatchSelf(ctx context.Context, id uuid.UUID, req UserSelfUpdateDTO, photo *file.FileUpload) (*UserResponseDTO, error)
	Patch(ctx context.Context, id uuid.UUID, req UserUpdateDTO, photo *file.FileUpload) (*UserResponseDTO, error)
	Delete(ctx context.Context, id uuid.UUID) error
}