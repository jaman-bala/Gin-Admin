package user

import (
	"context"
	"fmt"
	"gin_auth_service/internal/application/file"
	domainUser "gin_auth_service/internal/domain/user"
	"gin_auth_service/internal/pkg/hash"
	"gin_auth_service/internal/pkg/utils"
	"gin_auth_service/pkg/errors"
	"log/slog"
	"time"

	"uuid"
)

type usecase struct {
	repo        domainUser.Repository
	fileService file.UseCase
}

func NewUseCase(repo domainUser.Repository, fileService file.UseCase) UseCase {
	return &usecase{repo: repo, fileService: fileService}
}

func (uc *usecase) GetAll(ctx context.Context, page, limit int, search string, isActive *bool) (*UserListResponse, error) {
	params := domainUser.FilterParams{
		Page:     page,
		Limit:    limit,
		Search:   search,
		IsActive: isActive,
	}

	users, total, err := uc.repo.GetWithFilters(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	dtos := make([]*UserResponseDTO, 0, len(users))
	for _, u := range users {
		var dto UserResponseDTO
		dto.FromModel(u)
		uc.enrichDTO(ctx, &dto)
		dtos = append(dtos, &dto)
	}

	return &UserListResponse{
		Users: dtos,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (uc *usecase) Create(ctx context.Context, req UserRequestDTO, photo *file.FileUpload) (*UserResponseDTO, error) {
	req.Phone = utils.NormalizePhone(req.Phone)
	hashedPassword, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	u := &domainUser.User{
		ID:         uuid.NewV7(),
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		MiddleName: req.MiddleName,
		Phone:      req.Phone,
		Password:   hashedPassword,
		Role:       req.Role,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if photo != nil {
		filePath, err := uc.fileService.UploadFile(ctx, photo, "user")
		if err != nil {
			return nil, fmt.Errorf("error saving photo: %w", err)
		}
		u.Photo = filePath
	}

	// Unique phone constraint is enforced by the DB; the repo maps the
	// violation to ErrConflict, which propagates here without wrapping.
	if err := uc.repo.Create(ctx, u); err != nil {
		return nil, err
	}

	var dto UserResponseDTO
	dto.FromModel(u)
	uc.enrichDTO(ctx, &dto)
	return &dto, nil
}

func (uc *usecase) GetByID(ctx context.Context, id uuid.UUID) (*UserResponseDTO, error) {
	u, err := uc.repo.GetID(ctx, id)
	if err != nil {
		return nil, err
	}
	var dto UserResponseDTO
	dto.FromModel(u)
	uc.enrichDTO(ctx, &dto)
	return &dto, nil
}

func (uc *usecase) GetByPhone(ctx context.Context, phone string) (*UserResponseDTO, error) {
	phone = utils.NormalizePhone(phone)
	u, err := uc.repo.FindByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}
	var dto UserResponseDTO
	dto.FromModel(u)
	uc.enrichDTO(ctx, &dto)
	return &dto, nil
}

func (uc *usecase) GetMe(ctx context.Context, id uuid.UUID) (*UserResponseDTO, error) {
	u, err := uc.repo.GetID(ctx, id)
	if err != nil {
		return nil, err
	}
	var dto UserResponseDTO
	dto.FromModel(u)
	uc.enrichDTO(ctx, &dto)
	return &dto, nil
}

// PatchSelf updates the caller's own profile. Role and IsActive are excluded
// from req — users cannot escalate their own privileges.
func (uc *usecase) PatchSelf(ctx context.Context, id uuid.UUID, req UserSelfUpdateDTO, photo *file.FileUpload) (*UserResponseDTO, error) {
	if id == uuid.Nil() {
		return nil, errors.ErrInvalidUUID
	}

	if req.Phone != nil {
		normalized := utils.NormalizePhone(*req.Phone)
		req.Phone = &normalized
	}

	u, err := uc.repo.GetID(ctx, id)
	if err != nil {
		return nil, err
	}

	req.ApplyToModel(u)

	if req.Password != nil && *req.Password != "" {
		// A valid access token alone must not allow a password change.
		if req.CurrentPassword == nil || u.CheckPassword(*req.CurrentPassword) != nil {
			return nil, errors.ErrInvalidCredentials
		}
		hashed, err := hash.HashPassword(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("error hashing password: %w", err)
		}
		u.Password = hashed
	}

	if photo != nil {
		filePath, err := uc.fileService.UploadFile(ctx, photo, "user")
		if err != nil {
			return nil, fmt.Errorf("error saving photo: %w", err)
		}
		u.Photo = filePath
	}

	u.UpdatedAt = time.Now()
	if err := uc.repo.Patch(ctx, u); err != nil {
		return nil, err
	}

	var dto UserResponseDTO
	dto.FromModel(u)
	uc.enrichDTO(ctx, &dto)
	return &dto, nil
}

// Patch is the admin-only update; it allows changing Role and IsActive.
func (uc *usecase) Patch(ctx context.Context, id uuid.UUID, req UserUpdateDTO, photo *file.FileUpload) (*UserResponseDTO, error) {
	if id == uuid.Nil() {
		return nil, errors.ErrInvalidUUID
	}

	if req.Phone != nil {
		normalized := utils.NormalizePhone(*req.Phone)
		req.Phone = &normalized
	}

	u, err := uc.repo.GetID(ctx, id)
	if err != nil {
		return nil, err
	}

	req.ApplyToModel(u)

	if req.Password != nil && *req.Password != "" {
		hashed, err := hash.HashPassword(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("error hashing password: %w", err)
		}
		u.Password = hashed
	}

	if photo != nil {
		filePath, err := uc.fileService.UploadFile(ctx, photo, "user")
		if err != nil {
			return nil, fmt.Errorf("error saving photo: %w", err)
		}
		u.Photo = filePath
	}

	u.UpdatedAt = time.Now()
	if err := uc.repo.Patch(ctx, u); err != nil {
		return nil, err
	}

	var dto UserResponseDTO
	dto.FromModel(u)
	uc.enrichDTO(ctx, &dto)
	return &dto, nil
}

func (uc *usecase) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil() {
		return errors.ErrInvalidUUID
	}
	return uc.repo.Delete(ctx, id)
}

func (uc *usecase) enrichDTO(ctx context.Context, dto *UserResponseDTO) {
	if dto.Photo == "" {
		return
	}
	url, err := uc.fileService.GetFullURL(ctx, dto.Photo)
	if err != nil {
		slog.Error("enrichDTO: failed to generate photo URL", "objectName", dto.Photo, "error", err)
		return
	}
	dto.Photo = url
}