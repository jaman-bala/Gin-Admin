package user

import (
	"context"
	"fmt"
	"gin_auth_service/internal/application/file"
	"gin_auth_service/internal/domain/user"
	"gin_auth_service/internal/pkg/hash"
	"gin_auth_service/pkg/errors"
	"time"

	"github.com/google/uuid"
)


type usecase struct {
	repo        user.Repository
	fileService file.UseCase
}

// NewUseCase creates a new instance of UseCase.
func NewUseCase(repo user.Repository, fileService file.UseCase) UseCase {
	return &usecase{
		repo:        repo,
		fileService: fileService,
	}
}

func (uc *usecase) GetAll(ctx context.Context, page, limit int, search string, isActive *bool) (*UserListResponse, error) {
	params := user.FilterParams{
		Page:     page,
		Limit:    limit,
		Search:   search,
		IsActive: isActive,
	}

	users, total, err := uc.repo.GetWithFilters(ctx, params)
	if err != nil {
		return nil, errors.ErrUserNotFound
	}

	var dtos []*UserResponseDTO
	for _, u := range users {
		var dto UserResponseDTO
		dto.FromModel(u)
		uc.enrichDTO(ctx, &dto)
		dtos = append(dtos, &dto)
	}

	if dtos == nil {
		dtos = []*UserResponseDTO{}
	}

	return &UserListResponse{
		Users: dtos,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (uc *usecase) Create(ctx context.Context, req UserRequestDTO, photoFile *FileUpload) (*UserResponseDTO, error) {
	existing, _ := uc.repo.FindByPhone(ctx, req.Phone)
	if existing != nil {
		return nil, errors.ErrConflict
	}
	hashedPassword, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	u := &user.User{
		ID:         uuid.New(),
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

	if photoFile != nil {
		filePath, err := uc.saveUserPhoto(ctx, photoFile)
		if err != nil {
			return nil, err
		}
		u.Photo = filePath
	}

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
		return nil, errors.ErrUserNotFound
	}

	var dto UserResponseDTO
	dto.FromModel(u)
	uc.enrichDTO(ctx, &dto)
	return &dto, nil
}

func (uc *usecase) GetByPhone(ctx context.Context, phone string) (*UserResponseDTO, error) {
	u, err := uc.repo.FindByPhone(ctx, phone)
	if err != nil {
		return nil, errors.ErrUserNotFound
	}

	var dto UserResponseDTO
	dto.FromModel(u)
	uc.enrichDTO(ctx, &dto)
	return &dto, nil
}

func (uc *usecase) Patch(ctx context.Context, id uuid.UUID, request UserUpdateDTO, photoFile *FileUpload) (*UserResponseDTO, error) {
	if id == uuid.Nil {
		return nil, errors.ErrInvalidUUID
	}

	u, err := uc.repo.GetID(ctx, id)
	if err != nil {
		return nil, errors.ErrUserNotFound
	}

	request.ApplyToModel(u)

	if request.Password != nil && *request.Password != "" {
		hashed, err := hash.HashPassword(*request.Password)
		if err != nil {
			return nil, fmt.Errorf("error hashing password: %w", err)
		}
		u.Password = hashed
	}

	if photoFile != nil {
		filePath, err := uc.saveUserPhoto(ctx, photoFile)
		if err != nil {
			return nil, err
		}
		u.Photo = filePath
	}

	u.UpdatedAt = time.Now()
	if err := uc.repo.Patch(ctx, u); err != nil {
		return nil, errors.ErrUpdateConflict
	}

	var dto UserResponseDTO
	dto.FromModel(u)
	uc.enrichDTO(ctx, &dto)
	return &dto, nil
}

func (uc *usecase) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.ErrInvalidUUID
	}
	return uc.repo.Delete(ctx, id)
}

func (uc *usecase) saveUserPhoto(ctx context.Context, photoFile *FileUpload) (string, error) {
	if photoFile == nil {
		return "", nil
	}

	// Converting to file.FileUpload which we'll create next
	fileDTO := &file.FileUpload{
		Filename:    photoFile.Filename,
		Size:        photoFile.Size,
		ContentType: photoFile.ContentType,
		File:        photoFile.File,
	}

	filePath, err := uc.fileService.UploadFile(ctx, fileDTO, "user")
	if err != nil {
		return "", fmt.Errorf("error saving photo: %w", err)
	}

	return filePath, nil
}

func (uc *usecase) enrichDTO(ctx context.Context, dto *UserResponseDTO) {
	if dto.Photo != "" {
		url, err := uc.fileService.GetFullURL(ctx, dto.Photo)
		if err == nil {
			dto.Photo = url
		}
	}
}
