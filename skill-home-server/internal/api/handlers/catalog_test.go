package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
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

	dsn := fmt.Sprintf("file:%s?cache=shared&_busy_timeout=5000&_journal_mode=WAL", filepath.Join(t.TempDir(), "catalog.db"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
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

func TestEnsureCatalogStateHandlesConcurrentFirstAccess(t *testing.T) {
	db := newCatalogTestDatabase(t)
	sqlDB, err := db.DB.DB()
	if err != nil {
		t.Fatalf("open sql db failed: %v", err)
	}
	sqlDB.SetMaxOpenConns(8)

	const workers = 6
	start := make(chan struct{})
	var ready sync.WaitGroup
	ready.Add(workers)
	var createMu sync.Mutex

	if err := db.Callback().Create().Before("gorm:create").Register("catalog_state_concurrency_barrier", func(tx *gorm.DB) {
		if tx.Statement.Table != "catalog_states" {
			return
		}

		ready.Done()
		ready.Wait()
		createMu.Lock()
	}); err != nil {
		t.Fatalf("register create callback failed: %v", err)
	}

	if err := db.Callback().Create().After("gorm:create").Register("catalog_state_concurrency_release", func(tx *gorm.DB) {
		if tx.Statement.Table != "catalog_states" {
			return
		}

		createMu.Unlock()
	}); err != nil {
		t.Fatalf("register create release callback failed: %v", err)
	}

	errCh := make(chan error, workers)
	var callWg sync.WaitGroup
	for i := 0; i < workers; i++ {
		callWg.Add(1)
		go func() {
			defer callWg.Done()
			<-start
			_, err := ensureCatalogState(db.DB)
			errCh <- err
		}()
	}

	close(start)
	ready.Wait()
	callWg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("ensureCatalogState returned error under concurrency: %v", err)
		}
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

func TestGetCatalogVersionDoesNotWriteWhenStateExists(t *testing.T) {
	t.Parallel()

	db := newCatalogTestDatabase(t)
	if err := db.Create(defaultCatalogState()).Error; err != nil {
		t.Fatalf("seed catalog state failed: %v", err)
	}

	wrote := false
	if err := db.Callback().Create().Before("gorm:create").Register("catalog_state_no_write_on_read", func(tx *gorm.DB) {
		if tx.Statement.Table == "catalog_states" {
			wrote = true
		}
	}); err != nil {
		t.Fatalf("register create callback failed: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/catalog/version", GetCatalogVersion(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/version", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if wrote {
		t.Fatal("expected catalog version lookup to stay read-only when state exists")
	}
}
