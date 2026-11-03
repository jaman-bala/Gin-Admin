package seaweedfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Integration tests against a real SeaweedFS S3 gateway. Run with e.g.:
//
//	SEAWEEDFS_TEST_ENDPOINT=localhost:8333 \
//	SEAWEEDFS_TEST_ACCESS_KEY=... SEAWEEDFS_TEST_SECRET_KEY=... \
//	go test ./internal/infrastructure/seaweedfs/...
func newTestStorage(t *testing.T, secretKey string) *Storage {
	t.Helper()
	endpoint := os.Getenv("SEAWEEDFS_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("SEAWEEDFS_TEST_ENDPOINT not set; skipping SeaweedFS integration test")
	}
	if secretKey == "" {
		secretKey = os.Getenv("SEAWEEDFS_TEST_SECRET_KEY")
	}
	s, err := NewStorage(endpoint, os.Getenv("SEAWEEDFS_TEST_ACCESS_KEY"), secretKey, false, endpoint)
	if err != nil {
		t.Fatalf("NewStorage: %v", err)
	}
	return s
}

func TestStorageRoundTrip(t *testing.T) {
	s := newTestStorage(t, "")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bucket := fmt.Sprintf("it-%d", time.Now().UnixNano())

	exists, err := s.BucketExists(ctx, bucket)
	if err != nil || exists {
		t.Fatalf("BucketExists before create = %v, %v; want false, nil", exists, err)
	}
	if err := s.CreateBucket(ctx, bucket); err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}
	t.Cleanup(func() {
		_, _ = s.client.DeleteBucket(context.Background(), &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	})
	if err := s.CreateBucket(ctx, bucket); err != nil {
		t.Fatalf("CreateBucket on existing bucket must be idempotent: %v", err)
	}
	if exists, err := s.BucketExists(ctx, bucket); err != nil || !exists {
		t.Fatalf("BucketExists after create = %v, %v; want true, nil", exists, err)
	}

	cases := map[string]io.Reader{
		"seekable":     bytes.NewReader([]byte("seekable body")),
		"non-seekable": io.NopCloser(strings.NewReader("non-seekable body")),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			want := name + " body"
			key := "user/" + name + ".txt"
			if _, err := s.UploadFile(ctx, bucket, key, body, int64(len(want)), "text/plain"); err != nil {
				t.Fatalf("UploadFile: %v", err)
			}

			url, err := s.GetFileURL(ctx, bucket, key, time.Minute)
			if err != nil {
				t.Fatalf("GetFileURL: %v", err)
			}
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("GET presigned URL: %v", err)
			}
			got, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK || string(got) != want {
				t.Fatalf("presigned GET = %d %q; want 200 %q", resp.StatusCode, got, want)
			}
			if ct := resp.Header.Get("Content-Type"); ct != "text/plain" {
				t.Errorf("Content-Type = %q; want text/plain", ct)
			}

			if err := s.DeleteFile(ctx, bucket, key); err != nil {
				t.Fatalf("DeleteFile: %v", err)
			}
			resp, err = http.Get(url)
			if err != nil {
				t.Fatalf("GET after delete: %v", err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("GET after delete = %d; want 404", resp.StatusCode)
			}
		})
	}

	// Tampered presigned URL must be rejected.
	if _, err := s.UploadFile(ctx, bucket, "user/secret.txt", bytes.NewReader([]byte("x")), 1, "text/plain"); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	url, _ := s.GetFileURL(ctx, bucket, "user/secret.txt", time.Minute)
	resp, err := http.Get(strings.Replace(url, "secret.txt", "other.txt", 1))
	if err != nil {
		t.Fatalf("GET tampered URL: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("tampered presigned URL = %d; want 403", resp.StatusCode)
	}
	_ = s.DeleteFile(ctx, bucket, "user/secret.txt")
}

func TestStoragePingRejectsWrongCredentials(t *testing.T) {
	s := newTestStorage(t, "definitely-wrong-secret")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.Ping(ctx); err == nil {
		t.Fatal("Ping with wrong secret succeeded; want auth error")
	}
}

func TestStoragePingOK(t *testing.T) {
	s := newTestStorage(t, "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestNewStorageRequiresEndpoint(t *testing.T) {
	if _, err := NewStorage("", "k", "s", false, ""); err == nil {
		t.Fatal("NewStorage with empty endpoint succeeded; want error")
	}
}
