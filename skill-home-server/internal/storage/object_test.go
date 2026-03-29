package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/skill-home/server/internal/config"
)

func TestObjectStoragePublicURL(t *testing.T) {
	t.Run("returns public url when configured", func(t *testing.T) {
		objStorage := newTestObjectStorage(t, config.StorageConfig{
			Type:          "local",
			LocalPath:     t.TempDir(),
			PublicBaseURL: "https://skills-static.example.com",
		})

		got, ok := objStorage.PublicURL("skills/team/reviewer/pkg.zip")
		if !ok {
			t.Fatalf("PublicURL returned ok=false, want true")
		}
		if want := "https://skills-static.example.com/skills/team/reviewer/pkg.zip"; got != want {
			t.Fatalf("PublicURL() = %q, want %q", got, want)
		}
	})

	t.Run("returns empty when public base url is missing", func(t *testing.T) {
		objStorage := newTestObjectStorage(t, config.StorageConfig{
			Type:      "local",
			LocalPath: t.TempDir(),
		})

		got, ok := objStorage.PublicURL("skills/team/reviewer/pkg.zip")
		if ok {
			t.Fatalf("PublicURL returned ok=true, want false")
		}
		if got != "" {
			t.Fatalf("PublicURL() = %q, want empty string", got)
		}
	})

	t.Run("normalizes surrounding slashes", func(t *testing.T) {
		objStorage := newTestObjectStorage(t, config.StorageConfig{
			Type:          "local",
			LocalPath:     t.TempDir(),
			PublicBaseURL: "https://skills-static.example.com/",
		})

		got, ok := objStorage.PublicURL("/skills/team/reviewer/pkg.zip")
		if !ok {
			t.Fatalf("PublicURL returned ok=false, want true")
		}
		if want := "https://skills-static.example.com/skills/team/reviewer/pkg.zip"; got != want {
			t.Fatalf("PublicURL() = %q, want %q", got, want)
		}
	})

	t.Run("preserves object root path prefix", func(t *testing.T) {
		objStorage := newTestObjectStorage(t, config.StorageConfig{
			Type:          "local",
			LocalPath:     t.TempDir(),
			PublicBaseURL: "https://cdn.example.com/skill-home-assets",
		})

		got, ok := objStorage.PublicURL("skills/team/reviewer/pkg.zip")
		if !ok {
			t.Fatalf("PublicURL returned ok=false, want true")
		}
		if want := "https://cdn.example.com/skill-home-assets/skills/team/reviewer/pkg.zip"; got != want {
			t.Fatalf("PublicURL() = %q, want %q", got, want)
		}
	})
}

func TestObjectStorageExists(t *testing.T) {
	ctx := context.Background()
	objStorage := newTestObjectStorage(t, config.StorageConfig{
		Type:      "local",
		LocalPath: t.TempDir(),
	})

	exists, err := objStorage.Exists(ctx, "skills/team/reviewer/pkg.zip")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Fatalf("Exists() = true, want false for missing object")
	}

	if err := objStorage.Upload(ctx, "skills/team/reviewer/pkg.zip", bytes.NewReader([]byte("payload")), int64(len("payload"))); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	exists, err = objStorage.Exists(ctx, "skills/team/reviewer/pkg.zip")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Fatalf("Exists() = false, want true for stored object")
	}
}

func TestIsObjectNotFoundError(t *testing.T) {
	t.Run("returns true for no such key", func(t *testing.T) {
		err := minio.ErrorResponse{Code: "NoSuchKey", StatusCode: 404}
		if !isObjectNotFoundError(err) {
			t.Fatal("isObjectNotFoundError() = false, want true")
		}
	})

	t.Run("returns true for wrapped no such key", func(t *testing.T) {
		err := fmt.Errorf("stat failed: %w", minio.ErrorResponse{Code: "NoSuchKey", StatusCode: 404})
		if !isObjectNotFoundError(err) {
			t.Fatal("isObjectNotFoundError() = false, want true")
		}
	})

	t.Run("returns false for no such bucket", func(t *testing.T) {
		err := minio.ErrorResponse{Code: "NoSuchBucket", StatusCode: 404}
		if isObjectNotFoundError(err) {
			t.Fatal("isObjectNotFoundError() = true, want false")
		}
	})

	t.Run("returns false for unrelated 404", func(t *testing.T) {
		err := minio.ErrorResponse{Code: "SignatureDoesNotMatch", StatusCode: 404}
		if isObjectNotFoundError(err) {
			t.Fatal("isObjectNotFoundError() = true, want false")
		}
	})
}

func TestObjectStorageCopyFrom(t *testing.T) {
	ctx := context.Background()
	source := newTestObjectStorage(t, config.StorageConfig{
		Type:      "local",
		LocalPath: t.TempDir(),
	})
	target := newTestObjectStorage(t, config.StorageConfig{
		Type:      "local",
		LocalPath: t.TempDir(),
	})

	if err := source.Upload(ctx, "skills/team/reviewer/pkg.zip", bytes.NewReader([]byte("archive-bytes")), int64(len("archive-bytes"))); err != nil {
		t.Fatalf("source Upload() error = %v", err)
	}

	if err := target.CopyFrom(ctx, "skills/team/reviewer/pkg.zip", source, "skills/team/reviewer/pkg.zip"); err != nil {
		t.Fatalf("CopyFrom() error = %v", err)
	}

	reader, err := target.Download(ctx, "skills/team/reviewer/pkg.zip")
	if err != nil {
		t.Fatalf("target Download() error = %v", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if got, want := string(content), "archive-bytes"; got != want {
		t.Fatalf("copied content = %q, want %q", got, want)
	}
}

func TestObjectStorageCopyFromMissingSource(t *testing.T) {
	ctx := context.Background()
	source := newTestObjectStorage(t, config.StorageConfig{
		Type:      "local",
		LocalPath: t.TempDir(),
	})
	target := newTestObjectStorage(t, config.StorageConfig{
		Type:      "local",
		LocalPath: t.TempDir(),
	})

	if err := target.CopyFrom(ctx, "skills/team/reviewer/pkg.zip", source, "skills/team/reviewer/pkg.zip"); err == nil {
		t.Fatal("CopyFrom() error = nil, want error for missing source object")
	}
}

func newTestObjectStorage(t *testing.T, cfg config.StorageConfig) *ObjectStorage {
	t.Helper()

	objStorage, err := NewObjectStorage(cfg)
	if err != nil {
		t.Fatalf("NewObjectStorage() error = %v", err)
	}
	return objStorage
}
