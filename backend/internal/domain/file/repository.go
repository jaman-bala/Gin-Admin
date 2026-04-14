package file

import (
	"context"
	"io"
	"time"
)

// Repository defines the interface for file storage.
type Repository interface {
	UploadFile(ctx context.Context, bucket, objectName string, file io.Reader, fileSize int64, contentType string) (string, error)
	DeleteFile(ctx context.Context, bucket, objectName string) error
	GetFileURL(ctx context.Context, bucket, objectName string, expiry time.Duration) (string, error)
	BucketExists(ctx context.Context, bucket string) (bool, error)
	CreateBucket(ctx context.Context, bucket string) error
}
