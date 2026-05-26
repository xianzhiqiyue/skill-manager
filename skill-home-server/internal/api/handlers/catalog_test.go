package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	if err := db.Exec(`CREATE TABLE skills (
		id TEXT PRIMARY KEY,
		namespace TEXT NOT NULL,
		name TEXT NOT NULL,
		owner_id TEXT NOT NULL,
		description TEXT,
		category TEXT,
		description_zh TEXT,
		author TEXT,
		tags TEXT,
		license TEXT,
		homepage TEXT,
		repository TEXT,
		download_count INTEGER DEFAULT 0,
		like_count INTEGER DEFAULT 0,
		install_count INTEGER DEFAULT 0,
		rating_sum INTEGER DEFAULT 0,
		rating_count INTEGER DEFAULT 0,
		is_public NUMERIC DEFAULT 1,
		is_owner_only NUMERIC DEFAULT 0,
		is_deprecated NUMERIC DEFAULT 0,
		is_recommended NUMERIC DEFAULT 0,
		latest_version TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create skills table failed: %v", err)
	}
	if err := db.Exec(`CREATE TABLE skill_versions (
		id TEXT PRIMARY KEY,
		skill_id TEXT NOT NULL,
		version TEXT NOT NULL,
		manifest TEXT,
		dependencies TEXT,
		storage_path TEXT,
		size_bytes INTEGER,
		checksum TEXT,
		scan_status TEXT,
		scan_result TEXT,
		published_by TEXT NOT NULL,
		published_at DATETIME,
		created_at DATETIME,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create skill_versions table failed: %v", err)
	}
	if err := db.Exec(`CREATE TABLE users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL,
		display_name_zh TEXT,
		email TEXT NOT NULL,
		password TEXT,
		avatar_url TEXT,
		is_active NUMERIC DEFAULT 1,
		is_admin NUMERIC DEFAULT 0,
		is_super_admin NUMERIC DEFAULT 0,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create users table failed: %v", err)
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

func TestEnsureCatalogStateRetriesOnTransientLock(t *testing.T) {
	db := newCatalogTestDatabase(t)
	sqlDB, err := db.DB.DB()
	if err != nil {
		t.Fatalf("open sql db failed: %v", err)
	}
	sqlDB.SetMaxOpenConns(2)

	locked := make(chan struct{})
	release := make(chan struct{})
	lockErr := make(chan error, 1)

	go func() {
		tx := db.DB.Begin()
		if tx.Error != nil {
			lockErr <- tx.Error
			return
		}
		if err := tx.Exec(`INSERT INTO catalog_states (id, catalog_version, created_at, updated_at) VALUES (1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`).Error; err != nil {
			lockErr <- err
			_ = tx.Rollback().Error
			return
		}
		close(locked)
		<-release
		lockErr <- tx.Rollback().Error
	}()

	<-locked

	done := make(chan struct{})
	var state *models.CatalogState
	var ensureErr error
	go func() {
		state, ensureErr = ensureCatalogState(db.DB)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	close(release)
	<-done

	if err := <-lockErr; err != nil && !errors.Is(err, gorm.ErrInvalidTransaction) {
		t.Fatalf("locker returned error: %v", err)
	}
	if ensureErr != nil {
		t.Fatalf("ensureCatalogState returned error under transient lock: %v", ensureErr)
	}
	if state == nil {
		t.Fatal("ensureCatalogState returned nil state")
	}
	if state.CatalogVersion != 1 {
		t.Fatalf("CatalogVersion = %d, want 1", state.CatalogVersion)
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

func TestGetCatalogVersionReconcilesRecommendationMutation(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	skill := &models.Skill{
		ID:            uuid.New(),
		Namespace:     "team",
		Name:          "reviewer",
		OwnerID:       user.ID,
		Description:   "review skill",
		IsPublic:      true,
		IsRecommended: false,
	}
	if err := db.Create(skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}
	if err := db.Model(&models.CatalogState{}).Where("id = ?", catalogStateSingletonID).Update("updated_at", time.Now()).Error; err != nil {
		t.Fatalf("sync catalog state failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	if err := db.Model(skill).Update("is_recommended", true).Error; err != nil {
		t.Fatalf("update recommendation failed: %v", err)
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
	if got := currentCatalogVersion(t, db); got != 2 {
		t.Fatalf("catalog_version = %d, want 2", got)
	}
}

func TestGetCatalogVersionReconcilesPublishedVersionMutation(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	skill := &models.Skill{
		ID:            uuid.New(),
		Namespace:     "team",
		Name:          "builder",
		OwnerID:       user.ID,
		Description:   "build skill",
		IsPublic:      true,
		LatestVersion: "1.0.0",
	}
	if err := db.Create(skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}
	if err := db.Model(&models.CatalogState{}).Where("id = ?", catalogStateSingletonID).Update("updated_at", time.Now()).Error; err != nil {
		t.Fatalf("sync catalog state failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	version := &models.SkillVersion{
		ID:          uuid.New(),
		SkillID:     skill.ID,
		Version:     "1.1.0",
		StoragePath: "skills/team/builder/1.1.0.zip",
		PublishedBy: user.ID,
		PublishedAt: time.Now(),
		CreatedAt:   time.Now(),
	}
	if err := db.Create(version).Error; err != nil {
		t.Fatalf("create version failed: %v", err)
	}
	if err := db.Model(skill).Update("latest_version", version.Version).Error; err != nil {
		t.Fatalf("update latest_version failed: %v", err)
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
	if got := currentCatalogVersion(t, db); got != 2 {
		t.Fatalf("catalog_version = %d, want 2", got)
	}
}
