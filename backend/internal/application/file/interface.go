package file

import (
	"context"
	"time"
)

// UseCase provides high-level operations for file management.
type UseCase interface {
	UploadFile(ctx context.Context, input *FileUpload, folder string) (string, error)
	DeleteFile(ctx context.Context, bucket, objectName string) error
	GetFileURL(ctx context.Context, bucket, objectName string, expiry time.Duration) (string, error)
	GetFullURL(ctx context.Context, objectName string) (string, error)
}
