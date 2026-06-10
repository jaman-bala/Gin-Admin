package minio

import (
	"context"
	"gin_auth_service/internal/domain/file"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Storage provides an implementation of file storage using MinIO.
type Storage struct {
	client         *minio.Client
	publicClient   *minio.Client // Client configured with public endpoint for URL generation
	publicEndpoint string
	useSSL         bool
}

// NewStorage creates a new instance of MinIO storage.
func NewStorage(endpoint, accessKey, secretKey string, useSSL bool, publicEndpoint string) (*Storage, error) {
	// Internal client for operations (uses docker network hostname).
	// BucketLookupPath forces path-style URLs to avoid virtual-hosted DNS issues.
	// Region "us-east-1" skips the GetBucketLocation network round-trip.
	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:       useSSL,
		BucketLookup: minio.BucketLookupPath,
		Region:       "us-east-1",
	})
	if err != nil {
		return nil, err
	}
	client.SetAppInfo("gin-auth-service", "1.0")

	// Default to localhost:9000 if not specified
	if publicEndpoint == "" {
		publicEndpoint = "localhost:9000"
	}

	// Public client for presigned URL generation.
	// Uses publicEndpoint so generated URLs are browser-accessible.
	// Region skips the GetBucketLocation network call — from inside Docker,
	// "localhost" resolves to the container, not to the MinIO service.
	publicClient, err := minio.New(publicEndpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:       useSSL,
		BucketLookup: minio.BucketLookupPath,
		Region:       "us-east-1",
	})
	if err != nil {
		return nil, err
	}
	publicClient.SetAppInfo("gin-auth-service", "1.0")

	return &Storage{
		client:         client,
		publicClient:   publicClient,
		publicEndpoint: publicEndpoint,
		useSSL:         useSSL,
	}, nil
}

func (m *Storage) UploadFile(ctx context.Context, bucket, objectName string, f io.Reader, fileSize int64, contentType string) (string, error) {
	_, err := m.client.PutObject(ctx, bucket, objectName, f, fileSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}

	return objectName, nil
}

func (m *Storage) DeleteFile(ctx context.Context, bucket, objectName string) error {
	return m.client.RemoveObject(ctx, bucket, objectName, minio.RemoveObjectOptions{})
}

func (m *Storage) GetFileURL(ctx context.Context, bucket, objectName string, expiry time.Duration) (string, error) {
	if expiry == 0 {
		expiry = time.Hour * 24
	}
	url, err := m.publicClient.PresignedGetObject(ctx, bucket, objectName, expiry, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func (m *Storage) BucketExists(ctx context.Context, bucket string) (bool, error) {
	return m.client.BucketExists(ctx, bucket)
}

// Ping checks MinIO connectivity by listing buckets.
func (m *Storage) Ping(ctx context.Context) error {
	_, err := m.client.ListBuckets(ctx)
	return err
}

func (m *Storage) CreateBucket(ctx context.Context, bucket string) error {
	return m.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}

// Ensure Storage implements file.Repository
var _ file.Repository = (*Storage)(nil)
