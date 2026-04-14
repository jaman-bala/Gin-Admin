package file

import (
	"context"
	"fmt"
	"gin_auth_service/internal/domain/file"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// usecase provides high-level operations for file management.
type usecase struct {
	repo       file.Repository
	bucketName string
}

// NewUseCase creates a new instance of UseCase.
func NewUseCase(repo file.Repository, bucketName string) UseCase {
	if bucketName == "" {
		bucketName = "uploads"
	}
	return &usecase{
		repo:       repo,
		bucketName: bucketName,
	}
}

// UploadFile uploads a file to the storage and returns the URL.
func (uc *usecase) UploadFile(ctx context.Context, input *FileUpload, folder string) (string, error) {
	// Создаем уникальное имя файла
	ext := filepath.Ext(input.Filename)
	objectName := fmt.Sprintf("%s/%s%s", folder, uuid.New().String(), ext)

	// Используем сконфигурированный бакет
	bucket := uc.bucketName

	// Проверяем существование бакета (можно вынести в инициализацию)
	exists, err := uc.repo.BucketExists(ctx, bucket)
	if err != nil {
		return "", err
	}
	if !exists {
		if err := uc.repo.CreateBucket(ctx, bucket); err != nil {
			return "", err
		}
	}

	url, err := uc.repo.UploadFile(ctx, bucket, objectName, input.File, input.Size, input.ContentType)
	if err != nil {
		return "", err
	}

	return url, nil
}

// DeleteFile removes a file from storage.
func (uc *usecase) DeleteFile(ctx context.Context, bucket, objectName string) error {
	return uc.repo.DeleteFile(ctx, bucket, objectName)
}

// GetFileURL generates a URL for a file.
func (uc *usecase) GetFileURL(ctx context.Context, bucket, objectName string, expiry time.Duration) (string, error) {
	return uc.repo.GetFileURL(ctx, bucket, objectName, expiry)
}

// GetFullURL generates a signed URL for a file path stored in the DB.
func (uc *usecase) GetFullURL(ctx context.Context, objectName string) (string, error) {
	if objectName == "" {
		return "", nil
	}
	// Using the configured bucket
	return uc.repo.GetFileURL(ctx, uc.bucketName, objectName, time.Hour*24)
}
