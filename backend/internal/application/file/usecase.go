package file

import (
	"context"
	"fmt"
	"gin_auth_service/internal/domain/file"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"uuid"
)

type usecase struct {
	repo        file.Repository
	bucketName  string
	bucketReady atomic.Bool
	bucketMu    sync.Mutex
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

// ensureBucket creates the bucket on first use and caches success only —
// a transient MinIO failure (e.g. a brief network blip at startup) must not
// permanently poison every future upload for the life of the process, so a
// failed attempt is retried on the next call instead of cached forever.
func (uc *usecase) ensureBucket(ctx context.Context) error {
	if uc.bucketReady.Load() {
		return nil
	}
	uc.bucketMu.Lock()
	defer uc.bucketMu.Unlock()
	if uc.bucketReady.Load() {
		return nil
	}

	exists, err := uc.repo.BucketExists(ctx, uc.bucketName)
	if err != nil {
		return fmt.Errorf("bucket check: %w", err)
	}
	if !exists {
		if err := uc.repo.CreateBucket(ctx, uc.bucketName); err != nil {
			return fmt.Errorf("bucket create: %w", err)
		}
	}
	uc.bucketReady.Store(true)
	return nil
}

func (uc *usecase) UploadFile(ctx context.Context, input *FileUpload, folder string) (string, error) {
	defer input.File.Close()

	if err := uc.ensureBucket(ctx); err != nil {
		return "", err
	}

	ext := filepath.Ext(input.Filename)
	objectName := fmt.Sprintf("%s/%s%s", folder, uuid.NewV7().String(), ext)

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
