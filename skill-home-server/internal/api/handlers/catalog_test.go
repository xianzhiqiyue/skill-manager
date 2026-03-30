package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newCatalogTestDatabase(t *testing.T) *storage.Database {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	if err := db.AutoMigrate(&models.CatalogState{}); err != nil {
		t.Fatalf("auto migrate catalog state failed: %v", err)
	}

	return &storage.Database{DB: db}
}

func TestEnsureCatalogStateCreatesDefaultRow(t *testing.T) {
	t.Parallel()

	db := newCatalogTestDatabase(t)

	var state *models.CatalogState
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		state, err = ensureCatalogState(tx)
		return err
	})
	if err != nil {
		t.Fatalf("ensureCatalogState returned error: %v", err)
	}

	if state.CatalogVersion != 1 {
		t.Fatalf("CatalogVersion = %d, want 1", state.CatalogVersion)
	}
	if state.UpdatedAt.IsZero() {
		t.Fatal("UpdatedAt is zero")
	}

	var count int64
	if err := db.Model(&models.CatalogState{}).Count(&count).Error; err != nil {
		t.Fatalf("count catalog states failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("catalog state count = %d, want 1", count)
	}
}

func TestGetCatalogVersion(t *testing.T) {
	t.Parallel()

	db := newCatalogTestDatabase(t)
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/v1/catalog/version", GetCatalogVersion(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/version", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}

	if got := payload["catalog_version"]; got != float64(1) {
		t.Fatalf("catalog_version = %v, want 1", got)
	}

	updatedAt, ok := payload["updated_at"].(string)
	if !ok {
		t.Fatalf("updated_at = %T, want string", payload["updated_at"])
	}
	if updatedAt == "" {
		t.Fatal("updated_at is empty")
	}
	if _, err := time.Parse(time.RFC3339Nano, updatedAt); err != nil {
		t.Fatalf("updated_at is not RFC3339 time: %v", err)
	}
}
