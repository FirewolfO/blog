package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Object struct {
	Body        io.ReadCloser
	Size        int64
	ContentType string
}

type Store interface {
	EnsureBucket(context.Context) error
	Put(context.Context, string, io.Reader, int64, string) error
	Get(context.Context, string) (*Object, error)
}

type S3 struct {
	client *minio.Client
	bucket string
}

func NewS3(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*S3, error) {
	client, err := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(accessKey, secretKey, ""), Secure: useSSL, Region: "garage"})
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}
	return &S3{client: client, bucket: bucket}, nil
}

func (s *S3) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check storage bucket: %w", err)
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: "garage"}); err != nil {
		return fmt.Errorf("create storage bucket: %w", err)
	}
	return nil
}

func (s *S3) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("store object: %w", err)
	}
	return nil
}

func (s *S3) Get(ctx context.Context, key string) (*Object, error) {
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	stat, err := object.Stat()
	if err != nil {
		_ = object.Close()
		return nil, fmt.Errorf("stat object: %w", err)
	}
	return &Object{Body: object, Size: stat.Size, ContentType: stat.ContentType}, nil
}
