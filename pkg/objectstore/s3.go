package objectstore

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config holds the configuration for an S3-compatible object store.
type S3Config struct {
	Endpoint  string
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

// S3ObjectStore stores objects in an S3-compatible service (AWS S3, MinIO, LocalStack).
type S3ObjectStore struct {
	client *minio.Client
	bucket string
}

// NewS3ObjectStore creates an S3-backed object store. It ensures the bucket
// exists, creating it if necessary.
func NewS3ObjectStore(ctx context.Context, cfg S3Config) (*S3ObjectStore, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("objectstore/s3: create client: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("objectstore/s3: check bucket %q: %w", cfg.Bucket, err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
			return nil, fmt.Errorf("objectstore/s3: create bucket %q: %w", cfg.Bucket, err)
		}
	}

	return &S3ObjectStore{client: client, bucket: cfg.Bucket}, nil
}

// Put uploads data to the S3 bucket under the given key.
func (s *S3ObjectStore) Put(ctx context.Context, key string, data []byte) error {
	reader := bytes.NewReader(data)
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("objectstore/s3: put %q: %w", key, err)
	}
	return nil
}

// Get retrieves data from the S3 bucket by key.
func (s *S3ObjectStore) Get(ctx context.Context, key string) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("objectstore/s3: get %q: %w", key, err)
	}
	defer func() { _ = obj.Close() }()

	data, err := io.ReadAll(obj)
	if err != nil {
		// MinIO client returns an error on Read when the object doesn't exist.
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("objectstore/s3: read %q: %w", key, err)
	}
	return data, nil
}

// Delete removes an object from the S3 bucket by key.
func (s *S3ObjectStore) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("objectstore/s3: delete %q: %w", key, err)
	}
	return nil
}

// Exists checks whether an object exists in the S3 bucket.
func (s *S3ObjectStore) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("objectstore/s3: stat %q: %w", key, err)
	}
	return true, nil
}
