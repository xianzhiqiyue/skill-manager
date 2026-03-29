package storage

import (
	"testing"

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

func newTestObjectStorage(t *testing.T, cfg config.StorageConfig) *ObjectStorage {
	t.Helper()

	objStorage, err := NewObjectStorage(cfg)
	if err != nil {
		t.Fatalf("NewObjectStorage() error = %v", err)
	}
	return objStorage
}
