package storage

import (
	"context"
	"fmt"
	"io"
	"slate-backend/pkg/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStore struct {
	client *minio.Client
	bucket string
}

func NewMinIOStore(cfg *config.Config) (*MinIOStore, error) {
	client, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds: credentials.NewStaticV4(
			cfg.MinIOAccessKey,
			cfg.MinIOSecretKey,
			"",
		),
	})
	if err != nil {
		return &MinIOStore{}, err
	}

	err = ensureBucket(client, cfg.MinIOBucket)
	if err != nil {
		return &MinIOStore{}, err
	}

	return &MinIOStore{
		client: client,
		bucket: cfg.MinIOBucket,
	}, err
}

func ensureBucket(client *minio.Client, bucketName string) error {
	exists, err := client.BucketExists(context.Background(), bucketName)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	if err := client.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{}); err != nil {
		existsCheck, checkErr := client.BucketExists(context.Background(), bucketName)
		if checkErr == nil && existsCheck {
			return nil
		}
	}

	return nil
}

func (s *MinIOStore) Upload(ctx context.Context, key string, reader io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{})
	return err
}

func (s *MinIOStore) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
}

func (s *MinIOStore) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)

		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}

		return false, fmt.Errorf("failed to check existence for %q: %w", key, err)
	}

	return true, nil
}
