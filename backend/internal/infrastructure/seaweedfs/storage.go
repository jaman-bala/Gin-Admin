// Package seaweedfs implements file storage on top of SeaweedFS through its
// S3-compatible gateway (`weed s3` / `weed server -s3`, port 8333 by default).
package seaweedfs

import (
	"context"
	"errors"
	"fmt"
	"gin_auth_service/internal/domain/file"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// region is only used for SigV4 signing; SeaweedFS does not care about its
// value, but client and server must agree on it.
const region = "us-east-1"

// maxPresignExpiry is the SigV4 hard limit for presigned URLs.
const maxPresignExpiry = 7 * 24 * time.Hour

// Storage provides an implementation of file storage using SeaweedFS S3 API.
type Storage struct {
	client    *s3.Client
	presigner *s3.PresignClient // signs URLs for the public (browser-visible) host
}

// NewStorage creates a new SeaweedFS storage. endpoint is host:port reachable
// from the backend (e.g. "seaweedfs:8333" inside Docker); publicEndpoint is
// host:port the browser can reach and is used only for presigned URLs.
func NewStorage(endpoint, accessKey, secretKey string, useSSL bool, publicEndpoint string) (*Storage, error) {
	if endpoint == "" {
		return nil, errors.New("seaweedfs: endpoint is required")
	}
	if publicEndpoint == "" {
		publicEndpoint = "localhost:8333"
	}

	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")

	newClient := func(host string) *s3.Client {
		return s3.New(s3.Options{
			Region:       region,
			Credentials:  creds,
			BaseEndpoint: aws.String(scheme + "://" + host),
			// Path-style URLs (host/bucket/key) avoid virtual-hosted DNS.
			UsePathStyle: true,
			// Newer SDK versions add CRC checksums to every request by
			// default; only send them when an operation requires it, for
			// compatibility with S3-compatible servers.
			RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
			ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
			AppID:                      "gin-auth-service",
		})
	}

	return &Storage{
		client:    newClient(endpoint),
		presigner: s3.NewPresignClient(newClient(publicEndpoint)),
	}, nil
}

func (s *Storage) UploadFile(ctx context.Context, bucket, objectName string, f io.Reader, fileSize int64, contentType string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(objectName),
		Body:          f,
		ContentLength: aws.Int64(fileSize),
		ContentType:   aws.String(contentType),
	}

	// SigV4 over plain HTTP must hash the body, which requires rewinding it.
	// For non-seekable readers send UNSIGNED-PAYLOAD instead of buffering the
	// whole file in memory (the headers are still signed).
	var opts []func(*s3.Options)
	if _, seekable := f.(io.Seeker); !seekable {
		opts = append(opts, s3.WithAPIOptions(v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware))
	}

	if _, err := s.client.PutObject(ctx, input, opts...); err != nil {
		return "", fmt.Errorf("seaweedfs: put %s/%s: %w", bucket, objectName, err)
	}
	return objectName, nil
}

func (s *Storage) DeleteFile(ctx context.Context, bucket, objectName string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectName),
	})
	if err != nil {
		return fmt.Errorf("seaweedfs: delete %s/%s: %w", bucket, objectName, err)
	}
	return nil
}

func (s *Storage) GetFileURL(ctx context.Context, bucket, objectName string, expiry time.Duration) (string, error) {
	if expiry <= 0 {
		expiry = 24 * time.Hour
	}
	expiry = min(expiry, maxPresignExpiry)

	req, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectName),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("seaweedfs: presign %s/%s: %w", bucket, objectName, err)
	}
	return req.URL, nil
}

func (s *Storage) BucketExists(ctx context.Context, bucket string) (bool, error) {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	if err == nil {
		return true, nil
	}
	var notFound *types.NotFound
	if errors.As(err, &notFound) || hasErrorCode(err, "NotFound", "NoSuchBucket") {
		return false, nil
	}
	return false, fmt.Errorf("seaweedfs: head bucket %s: %w", bucket, err)
}

// Ping checks SeaweedFS connectivity and credentials by listing buckets.
func (s *Storage) Ping(ctx context.Context) error {
	_, err := s.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	return err
}

// CreateBucket creates the bucket. Losing a creation race against another
// replica is not an error — the bucket exists either way.
func (s *Storage) CreateBucket(ctx context.Context, bucket string) error {
	_, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	if err == nil || hasErrorCode(err, "BucketAlreadyOwnedByYou", "BucketAlreadyExists") {
		return nil
	}
	return fmt.Errorf("seaweedfs: create bucket %s: %w", bucket, err)
}

func hasErrorCode(err error, codes ...string) bool {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	for _, c := range codes {
		if apiErr.ErrorCode() == c {
			return true
		}
	}
	return false
}

// Ensure Storage implements file.Repository
var _ file.Repository = (*Storage)(nil)
