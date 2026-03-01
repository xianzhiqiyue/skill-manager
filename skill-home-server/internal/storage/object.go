package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/skill-home/server/internal/config"
)

// ObjectStorage 对象存储
type ObjectStorage struct {
	client *minio.Client
	bucket string
	type_  string
	localPath string
}

// NewObjectStorage 创建对象存储连接
func NewObjectStorage(cfg config.StorageConfig) (*ObjectStorage, error) {
	switch cfg.Type {
	case "minio", "s3":
		client, err := minio.New(cfg.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
			Secure: cfg.UseSSL,
		})
		if err != nil {
			return nil, err
		}
		return &ObjectStorage{client: client, bucket: cfg.Bucket, type_: cfg.Type}, nil

	case "local":
		path := cfg.LocalPath
		if path == "" {
			path = "./data/storage"
		}
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, err
		}
		return &ObjectStorage{type_: "local", localPath: path}, nil

	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Type)
	}
}

// Upload 上传文件
func (s *ObjectStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64) error {
	if s.type_ == "local" {
		return s.uploadLocal(key, reader)
	}
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{})
	return err
}

// Download 下载文件
func (s *ObjectStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	if s.type_ == "local" {
		return s.downloadLocal(key)
	}
	return s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
}

// Delete 删除文件
func (s *ObjectStorage) Delete(ctx context.Context, key string) error {
	if s.type_ == "local" {
		return s.deleteLocal(key)
	}
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

// 本地存储实现
func (s *ObjectStorage) uploadLocal(key string, reader io.Reader) error {
	path := filepath.Join(s.localPath, key)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, reader)
	return err
}

func (s *ObjectStorage) downloadLocal(key string) (io.ReadCloser, error) {
	path := filepath.Join(s.localPath, key)
	return os.Open(path)
}

func (s *ObjectStorage) deleteLocal(key string) error {
	path := filepath.Join(s.localPath, key)
	return os.Remove(path)
}
