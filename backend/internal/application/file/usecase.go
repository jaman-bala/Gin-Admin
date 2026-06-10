package file

import (
	"context"
	"fmt"
	"gin_auth_service/internal/domain/file"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type usecase struct {
	repo           file.Repository
	bucketName     string
	initBucketOnce sync.Once
	initBucketErr  error
}

func NewUseCase(repo file.Repository, bucketName string) UseCase {
	if bucketName == "" {
		bucketName = "uploads"
	}
	return &usecase{
		repo:       repo,
		bucketName: bucketName,
	}
}

// ensureBucket creates the bucket on the first upload. If the first attempt
// fails the Once is spent and subsequent calls return the cached error; the
// container should be restarted in that case.
func (uc *usecase) ensureBucket(ctx context.Context) error {
	uc.initBucketOnce.Do(func() {
		exists, err := uc.repo.BucketExists(ctx, uc.bucketName)
		if err != nil {
			uc.initBucketErr = fmt.Errorf("bucket check: %w", err)
			return
		}
		if !exists {
			if err := uc.repo.CreateBucket(ctx, uc.bucketName); err != nil {
				uc.initBucketErr = fmt.Errorf("bucket create: %w", err)
			}
		}
	})
	return uc.initBucketErr
}

func (uc *usecase) UploadFile(ctx context.Context, input *FileUpload, folder string) (string, error) {
	defer input.File.Close()

	if err := uc.ensureBucket(ctx); err != nil {
		return "", err
	}

	ext := filepath.Ext(input.Filename)
	objectName := fmt.Sprintf("%s/%s%s", folder, uuid.Must(uuid.NewV7()).String(), ext)

	if _, err := uc.repo.UploadFile(ctx, uc.bucketName, objectName, input.File, input.Size, input.ContentType); err != nil {
		return "", err
	}
	return objectName, nil
}

func (uc *usecase) DeleteFile(ctx context.Context, bucket, objectName string) error {
	return uc.repo.DeleteFile(ctx, bucket, objectName)
}

func (uc *usecase) GetFileURL(ctx context.Context, bucket, objectName string, expiry time.Duration) (string, error) {
	return uc.repo.GetFileURL(ctx, bucket, objectName, expiry)
}

func (uc *usecase) GetFullURL(ctx context.Context, objectName string) (string, error) {
	if objectName == "" {
		return "", nil
	}
	return uc.repo.GetFileURL(ctx, uc.bucketName, objectName, time.Hour*24)
}