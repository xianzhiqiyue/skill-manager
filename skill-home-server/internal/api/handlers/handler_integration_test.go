package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skill-home/server/internal/api/middleware"
	"github.com/skill-home/server/internal/config"
	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"github.com/skill-home/server/pkg/validator"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDatabase(t *testing.T) *storage.Database {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	database := &storage.Database{DB: db}
	mustExec(t, db, `CREATE TABLE users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL,
		email TEXT NOT NULL,
		password TEXT,
		avatar_url TEXT,
		is_active NUMERIC DEFAULT 1,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	mustExec(t, db, `CREATE TABLE skills (
		id TEXT PRIMARY KEY,
		namespace TEXT NOT NULL,
		name TEXT NOT NULL,
		owner_id TEXT NOT NULL,
		description TEXT,
		description_zh TEXT,
		author TEXT,
		tags TEXT,
		license TEXT,
		homepage TEXT,
		repository TEXT,
		download_count INTEGER DEFAULT 0,
		rating_sum INTEGER DEFAULT 0,
		rating_count INTEGER DEFAULT 0,
		is_public NUMERIC DEFAULT 1,
		is_deprecated NUMERIC DEFAULT 0,
		latest_version TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	mustExec(t, db, `CREATE TABLE catalog_states (
		id INTEGER PRIMARY KEY,
		catalog_version INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	if err := db.Create(&models.CatalogState{}).Error; err != nil {
		t.Fatalf("seed catalog state failed: %v", err)
	}
	mustExec(t, db, `CREATE TABLE skill_ratings (
		id TEXT PRIMARY KEY,
		skill_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		rating INTEGER NOT NULL,
		comment TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	mustExec(t, db, `CREATE UNIQUE INDEX idx_skill_user_rating ON skill_ratings(skill_id, user_id)`)
	mustExec(t, db, `CREATE TABLE skill_community_tags (
		id TEXT PRIMARY KEY,
		skill_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		tag TEXT NOT NULL,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	mustExec(t, db, `CREATE TABLE skill_versions (
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
		published_by TEXT,
		published_at DATETIME,
		created_at DATETIME,
		deleted_at DATETIME
	)`)
	mustExec(t, db, `CREATE TABLE audit_logs (
		id TEXT PRIMARY KEY,
		user_id TEXT,
		action TEXT NOT NULL,
		resource_type TEXT NOT NULL,
		resource_id TEXT,
		metadata TEXT,
		ip_address TEXT,
		user_agent TEXT,
		created_at DATETIME
	)`)
	mustExec(t, db, `CREATE TABLE api_keys (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		key_hash TEXT NOT NULL,
		name TEXT,
		prefix TEXT,
		last_used_at DATETIME,
		expires_at DATETIME,
		created_at DATETIME,
		deleted_at DATETIME
	)`)
	return database
}

func mustExec(t *testing.T, db *gorm.DB, sql string) {
	t.Helper()
	if err := db.Exec(sql).Error; err != nil {
		t.Fatalf("exec sql failed: %v\nsql=%s", err, sql)
	}
}

func currentCatalogVersion(t *testing.T, db *storage.Database) int64 {
	t.Helper()

	var state models.CatalogState
	if err := db.First(&state, "id = ?", catalogStateSingletonID).Error; err != nil {
		t.Fatalf("load catalog state failed: %v", err)
	}
	return state.CatalogVersion
}

func newAuthedRouter(user *models.User) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if user != nil {
		router.Use(func(c *gin.Context) {
			c.Set("user", user)
			c.Next()
		})
	}
	return router
}

func newTestObjectStorage(t *testing.T) *storage.ObjectStorage {
	t.Helper()

	objStorage, err := storage.NewObjectStorage(config.StorageConfig{
		Type:      "local",
		LocalPath: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create object storage failed: %v", err)
	}
	return objStorage
}

func newPublicTestObjectStorage(t *testing.T) *storage.ObjectStorage {
	t.Helper()

	objStorage, err := storage.NewObjectStorage(config.StorageConfig{
		Type:          "local",
		LocalPath:     t.TempDir(),
		PublicBaseURL: "https://skills-static.example.com",
	})
	if err != nil {
		t.Fatalf("create public object storage failed: %v", err)
	}
	return objStorage
}

func newSkillArchive(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	file, err := writer.Create("SKILL.md")
	if err != nil {
		t.Fatalf("create zip entry failed: %v", err)
	}
	if _, err := io.WriteString(file, "# Example Skill\n"); err != nil {
		t.Fatalf("write zip entry failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer failed: %v", err)
	}
	return buf.Bytes()
}

func newCreateSkillRequest(t *testing.T, fields map[string]string, archive []byte) (*http.Request, string) {
	t.Helper()
	return newCreateSkillRequestWithFilename(t, fields, "example.zip", archive)
}

func newCreateSkillRequestWithFilename(t *testing.T, fields map[string]string, filename string, archive []byte) (*http.Request, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field %s failed: %v", key, err)
		}
	}
	part, err := writer.CreateFormFile("skill", filename)
	if err != nil {
		t.Fatalf("create form file failed: %v", err)
	}
	if _, err := part.Write(archive); err != nil {
		t.Fatalf("write archive failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/skills", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, writer.FormDataContentType()
}

func TestCreateSkillRequiresAuth(t *testing.T) {
	db := newTestDatabase(t)

	router := gin.New()
	router.POST("/api/v1/skills", middleware.Auth(db), CreateSkill(db, newTestObjectStorage(t), validator.NewScanner()))

	req, _ := newCreateSkillRequest(t, map[string]string{
		"namespace": "team",
		"name":      "github",
		"version":   "1.0.0",
	}, newSkillArchive(t))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDeleteSkillRequiresAuth(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	skill := models.Skill{
		ID:          uuid.New(),
		Namespace:   "team",
		Name:        "github",
		OwnerID:     user.ID,
		Description: "code review skill",
		IsPublic:    true,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	router := gin.New()
	router.DELETE("/api/v1/skills/:namespace/:name", middleware.Auth(db), DeleteSkill(db, newTestObjectStorage(t)))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/skills/team/github", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDeleteVersionRequiresAuth(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	skill := models.Skill{
		ID:            uuid.New(),
		Namespace:     "team",
		Name:          "github",
		OwnerID:       user.ID,
		Description:   "code review skill",
		IsPublic:      true,
		LatestVersion: "1.0.0",
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	version := models.SkillVersion{
		ID:          uuid.New(),
		SkillID:     skill.ID,
		Version:     "1.0.0",
		StoragePath: "skills/team/github/test.zip",
		SizeBytes:   128,
		ScanStatus:  "pass",
		PublishedBy: user.ID,
	}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("create version failed: %v", err)
	}

	router := gin.New()
	router.DELETE("/api/v1/skills/:namespace/:name/versions/:version", middleware.Auth(db), DeleteVersion(db, newTestObjectStorage(t)))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/skills/team/github/versions/1.0.0", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPublicSkillDetailAllowsAnonymousAccess(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	skill := models.Skill{
		ID:          uuid.New(),
		Namespace:   "team",
		Name:        "github",
		OwnerID:     user.ID,
		Description: "public skill",
		IsPublic:    true,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	router := gin.New()
	router.GET("/api/v1/skills/:namespace/:name", middleware.OptionalAuth(db), GetSkill(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills/team/github", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPrivateSkillDetailRejectsAnonymousAccess(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	skill := models.Skill{
		ID:          uuid.New(),
		Namespace:   "team",
		Name:        "private-skill",
		OwnerID:     user.ID,
		Description: "private skill",
		IsPublic:    false,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}
	if err := db.Model(&skill).Update("is_public", false).Error; err != nil {
		t.Fatalf("set skill private failed: %v", err)
	}

	router := gin.New()
	router.GET("/api/v1/skills/:namespace/:name", middleware.OptionalAuth(db), GetSkill(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills/team/private-skill", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPublicSkillDetailReturnsPublicDownloadURL(t *testing.T) {
	db := newTestDatabase(t)
	objStorage := newPublicTestObjectStorage(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	archive := newSkillArchive(t)
	storagePath := "skills/team/github/test.zip"
	if err := objStorage.Upload(context.Background(), storagePath, bytes.NewReader(archive), int64(len(archive))); err != nil {
		t.Fatalf("upload archive failed: %v", err)
	}

	skill := models.Skill{
		ID:            uuid.New(),
		Namespace:     "team",
		Name:          "github",
		OwnerID:       user.ID,
		Description:   "public skill",
		IsPublic:      true,
		LatestVersion: "1.0.0",
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	version := models.SkillVersion{
		ID:          uuid.New(),
		SkillID:     skill.ID,
		Version:     "1.0.0",
		StoragePath: storagePath,
		SizeBytes:   int64(len(archive)),
		ScanStatus:  "pass",
		PublishedBy: user.ID,
	}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("create version failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.GET("/api/v1/skills/:namespace/:name", middleware.OptionalAuth(db), GetSkill(db, objStorage))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills/team/github", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp models.Skill
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	want, ok := objStorage.PublicURL(storagePath)
	if !ok {
		t.Fatalf("expected public url to resolve")
	}
	if resp.DownloadURL != want {
		t.Fatalf("download_url = %q, want %q", resp.DownloadURL, want)
	}
	if len(resp.Versions) != 1 || resp.Versions[0].DownloadURL != want {
		t.Fatalf("unexpected version download url: %+v", resp.Versions)
	}
}

func TestPublicSkillDetailUsesLatestVersionDownloadURL(t *testing.T) {
	db := newTestDatabase(t)
	objStorage := newPublicTestObjectStorage(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	oldArchive := mustZipArchive(t, map[string]string{"SKILL.md": "---\nname: github\nversion: 1.0.0\n---\nold\n"})
	oldStoragePath := "skills/team/github/old.zip"
	if err := objStorage.Upload(context.Background(), oldStoragePath, bytes.NewReader(oldArchive), int64(len(oldArchive))); err != nil {
		t.Fatalf("upload old archive failed: %v", err)
	}

	latestArchive := mustZipArchive(t, map[string]string{"SKILL.md": "---\nname: github\nversion: 1.0.1\n---\nlatest\n"})
	latestStoragePath := "skills/team/github/latest.zip"
	if err := objStorage.Upload(context.Background(), latestStoragePath, bytes.NewReader(latestArchive), int64(len(latestArchive))); err != nil {
		t.Fatalf("upload latest archive failed: %v", err)
	}

	skill := models.Skill{
		ID:            uuid.New(),
		Namespace:     "team",
		Name:          "github",
		OwnerID:       user.ID,
		Description:   "public skill",
		IsPublic:      true,
		LatestVersion: "1.0.1",
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	oldVersion := models.SkillVersion{
		ID:          uuid.New(),
		SkillID:     skill.ID,
		Version:     "1.0.0",
		StoragePath: oldStoragePath,
		SizeBytes:   int64(len(oldArchive)),
		ScanStatus:  "pass",
		PublishedBy: user.ID,
	}
	if err := db.Create(&oldVersion).Error; err != nil {
		t.Fatalf("create old version failed: %v", err)
	}

	latestVersion := models.SkillVersion{
		ID:          uuid.New(),
		SkillID:     skill.ID,
		Version:     "1.0.1",
		StoragePath: latestStoragePath,
		SizeBytes:   int64(len(latestArchive)),
		ScanStatus:  "pass",
		PublishedBy: user.ID,
	}
	if err := db.Create(&latestVersion).Error; err != nil {
		t.Fatalf("create latest version failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.GET("/api/v1/skills/:namespace/:name", middleware.OptionalAuth(db), GetSkill(db, objStorage))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills/team/github", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp models.Skill
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	want, ok := objStorage.PublicURL(latestStoragePath)
	if !ok {
		t.Fatalf("expected latest public url to resolve")
	}
	if resp.DownloadURL != want {
		t.Fatalf("download_url = %q, want latest %q", resp.DownloadURL, want)
	}
}

func TestPublicTarGzSkillDetailKeepsRelativeDownloadURL(t *testing.T) {
	db := newTestDatabase(t)
	objStorage := newPublicTestObjectStorage(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	archive := mustTarGzArchive(t, map[string]string{"SKILL.md": "# Example Skill\n"})
	storagePath := "skills/team/github/test.tar.gz"
	if err := objStorage.Upload(context.Background(), storagePath, bytes.NewReader(archive), int64(len(archive))); err != nil {
		t.Fatalf("upload archive failed: %v", err)
	}

	skill := models.Skill{
		ID:            uuid.New(),
		Namespace:     "team",
		Name:          "github",
		OwnerID:       user.ID,
		Description:   "public tgz skill",
		IsPublic:      true,
		LatestVersion: "1.0.0",
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	version := models.SkillVersion{
		ID:          uuid.New(),
		SkillID:     skill.ID,
		Version:     "1.0.0",
		StoragePath: storagePath,
		SizeBytes:   int64(len(archive)),
		ScanStatus:  "pass",
		PublishedBy: user.ID,
	}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("create version failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.GET("/api/v1/skills/:namespace/:name", middleware.OptionalAuth(db), GetSkill(db, objStorage))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills/team/github", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp models.Skill
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	want := "/api/v1/download/team/github/1.0.0"
	if resp.DownloadURL != want {
		t.Fatalf("download_url = %q, want %q", resp.DownloadURL, want)
	}
	if len(resp.Versions) != 1 || resp.Versions[0].DownloadURL != want {
		t.Fatalf("unexpected version download url: %+v", resp.Versions)
	}
}

func TestPrivateSkillDetailKeepsRelativeDownloadURL(t *testing.T) {
	db := newTestDatabase(t)
	objStorage := newPublicTestObjectStorage(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	archive := newSkillArchive(t)
	storagePath := "skills/team/private-skill/test.zip"
	if err := objStorage.Upload(context.Background(), storagePath, bytes.NewReader(archive), int64(len(archive))); err != nil {
		t.Fatalf("upload archive failed: %v", err)
	}

	skill := models.Skill{
		ID:            uuid.New(),
		Namespace:     "team",
		Name:          "private-skill",
		OwnerID:       user.ID,
		Description:   "private skill",
		IsPublic:      true,
		LatestVersion: "1.0.0",
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}
	if err := db.Model(&skill).Update("is_public", false).Error; err != nil {
		t.Fatalf("set skill private failed: %v", err)
	}

	version := models.SkillVersion{
		ID:          uuid.New(),
		SkillID:     skill.ID,
		Version:     "1.0.0",
		StoragePath: storagePath,
		SizeBytes:   int64(len(archive)),
		ScanStatus:  "pass",
		PublishedBy: user.ID,
	}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("create version failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.GET("/api/v1/skills/:namespace/:name", middleware.OptionalAuth(db), GetSkill(db, objStorage))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills/team/private-skill", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp models.Skill
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	want := "/api/v1/download/team/private-skill/1.0.0"
	if resp.DownloadURL != want {
		t.Fatalf("download_url = %q, want %q", resp.DownloadURL, want)
	}
	if len(resp.Versions) != 1 || resp.Versions[0].DownloadURL != want {
		t.Fatalf("unexpected version download url: %+v", resp.Versions)
	}
}

func TestPublicDownloadAllowsAnonymousAccess(t *testing.T) {
	db := newTestDatabase(t)
	objStorage := newTestObjectStorage(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	archive := newSkillArchive(t)
	storagePath := "skills/team/github/test.zip"
	if err := objStorage.Upload(context.Background(), storagePath, bytes.NewReader(archive), int64(len(archive))); err != nil {
		t.Fatalf("upload archive failed: %v", err)
	}

	skill := models.Skill{
		ID:            uuid.New(),
		Namespace:     "team",
		Name:          "github",
		OwnerID:       user.ID,
		Description:   "public skill",
		IsPublic:      true,
		LatestVersion: "1.0.0",
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	version := models.SkillVersion{
		ID:          uuid.New(),
		SkillID:     skill.ID,
		Version:     "1.0.0",
		StoragePath: storagePath,
		SizeBytes:   int64(len(archive)),
		ScanStatus:  "pass",
		PublishedBy: user.ID,
	}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("create version failed: %v", err)
	}

	router := gin.New()
	router.GET("/api/v1/download/:namespace/:name/:version", middleware.OptionalAuth(db), DownloadSkill(db, objStorage))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/download/team/github/1.0.0?format=zip", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(rec.Body.Bytes()) == 0 {
		t.Fatalf("expected download payload")
	}
}

func TestCreateSkillReturnsPublicDownloadURL(t *testing.T) {
	db := newTestDatabase(t)
	objStorage := newPublicTestObjectStorage(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.POST("/api/v1/skills", CreateSkill(db, objStorage, validator.NewScanner()))

	req, _ := newCreateSkillRequest(t, map[string]string{
		"namespace": "team",
		"name":      "github",
		"version":   "1.0.0",
	}, newSkillArchive(t))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	var skill models.Skill
	if err := db.Where("namespace = ? AND name = ?", "team", "github").First(&skill).Error; err != nil {
		t.Fatalf("load created skill failed: %v", err)
	}
	var version models.SkillVersion
	if err := db.Where("skill_id = ? AND version = ?", skill.ID, "1.0.0").First(&version).Error; err != nil {
		t.Fatalf("load created version failed: %v", err)
	}

	got, _ := resp["download_url"].(string)
	want, ok := objStorage.PublicURL(version.StoragePath)
	if !ok {
		t.Fatalf("expected public url to resolve")
	}
	if got != want {
		t.Fatalf("download_url = %q, want %q", got, want)
	}
}

func TestCreateTarGzSkillKeepsRelativeDownloadURL(t *testing.T) {
	db := newTestDatabase(t)
	objStorage := newPublicTestObjectStorage(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.POST("/api/v1/skills", CreateSkill(db, objStorage, validator.NewScanner()))

	req, _ := newCreateSkillRequestWithFilename(t, map[string]string{
		"namespace": "team",
		"name":      "github-tgz",
		"version":   "1.0.0",
	}, "example.tar.gz", mustTarGzArchive(t, map[string]string{"SKILL.md": "# Example Skill\n"}))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	got, _ := resp["download_url"].(string)
	want := "/api/v1/download/team/github-tgz/1.0.0"
	if got != want {
		t.Fatalf("download_url = %q, want %q", got, want)
	}
}

func TestCreatePrivateSkillKeepsRelativeDownloadURL(t *testing.T) {
	db := newTestDatabase(t)
	objStorage := newPublicTestObjectStorage(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.POST("/api/v1/skills", CreateSkill(db, objStorage, validator.NewScanner()))

	req, _ := newCreateSkillRequest(t, map[string]string{
		"namespace": "team",
		"name":      "private-github",
		"version":   "1.0.0",
		"is_public": "false",
	}, newSkillArchive(t))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	got, _ := resp["download_url"].(string)
	want := "/api/v1/download/team/private-github/1.0.0"
	if got != want {
		t.Fatalf("download_url = %q, want %q", got, want)
	}
}

func TestCatalogVersionBumpsOnSkillMutations(t *testing.T) {
	t.Run("create skill", func(t *testing.T) {
		db := newTestDatabase(t)
		user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user failed: %v", err)
		}

		router := newAuthedRouter(user)
		router.POST("/api/v1/skills", CreateSkill(db, newTestObjectStorage(t), validator.NewScanner()))

		req, _ := newCreateSkillRequest(t, map[string]string{
			"namespace": "team",
			"name":      "github",
			"version":   "1.0.0",
		}, newSkillArchive(t))

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
		}
		if got := currentCatalogVersion(t, db); got != 2 {
			t.Fatalf("catalog_version = %d, want 2", got)
		}
	})

	t.Run("update skill", func(t *testing.T) {
		db := newTestDatabase(t)
		user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user failed: %v", err)
		}
		skill := models.Skill{
			ID:          uuid.New(),
			Namespace:   "team",
			Name:        "github",
			OwnerID:     user.ID,
			Description: "before",
			IsPublic:    true,
		}
		if err := db.Create(&skill).Error; err != nil {
			t.Fatalf("create skill failed: %v", err)
		}

		router := newAuthedRouter(user)
		router.PUT("/api/v1/skills/:namespace/:name", UpdateSkill(db))

		body := bytes.NewBufferString(`{"description":"after","tags":["review"],"license":"Apache-2.0","is_public":true}`)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/skills/team/github", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
		}
		if got := currentCatalogVersion(t, db); got != 2 {
			t.Fatalf("catalog_version = %d, want 2", got)
		}
	})

	t.Run("delete skill", func(t *testing.T) {
		db := newTestDatabase(t)
		user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user failed: %v", err)
		}
		skill := models.Skill{
			ID:          uuid.New(),
			Namespace:   "team",
			Name:        "github",
			OwnerID:     user.ID,
			Description: "code review skill",
			IsPublic:    true,
		}
		if err := db.Create(&skill).Error; err != nil {
			t.Fatalf("create skill failed: %v", err)
		}

		router := newAuthedRouter(user)
		router.DELETE("/api/v1/skills/:namespace/:name", DeleteSkill(db, newTestObjectStorage(t)))

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/skills/team/github", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
		}
		if got := currentCatalogVersion(t, db); got != 2 {
			t.Fatalf("catalog_version = %d, want 2", got)
		}
	})

	t.Run("publish version", func(t *testing.T) {
		db := newTestDatabase(t)
		objStorage := newTestObjectStorage(t)
		user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user failed: %v", err)
		}
		skill := models.Skill{
			ID:            uuid.New(),
			Namespace:     "team",
			Name:          "github",
			OwnerID:       user.ID,
			IsPublic:      true,
			LatestVersion: "1.0.0",
		}
		if err := db.Create(&skill).Error; err != nil {
			t.Fatalf("create skill failed: %v", err)
		}

		router := newAuthedRouter(user)
		router.POST("/api/v1/skills/:namespace/:name/versions", PublishVersion(db, objStorage, validator.NewScanner()))

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		if err := writer.WriteField("version", "1.1.0"); err != nil {
			t.Fatalf("WriteField version returned error: %v", err)
		}
		part, err := writer.CreateFormFile("skill", "github.zip")
		if err != nil {
			t.Fatalf("CreateFormFile returned error: %v", err)
		}
		if _, err := part.Write(newSkillArchive(t)); err != nil {
			t.Fatalf("part.Write returned error: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("writer.Close returned error: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/v1/skills/team/github/versions", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
		}
		if got := currentCatalogVersion(t, db); got != 2 {
			t.Fatalf("catalog_version = %d, want 2", got)
		}
	})

	t.Run("delete version", func(t *testing.T) {
		db := newTestDatabase(t)
		user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user failed: %v", err)
		}
		skill := models.Skill{
			ID:            uuid.New(),
			Namespace:     "team",
			Name:          "github",
			OwnerID:       user.ID,
			IsPublic:      true,
			LatestVersion: "1.0.0",
		}
		if err := db.Create(&skill).Error; err != nil {
			t.Fatalf("create skill failed: %v", err)
		}
		version := models.SkillVersion{
			ID:          uuid.New(),
			SkillID:     skill.ID,
			Version:     "1.0.0",
			StoragePath: "skills/team/github/test.zip",
			SizeBytes:   128,
			ScanStatus:  "pass",
			PublishedBy: user.ID,
		}
		if err := db.Create(&version).Error; err != nil {
			t.Fatalf("create version failed: %v", err)
		}

		router := newAuthedRouter(user)
		router.DELETE("/api/v1/skills/:namespace/:name/versions/:version", DeleteVersion(db, newTestObjectStorage(t)))

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/skills/team/github/versions/1.0.0", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
		}
		if got := currentCatalogVersion(t, db); got != 2 {
			t.Fatalf("catalog_version = %d, want 2", got)
		}
	})
}

func TestUpdateSkillVisibilityTransitionsBumpCatalogVersion(t *testing.T) {
	t.Run("public to private", func(t *testing.T) {
		db := newTestDatabase(t)
		user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user failed: %v", err)
		}
		skill := models.Skill{
			ID:          uuid.New(),
			Namespace:   "team",
			Name:        "github",
			OwnerID:     user.ID,
			Description: "before",
			IsPublic:    true,
		}
		if err := db.Create(&skill).Error; err != nil {
			t.Fatalf("create skill failed: %v", err)
		}

		router := newAuthedRouter(user)
		router.PUT("/api/v1/skills/:namespace/:name", UpdateSkill(db))

		body := bytes.NewBufferString(`{"description":"after","tags":["review"],"license":"Apache-2.0","is_public":false}`)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/skills/team/github", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
		}
		if got := currentCatalogVersion(t, db); got != 2 {
			t.Fatalf("catalog_version = %d, want 2", got)
		}
	})

	t.Run("private to public", func(t *testing.T) {
		db := newTestDatabase(t)
		user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user failed: %v", err)
		}
		skill := models.Skill{
			ID:          uuid.New(),
			Namespace:   "team",
			Name:        "github",
			OwnerID:     user.ID,
			Description: "before",
			IsPublic:    true,
		}
		if err := db.Create(&skill).Error; err != nil {
			t.Fatalf("create skill failed: %v", err)
		}
		if err := db.Model(&skill).Update("is_public", false).Error; err != nil {
			t.Fatalf("set skill private failed: %v", err)
		}

		router := newAuthedRouter(user)
		router.PUT("/api/v1/skills/:namespace/:name", UpdateSkill(db))

		body := bytes.NewBufferString(`{"description":"after","tags":["review"],"license":"Apache-2.0","is_public":true}`)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/skills/team/github", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
		}
		if got := currentCatalogVersion(t, db); got != 2 {
			t.Fatalf("catalog_version = %d, want 2", got)
		}
	})
}

func TestCreateSkillReturnsAlreadyExistsConflictDoesNotBump(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	existing := models.Skill{
		ID:          uuid.New(),
		Namespace:   "team",
		Name:        "github",
		OwnerID:     user.ID,
		Description: "existing skill",
		IsPublic:    true,
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("create existing skill failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.POST("/api/v1/skills", CreateSkill(db, newTestObjectStorage(t), validator.NewScanner()))

	req, _ := newCreateSkillRequest(t, map[string]string{
		"namespace": "team",
		"name":      "github",
		"version":   "1.0.0",
	}, newSkillArchive(t))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	if got := currentCatalogVersion(t, db); got != 1 {
		t.Fatalf("catalog_version = %d, want 1", got)
	}
}

func TestPrivateSkillMutationsDoNotBumpCatalogVersion(t *testing.T) {
	t.Run("create skill", func(t *testing.T) {
		db := newTestDatabase(t)
		user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user failed: %v", err)
		}

		router := newAuthedRouter(user)
		router.POST("/api/v1/skills", CreateSkill(db, newTestObjectStorage(t), validator.NewScanner()))

		req, _ := newCreateSkillRequest(t, map[string]string{
			"namespace": "team",
			"name":      "private-github",
			"version":   "1.0.0",
			"is_public": "false",
		}, newSkillArchive(t))

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
		}
		if got := currentCatalogVersion(t, db); got != 1 {
			t.Fatalf("catalog_version = %d, want 1", got)
		}
	})

	t.Run("update skill", func(t *testing.T) {
		db := newTestDatabase(t)
		user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user failed: %v", err)
		}
		skill := models.Skill{
			ID:          uuid.New(),
			Namespace:   "team",
			Name:        "private-github",
			OwnerID:     user.ID,
			Description: "before",
			IsPublic:    true,
		}
		if err := db.Create(&skill).Error; err != nil {
			t.Fatalf("create skill failed: %v", err)
		}
		if err := db.Model(&skill).Update("is_public", false).Error; err != nil {
			t.Fatalf("set skill private failed: %v", err)
		}

		router := newAuthedRouter(user)
		router.PUT("/api/v1/skills/:namespace/:name", UpdateSkill(db))

		body := bytes.NewBufferString(`{"description":"after","tags":["review"],"license":"Apache-2.0","is_public":false}`)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/skills/team/private-github", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
		}
		if got := currentCatalogVersion(t, db); got != 1 {
			t.Fatalf("catalog_version = %d, want 1", got)
		}
	})

	t.Run("delete skill", func(t *testing.T) {
		db := newTestDatabase(t)
		user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user failed: %v", err)
		}
		skill := models.Skill{
			ID:          uuid.New(),
			Namespace:   "team",
			Name:        "private-github",
			OwnerID:     user.ID,
			Description: "private skill",
			IsPublic:    true,
		}
		if err := db.Create(&skill).Error; err != nil {
			t.Fatalf("create skill failed: %v", err)
		}
		if err := db.Model(&skill).Update("is_public", false).Error; err != nil {
			t.Fatalf("set skill private failed: %v", err)
		}

		router := newAuthedRouter(user)
		router.DELETE("/api/v1/skills/:namespace/:name", DeleteSkill(db, newTestObjectStorage(t)))

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/skills/team/private-github", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
		}
		if got := currentCatalogVersion(t, db); got != 1 {
			t.Fatalf("catalog_version = %d, want 1", got)
		}
	})

	t.Run("publish version", func(t *testing.T) {
		db := newTestDatabase(t)
		objStorage := newTestObjectStorage(t)
		user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user failed: %v", err)
		}
		skill := models.Skill{
			ID:            uuid.New(),
			Namespace:     "team",
			Name:          "private-github",
			OwnerID:       user.ID,
			IsPublic:      true,
			LatestVersion: "1.0.0",
		}
		if err := db.Create(&skill).Error; err != nil {
			t.Fatalf("create skill failed: %v", err)
		}
		if err := db.Model(&skill).Update("is_public", false).Error; err != nil {
			t.Fatalf("set skill private failed: %v", err)
		}

		router := newAuthedRouter(user)
		router.POST("/api/v1/skills/:namespace/:name/versions", PublishVersion(db, objStorage, validator.NewScanner()))

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		if err := writer.WriteField("version", "1.1.0"); err != nil {
			t.Fatalf("WriteField version returned error: %v", err)
		}
		part, err := writer.CreateFormFile("skill", "private-github.zip")
		if err != nil {
			t.Fatalf("CreateFormFile returned error: %v", err)
		}
		if _, err := part.Write(newSkillArchive(t)); err != nil {
			t.Fatalf("part.Write returned error: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("writer.Close returned error: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/v1/skills/team/private-github/versions", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
		}
		if got := currentCatalogVersion(t, db); got != 1 {
			t.Fatalf("catalog_version = %d, want 1", got)
		}
	})

	t.Run("delete version", func(t *testing.T) {
		db := newTestDatabase(t)
		user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user failed: %v", err)
		}
		skill := models.Skill{
			ID:            uuid.New(),
			Namespace:     "team",
			Name:          "private-github",
			OwnerID:       user.ID,
			IsPublic:      true,
			LatestVersion: "1.0.0",
		}
		if err := db.Create(&skill).Error; err != nil {
			t.Fatalf("create skill failed: %v", err)
		}
		if err := db.Model(&skill).Update("is_public", false).Error; err != nil {
			t.Fatalf("set skill private failed: %v", err)
		}
		version := models.SkillVersion{
			ID:          uuid.New(),
			SkillID:     skill.ID,
			Version:     "1.0.0",
			StoragePath: "skills/team/private-github/test.zip",
			SizeBytes:   128,
			ScanStatus:  "pass",
			PublishedBy: user.ID,
		}
		if err := db.Create(&version).Error; err != nil {
			t.Fatalf("create version failed: %v", err)
		}

		router := newAuthedRouter(user)
		router.DELETE("/api/v1/skills/:namespace/:name/versions/:version", DeleteVersion(db, newTestObjectStorage(t)))

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/skills/team/private-github/versions/1.0.0", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
		}
		if got := currentCatalogVersion(t, db); got != 1 {
			t.Fatalf("catalog_version = %d, want 1", got)
		}
	})
}

func TestCreateSkillRollsBackWhenAuditLogWriteFails(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	if err := db.Callback().Create().Before("gorm:create").Register("catalog_state_fail_audit_log", func(tx *gorm.DB) {
		if tx.Statement.Table == "audit_logs" {
			tx.AddError(errors.New("forced audit log failure"))
		}
	}); err != nil {
		t.Fatalf("register audit log failure callback failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.POST("/api/v1/skills", CreateSkill(db, newTestObjectStorage(t), validator.NewScanner()))

	req, _ := newCreateSkillRequest(t, map[string]string{
		"namespace": "team",
		"name":      "github",
		"version":   "1.0.0",
	}, newSkillArchive(t))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	if got := currentCatalogVersion(t, db); got != 1 {
		t.Fatalf("catalog_version = %d, want 1", got)
	}

	var skill models.Skill
	if err := db.Where("namespace = ? AND name = ?", "team", "github").First(&skill).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected no skill to be committed, got err=%v", err)
	}
}

func TestPublishVersionRollsBackWhenAuditLogWriteFails(t *testing.T) {
	db := newTestDatabase(t)
	objStorage := newTestObjectStorage(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	skill := models.Skill{
		ID:            uuid.New(),
		Namespace:     "team",
		Name:          "github",
		OwnerID:       user.ID,
		IsPublic:      true,
		LatestVersion: "1.0.0",
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}
	if err := db.Create(&models.SkillVersion{
		ID:          uuid.New(),
		SkillID:     skill.ID,
		Version:     "1.0.0",
		StoragePath: "skills/team/github/base.zip",
		SizeBytes:   16,
		ScanStatus:  "pass",
		PublishedBy: user.ID,
		PublishedAt: time.Now(),
	}).Error; err != nil {
		t.Fatalf("create base version failed: %v", err)
	}

	if err := db.Callback().Create().Before("gorm:create").Register("catalog_state_fail_publish_audit_log", func(tx *gorm.DB) {
		if tx.Statement.Table == "audit_logs" {
			tx.AddError(errors.New("forced audit log failure"))
		}
	}); err != nil {
		t.Fatalf("register audit log failure callback failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.POST("/api/v1/skills/:namespace/:name/versions", PublishVersion(db, objStorage, validator.NewScanner()))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("version", "1.1.0"); err != nil {
		t.Fatalf("WriteField version returned error: %v", err)
	}
	part, err := writer.CreateFormFile("skill", "github.zip")
	if err != nil {
		t.Fatalf("CreateFormFile returned error: %v", err)
	}
	if _, err := part.Write(newSkillArchive(t)); err != nil {
		t.Fatalf("part.Write returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/skills/team/github/versions", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	if got := currentCatalogVersion(t, db); got != 1 {
		t.Fatalf("catalog_version = %d, want 1", got)
	}

	var reloaded models.Skill
	if err := db.First(&reloaded, "id = ?", skill.ID).Error; err != nil {
		t.Fatalf("reload skill failed: %v", err)
	}
	if reloaded.LatestVersion != "1.0.0" {
		t.Fatalf("latest_version = %q, want 1.0.0", reloaded.LatestVersion)
	}

	var version models.SkillVersion
	if err := db.Where("skill_id = ? AND version = ?", skill.ID, "1.1.0").First(&version).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected no published version to be committed, got err=%v", err)
	}
}

func TestPublishVersionReturnsPublicDownloadURL(t *testing.T) {
	db := newTestDatabase(t)
	objStorage := newPublicTestObjectStorage(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	skill := models.Skill{
		ID:            uuid.New(),
		Namespace:     "team",
		Name:          "github",
		OwnerID:       user.ID,
		IsPublic:      true,
		LatestVersion: "1.0.0",
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.POST("/api/v1/skills/:namespace/:name/versions", PublishVersion(db, objStorage, validator.NewScanner()))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("version", "1.1.0"); err != nil {
		t.Fatalf("WriteField version returned error: %v", err)
	}
	part, err := writer.CreateFormFile("skill", "github.zip")
	if err != nil {
		t.Fatalf("CreateFormFile returned error: %v", err)
	}
	if _, err := part.Write(newSkillArchive(t)); err != nil {
		t.Fatalf("part.Write returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/skills/team/github/versions", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	var version models.SkillVersion
	if err := db.Where("skill_id = ? AND version = ?", skill.ID, "1.1.0").First(&version).Error; err != nil {
		t.Fatalf("load created version failed: %v", err)
	}

	got, _ := resp["download_url"].(string)
	want, ok := objStorage.PublicURL(version.StoragePath)
	if !ok {
		t.Fatalf("expected public url to resolve")
	}
	if got != want {
		t.Fatalf("download_url = %q, want %q", got, want)
	}
}

func TestDownloadSkillRedirectsPublicZipToOSS(t *testing.T) {
	db := newTestDatabase(t)
	objStorage := newPublicTestObjectStorage(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	archive := newSkillArchive(t)
	storagePath := "skills/team/github/test.zip"
	if err := objStorage.Upload(context.Background(), storagePath, bytes.NewReader(archive), int64(len(archive))); err != nil {
		t.Fatalf("upload archive failed: %v", err)
	}

	skill := models.Skill{
		ID:            uuid.New(),
		Namespace:     "team",
		Name:          "github",
		OwnerID:       user.ID,
		Description:   "public skill",
		IsPublic:      true,
		LatestVersion: "1.0.0",
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	version := models.SkillVersion{
		ID:          uuid.New(),
		SkillID:     skill.ID,
		Version:     "1.0.0",
		StoragePath: storagePath,
		SizeBytes:   int64(len(archive)),
		ScanStatus:  "pass",
		PublishedBy: user.ID,
	}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("create version failed: %v", err)
	}

	router := gin.New()
	router.GET("/api/v1/download/:namespace/:name/:version", middleware.OptionalAuth(db), DownloadSkill(db, objStorage))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/download/team/github/1.0.0?format=zip", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d body=%s", rec.Code, rec.Body.String())
	}
	want, ok := objStorage.PublicURL(storagePath)
	if !ok {
		t.Fatalf("expected public url to resolve")
	}
	if got := rec.Header().Get("Location"); got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
}

func TestDownloadSkillDoesNotRedirectPublicTarGzForDefaultZipRequest(t *testing.T) {
	db := newTestDatabase(t)
	objStorage := newPublicTestObjectStorage(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	archive := mustTarGzArchive(t, map[string]string{"SKILL.md": "# Example Skill\n"})
	storagePath := "skills/team/github/test.tar.gz"
	if err := objStorage.Upload(context.Background(), storagePath, bytes.NewReader(archive), int64(len(archive))); err != nil {
		t.Fatalf("upload archive failed: %v", err)
	}

	skill := models.Skill{
		ID:            uuid.New(),
		Namespace:     "team",
		Name:          "github",
		OwnerID:       user.ID,
		Description:   "public tgz skill",
		IsPublic:      true,
		LatestVersion: "1.0.0",
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	version := models.SkillVersion{
		ID:          uuid.New(),
		SkillID:     skill.ID,
		Version:     "1.0.0",
		StoragePath: storagePath,
		SizeBytes:   int64(len(archive)),
		ScanStatus:  "pass",
		PublishedBy: user.ID,
	}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("create version failed: %v", err)
	}

	router := gin.New()
	router.GET("/api/v1/download/:namespace/:name/:version", middleware.OptionalAuth(db), DownloadSkill(db, objStorage))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/download/team/github/1.0.0", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "" {
		t.Fatalf("expected no redirect Location, got %q", got)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/zip" {
		t.Fatalf("Content-Type = %q, want application/zip", got)
	}
	if disposition := rec.Header().Get("Content-Disposition"); !strings.Contains(disposition, "github-1.0.0.zip") {
		t.Fatalf("Content-Disposition = %q, want zip download filename", disposition)
	}
}

func TestDownloadSkillFallsBackWithoutPublicBaseURL(t *testing.T) {
	db := newTestDatabase(t)
	objStorage := newTestObjectStorage(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	archive := newSkillArchive(t)
	storagePath := "skills/team/github/test.zip"
	if err := objStorage.Upload(context.Background(), storagePath, bytes.NewReader(archive), int64(len(archive))); err != nil {
		t.Fatalf("upload archive failed: %v", err)
	}

	skill := models.Skill{
		ID:            uuid.New(),
		Namespace:     "team",
		Name:          "github",
		OwnerID:       user.ID,
		Description:   "public skill",
		IsPublic:      true,
		LatestVersion: "1.0.0",
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	version := models.SkillVersion{
		ID:          uuid.New(),
		SkillID:     skill.ID,
		Version:     "1.0.0",
		StoragePath: storagePath,
		SizeBytes:   int64(len(archive)),
		ScanStatus:  "pass",
		PublishedBy: user.ID,
	}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("create version failed: %v", err)
	}

	router := gin.New()
	router.GET("/api/v1/download/:namespace/:name/:version", middleware.OptionalAuth(db), DownloadSkill(db, objStorage))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/download/team/github/1.0.0?format=zip", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "" {
		t.Fatalf("expected no redirect Location, got %q", got)
	}
	if len(rec.Body.Bytes()) == 0 {
		t.Fatalf("expected download payload")
	}
}

func TestPrivateDownloadRejectsAnonymousAccess(t *testing.T) {
	db := newTestDatabase(t)
	objStorage := newTestObjectStorage(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	archive := newSkillArchive(t)
	storagePath := "skills/team/private-skill/test.zip"
	if err := objStorage.Upload(context.Background(), storagePath, bytes.NewReader(archive), int64(len(archive))); err != nil {
		t.Fatalf("upload archive failed: %v", err)
	}

	skill := models.Skill{
		ID:            uuid.New(),
		Namespace:     "team",
		Name:          "private-skill",
		OwnerID:       user.ID,
		Description:   "private skill",
		IsPublic:      false,
		LatestVersion: "1.0.0",
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}
	if err := db.Model(&skill).Update("is_public", false).Error; err != nil {
		t.Fatalf("set skill private failed: %v", err)
	}

	version := models.SkillVersion{
		ID:          uuid.New(),
		SkillID:     skill.ID,
		Version:     "1.0.0",
		StoragePath: storagePath,
		SizeBytes:   int64(len(archive)),
		ScanStatus:  "pass",
		PublishedBy: user.ID,
	}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("create version failed: %v", err)
	}

	router := gin.New()
	router.GET("/api/v1/download/:namespace/:name/:version", middleware.OptionalAuth(db), DownloadSkill(db, objStorage))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/download/team/private-skill/1.0.0?format=zip", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRateSkillCreatesRatingAndAuditLog(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "alice", Email: "alice@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	skill := models.Skill{
		ID:          uuid.New(),
		Namespace:   "team",
		Name:        "reviewer",
		OwnerID:     user.ID,
		Description: "code review skill",
		IsPublic:    true,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.POST("/api/v1/skills/:namespace/:name/rating", RateSkill(db))

	body := bytes.NewBufferString(`{"rating":5,"comment":"great"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/skills/team/reviewer/rating", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var updated models.Skill
	if err := db.First(&updated, "id = ?", skill.ID).Error; err != nil {
		t.Fatalf("reload skill failed: %v", err)
	}
	if updated.RatingCount != 1 || updated.RatingSum != 5 {
		t.Fatalf("unexpected rating aggregate: %+v", updated)
	}

	var ratings []models.SkillRating
	if err := db.Find(&ratings).Error; err != nil {
		t.Fatalf("list ratings failed: %v", err)
	}
	if len(ratings) != 1 || ratings[0].Rating != 5 {
		t.Fatalf("unexpected ratings: %+v", ratings)
	}

	var logs []models.AuditLog
	if err := db.Find(&logs).Error; err != nil {
		t.Fatalf("list audit logs failed: %v", err)
	}
	if len(logs) != 1 || logs[0].Action != "skill.rate" {
		t.Fatalf("unexpected audit logs: %+v", logs)
	}
}

func TestRateSkillUpdatesExistingRatingWithoutCreatingDuplicate(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "alice", Email: "alice@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	skill := models.Skill{
		ID:          uuid.New(),
		Namespace:   "team",
		Name:        "reviewer",
		OwnerID:     user.ID,
		Description: "code review skill",
		IsPublic:    true,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	existing := models.SkillRating{
		ID:        uuid.New(),
		SkillID:   skill.ID,
		UserID:    user.ID,
		Rating:    2,
		Comment:   "old",
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("create existing rating failed: %v", err)
	}
	if err := db.Model(&models.Skill{}).
		Where("id = ?", skill.ID).
		Updates(map[string]interface{}{"rating_sum": 2, "rating_count": 1}).Error; err != nil {
		t.Fatalf("seed rating aggregate failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.POST("/api/v1/skills/:namespace/:name/rating", RateSkill(db))

	body := bytes.NewBufferString(`{"rating":5,"comment":"updated"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/skills/team/reviewer/rating", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var updated models.Skill
	if err := db.First(&updated, "id = ?", skill.ID).Error; err != nil {
		t.Fatalf("reload skill failed: %v", err)
	}
	if updated.RatingCount != 1 || updated.RatingSum != 5 {
		t.Fatalf("unexpected rating aggregate after update: %+v", updated)
	}

	var ratings []models.SkillRating
	if err := db.Where("skill_id = ?", skill.ID).Find(&ratings).Error; err != nil {
		t.Fatalf("list ratings failed: %v", err)
	}
	if len(ratings) != 1 || ratings[0].Rating != 5 || ratings[0].Comment != "updated" {
		t.Fatalf("unexpected ratings after update: %+v", ratings)
	}
}

func TestCreateSkillSetsPublishedAtOnInitialVersion(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "alice", Email: "alice@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	objStorage, err := storage.NewObjectStorage(config.StorageConfig{
		Type:      "local",
		LocalPath: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewObjectStorage returned error: %v", err)
	}

	router := newAuthedRouter(user)
	router.POST("/api/v1/skills", CreateSkill(db, objStorage, validator.NewScanner()))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("namespace", "alice"); err != nil {
		t.Fatalf("WriteField namespace returned error: %v", err)
	}
	if err := writer.WriteField("name", "first-skill"); err != nil {
		t.Fatalf("WriteField name returned error: %v", err)
	}
	if err := writer.WriteField("version", "0.1.0"); err != nil {
		t.Fatalf("WriteField version returned error: %v", err)
	}
	part, err := writer.CreateFormFile("skill", "first-skill.zip")
	if err != nil {
		t.Fatalf("CreateFormFile returned error: %v", err)
	}
	if _, err := part.Write(mustZipArchive(t, map[string]string{"SKILL.md": "name: first-skill"})); err != nil {
		t.Fatalf("part.Write returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/skills", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var skill models.Skill
	if err := db.Where("namespace = ? AND name = ?", "alice", "first-skill").First(&skill).Error; err != nil {
		t.Fatalf("load created skill failed: %v", err)
	}

	var version models.SkillVersion
	if err := db.Where("skill_id = ? AND version = ?", skill.ID, "0.1.0").First(&version).Error; err != nil {
		t.Fatalf("load created version failed: %v", err)
	}
	if version.PublishedAt.IsZero() {
		t.Fatalf("expected published_at to be set")
	}
}

func TestPublishVersionSetsPublishedAt(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "alice", Email: "alice@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	skill := models.Skill{
		ID:            uuid.New(),
		Namespace:     "alice",
		Name:          "first-skill",
		OwnerID:       user.ID,
		IsPublic:      true,
		LatestVersion: "0.1.0",
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	objStorage, err := storage.NewObjectStorage(config.StorageConfig{
		Type:      "local",
		LocalPath: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewObjectStorage returned error: %v", err)
	}

	router := newAuthedRouter(user)
	router.POST("/api/v1/skills/:namespace/:name/versions", PublishVersion(db, objStorage, validator.NewScanner()))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("version", "0.2.0"); err != nil {
		t.Fatalf("WriteField version returned error: %v", err)
	}
	part, err := writer.CreateFormFile("skill", "first-skill.zip")
	if err != nil {
		t.Fatalf("CreateFormFile returned error: %v", err)
	}
	if _, err := part.Write(mustZipArchive(t, map[string]string{"SKILL.md": "name: first-skill"})); err != nil {
		t.Fatalf("part.Write returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/skills/alice/first-skill/versions", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if publishedAt, ok := resp["published_at"].(string); !ok || publishedAt == "" {
		t.Fatalf("expected published_at in response, got %#v", resp["published_at"])
	}

	var version models.SkillVersion
	if err := db.Where("skill_id = ? AND version = ?", skill.ID, "0.2.0").First(&version).Error; err != nil {
		t.Fatalf("load created version failed: %v", err)
	}
	if version.PublishedAt.IsZero() {
		t.Fatalf("expected published_at to be set")
	}
}

func TestSearchSkillsSupportsSQLiteFallback(t *testing.T) {
	db := newTestDatabase(t)
	ownerID := uuid.New()
	skills := []models.Skill{
		{ID: uuid.New(), Namespace: "team", Name: "lint-checker", OwnerID: ownerID, Description: "lint code", IsPublic: true},
		{ID: uuid.New(), Namespace: "team", Name: "deploy-helper", OwnerID: ownerID, Description: "deploy service", IsPublic: true},
	}
	if err := db.Create(&skills).Error; err != nil {
		t.Fatalf("seed skills failed: %v", err)
	}

	router := gin.New()
	router.GET("/api/v1/search", SearchSkills(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=lint", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Results []models.Skill `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(resp.Results) != 1 || resp.Results[0].Name != "lint-checker" {
		t.Fatalf("unexpected search results: %+v", resp.Results)
	}
}

func TestSearchSkillsPrefersNewestAmongExactNameMatches(t *testing.T) {
	db := newTestDatabase(t)
	ownerID := uuid.New()
	now := time.Now()
	skills := []models.Skill{
		{
			ID:            uuid.New(),
			Namespace:     "legacy",
			Name:          "skill-home-manager",
			OwnerID:       ownerID,
			Description:   "older exact match",
			IsPublic:      true,
			DownloadCount: 20,
			CreatedAt:     now.Add(-48 * time.Hour),
			UpdatedAt:     now.Add(-48 * time.Hour),
		},
		{
			ID:            uuid.New(),
			Namespace:     "skill-home",
			Name:          "skill-home-manager",
			OwnerID:       ownerID,
			Description:   "newer exact match",
			IsPublic:      true,
			DownloadCount: 1,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
	if err := db.Create(&skills).Error; err != nil {
		t.Fatalf("seed skills failed: %v", err)
	}

	router := gin.New()
	router.GET("/api/v1/search", SearchSkills(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=skill-home-manager", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Results []models.Skill `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("unexpected search results count: %+v", resp.Results)
	}
	if resp.Results[0].Namespace != "skill-home" {
		t.Fatalf("expected newest exact match first, got %+v", resp.Results)
	}
}

func TestListAPIKeysReturnsCurrentUsersKeys(t *testing.T) {
	db := newTestDatabase(t)

	user := &models.User{ID: uuid.New(), Username: "alice", Email: "alice@example.com"}
	otherUser := &models.User{ID: uuid.New(), Username: "bob", Email: "bob@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	if err := db.Create(otherUser).Error; err != nil {
		t.Fatalf("create other user failed: %v", err)
	}

	now := time.Now().UTC()
	later := now.Add(2 * time.Hour)
	keys := []models.APIKey{
		{
			ID:        uuid.New(),
			UserID:    user.ID,
			KeyHash:   "hash-1",
			Name:      "Deploy",
			Prefix:    "sk_old12",
			CreatedAt: now,
		},
		{
			ID:         uuid.New(),
			UserID:     user.ID,
			KeyHash:    "hash-2",
			Name:       "CI",
			Prefix:     "sk_new34",
			LastUsedAt: &later,
			CreatedAt:  later,
		},
		{
			ID:        uuid.New(),
			UserID:    otherUser.ID,
			KeyHash:   "hash-3",
			Name:      "Other",
			Prefix:    "sk_oth56",
			CreatedAt: later,
		},
	}
	if err := db.Create(&keys).Error; err != nil {
		t.Fatalf("seed api keys failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.GET("/api/v1/user/api-keys", ListAPIKeys(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/api-keys", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp []APIKeySummaryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if len(resp) != 2 {
		t.Fatalf("unexpected api key count: %+v", resp)
	}
	if resp[0].Name != "CI" || resp[1].Name != "Deploy" {
		t.Fatalf("unexpected key order: %+v", resp)
	}
	if resp[0].Prefix != "sk_new34" || resp[1].Prefix != "sk_old12" {
		t.Fatalf("unexpected prefixes: %+v", resp)
	}
}

func TestCreateSkillPersistsOfficialTagsFromPublishForm(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.POST("/api/v1/skills", CreateSkill(db, newTestObjectStorage(t), validator.NewScanner()))

	req, _ := newCreateSkillRequest(t, map[string]string{
		"namespace":   "team",
		"name":        "github",
		"description": "Interact with GitHub using gh.",
		"version":     "1.0.0",
		"license":     "MIT",
		"tags":        "automation, github",
		"is_public":   "true",
	}, newSkillArchive(t))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var skill models.Skill
	if err := scopeNamespaceName(db.DB, "team", "github").First(&skill).Error; err != nil {
		t.Fatalf("load skill failed: %v", err)
	}

	if len(skill.Tags) != 2 || skill.Tags[0] != "automation" || skill.Tags[1] != "github" {
		t.Fatalf("unexpected tags: %+v", skill.Tags)
	}
}

func TestCommunityTagsRoundTripAndAggregatePerViewer(t *testing.T) {
	db := newTestDatabase(t)

	owner := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	viewer := &models.User{ID: uuid.New(), Username: "alice", Email: "alice@example.com"}
	otherViewer := &models.User{ID: uuid.New(), Username: "bob", Email: "bob@example.com"}
	if err := db.Create([]*models.User{owner, viewer, otherViewer}).Error; err != nil {
		t.Fatalf("create users failed: %v", err)
	}

	skill := models.Skill{
		ID:          uuid.New(),
		Namespace:   "team",
		Name:        "github",
		OwnerID:     owner.ID,
		Description: "Interact with GitHub using gh.",
		IsPublic:    true,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	seeded := []models.SkillCommunityTag{
		{ID: uuid.New(), SkillID: skill.ID, UserID: otherViewer.ID, Tag: "ci-deploy"},
		{ID: uuid.New(), SkillID: skill.ID, UserID: otherViewer.ID, Tag: "ops"},
	}
	if err := db.Create(&seeded).Error; err != nil {
		t.Fatalf("seed community tags failed: %v", err)
	}

	router := newAuthedRouter(viewer)
	router.GET("/api/v1/skills/:namespace/:name", GetSkill(db))
	router.POST("/api/v1/skills/:namespace/:name/community-tags", AddCommunityTag(db))
	router.DELETE("/api/v1/skills/:namespace/:name/community-tags/:tag", RemoveCommunityTag(db))

	postBody := bytes.NewBufferString(`{"tag":"CI Deploy"}`)
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/skills/team/github/community-tags", postBody)
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	router.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("unexpected add status: %d body=%s", postRec.Code, postRec.Body.String())
	}

	var postResp models.Skill
	if err := json.Unmarshal(postRec.Body.Bytes(), &postResp); err != nil {
		t.Fatalf("decode add response failed: %v", err)
	}
	if len(postResp.CommunityTags) != 2 {
		t.Fatalf("unexpected community tags after add: %+v", postResp.CommunityTags)
	}
	if postResp.CommunityTags[0].Tag != "ci-deploy" || postResp.CommunityTags[0].Count != 2 {
		t.Fatalf("unexpected aggregate after add: %+v", postResp.CommunityTags)
	}
	if len(postResp.ViewerTags) != 1 || postResp.ViewerTags[0] != "ci-deploy" {
		t.Fatalf("unexpected viewer tags after add: %+v", postResp.ViewerTags)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/skills/team/github", nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("unexpected get status: %d body=%s", getRec.Code, getRec.Body.String())
	}

	var getResp models.Skill
	if err := json.Unmarshal(getRec.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode get response failed: %v", err)
	}
	if len(getResp.CommunityTags) != 2 || getResp.CommunityTags[1].Tag != "ops" || getResp.CommunityTags[1].Count != 1 {
		t.Fatalf("unexpected aggregate on get: %+v", getResp.CommunityTags)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/skills/team/github/community-tags/ci-deploy", nil)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusOK {
		t.Fatalf("unexpected delete status: %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}

	var deleteResp models.Skill
	if err := json.Unmarshal(deleteRec.Body.Bytes(), &deleteResp); err != nil {
		t.Fatalf("decode delete response failed: %v", err)
	}
	if len(deleteResp.ViewerTags) != 0 {
		t.Fatalf("unexpected viewer tags after delete: %+v", deleteResp.ViewerTags)
	}
	if len(deleteResp.CommunityTags) != 2 || deleteResp.CommunityTags[0].Tag != "ci-deploy" || deleteResp.CommunityTags[0].Count != 1 {
		t.Fatalf("unexpected aggregate after delete: %+v", deleteResp.CommunityTags)
	}
}

func TestUpdateSkillSupportsExplicitDeprecationToggle(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	skill := models.Skill{
		ID:           uuid.New(),
		Namespace:    "team",
		Name:         "reviewer",
		OwnerID:      user.ID,
		Description:  "before",
		License:      "MIT",
		IsPublic:     true,
		IsDeprecated: false,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.PUT("/api/v1/skills/:namespace/:name", UpdateSkill(db))

	body := bytes.NewBufferString(`{"description":"after","tags":["review"],"license":"Apache-2.0","is_public":true,"is_deprecated":true}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/skills/team/reviewer", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var updated models.Skill
	if err := db.First(&updated, "id = ?", skill.ID).Error; err != nil {
		t.Fatalf("reload skill failed: %v", err)
	}
	if !updated.IsDeprecated {
		t.Fatalf("expected skill to be deprecated, got %+v", updated)
	}
	if updated.License != "Apache-2.0" || updated.Description != "after" {
		t.Fatalf("unexpected updated fields: %+v", updated)
	}
}

func TestUpdateSkillDoesNotBumpCatalogVersionForNoOpPublicUpdate(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	skill := models.Skill{
		ID:           uuid.New(),
		Namespace:    "team",
		Name:         "reviewer",
		OwnerID:      user.ID,
		Description:  "before",
		Tags:         models.StringArray{"review", "quality"},
		License:      "MIT",
		IsPublic:     true,
		IsDeprecated: false,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.PUT("/api/v1/skills/:namespace/:name", UpdateSkill(db))

	body := bytes.NewBufferString(`{"description":"before","tags":["review","quality"],"license":"MIT","is_public":true,"is_deprecated":false}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/skills/team/reviewer", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	if got := currentCatalogVersion(t, db); got != 1 {
		t.Fatalf("catalog_version = %d, want 1", got)
	}
}

func TestListSkillsSupportsLicenseFilterAndSort(t *testing.T) {
	db := newTestDatabase(t)
	ownerID := uuid.New()
	now := time.Now()
	skills := []models.Skill{
		{
			ID:            uuid.New(),
			Namespace:     "team",
			Name:          "stable-reviewer",
			OwnerID:       ownerID,
			Description:   "review skill",
			License:       "MIT",
			DownloadCount: 3,
			UpdatedAt:     now.Add(-time.Hour),
			IsPublic:      true,
		},
		{
			ID:            uuid.New(),
			Namespace:     "team",
			Name:          "fresh-reviewer",
			OwnerID:       ownerID,
			Description:   "review skill",
			License:       "MIT",
			DownloadCount: 1,
			UpdatedAt:     now,
			IsPublic:      true,
		},
		{
			ID:            uuid.New(),
			Namespace:     "team",
			Name:          "apache-helper",
			OwnerID:       ownerID,
			Description:   "deploy helper",
			License:       "Apache-2.0",
			DownloadCount: 50,
			UpdatedAt:     now.Add(-2 * time.Hour),
			IsPublic:      true,
		},
	}
	if err := db.Create(&skills).Error; err != nil {
		t.Fatalf("seed skills failed: %v", err)
	}

	router := gin.New()
	router.GET("/api/v1/skills", ListSkills(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills?license=MIT&sort=updated", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Results []models.Skill `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("unexpected result count: %+v", resp.Results)
	}
	if resp.Results[0].Name != "fresh-reviewer" || resp.Results[1].Name != "stable-reviewer" {
		t.Fatalf("unexpected ordering: %+v", resp.Results)
	}
}

func TestListSkillsReturnsPublicDownloadURLs(t *testing.T) {
	db := newTestDatabase(t)
	objStorage := newPublicTestObjectStorage(t)
	ownerID := uuid.New()

	skills := []models.Skill{
		{
			ID:            uuid.New(),
			Namespace:     "team",
			Name:          "alpha",
			OwnerID:       ownerID,
			Description:   "alpha skill",
			IsPublic:      true,
			LatestVersion: "1.0.0",
		},
		{
			ID:            uuid.New(),
			Namespace:     "team",
			Name:          "beta",
			OwnerID:       ownerID,
			Description:   "beta skill",
			IsPublic:      true,
			LatestVersion: "2.0.0",
		},
	}
	if err := db.Create(&skills).Error; err != nil {
		t.Fatalf("seed skills failed: %v", err)
	}

	archives := map[string][]byte{
		"alpha": newSkillArchive(t),
		"beta":  newSkillArchive(t),
	}
	for _, skill := range skills {
		storagePath := fmt.Sprintf("skills/%s/%s/test.zip", skill.Namespace, skill.Name)
		if err := objStorage.Upload(context.Background(), storagePath, bytes.NewReader(archives[skill.Name]), int64(len(archives[skill.Name]))); err != nil {
			t.Fatalf("upload archive for %s failed: %v", skill.Name, err)
		}
		version := models.SkillVersion{
			ID:          uuid.New(),
			SkillID:     skill.ID,
			Version:     skill.LatestVersion,
			StoragePath: storagePath,
			SizeBytes:   int64(len(archives[skill.Name])),
			ScanStatus:  "pass",
			PublishedBy: ownerID,
		}
		if err := db.Create(&version).Error; err != nil {
			t.Fatalf("seed version for %s failed: %v", skill.Name, err)
		}
	}

	router := gin.New()
	router.GET("/api/v1/skills", ListSkills(db, objStorage))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Results []models.Skill `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("unexpected result count: %+v", resp.Results)
	}

	got := map[string]string{}
	for _, skill := range resp.Results {
		got[skill.Name] = skill.DownloadURL
	}
	for _, skill := range skills {
		want, ok := objStorage.PublicURL(fmt.Sprintf("skills/%s/%s/test.zip", skill.Namespace, skill.Name))
		if !ok {
			t.Fatalf("expected public url to resolve for %s", skill.Name)
		}
		if got[skill.Name] != want {
			t.Fatalf("download_url for %s = %q, want %q", skill.Name, got[skill.Name], want)
		}
	}
}

func TestSearchSkillsSupportsTagAndRatingSortSQLiteFallback(t *testing.T) {
	db := newTestDatabase(t)
	ownerID := uuid.New()
	skills := []models.Skill{
		{
			ID:          uuid.New(),
			Namespace:   "team",
			Name:        "review-gold",
			OwnerID:     ownerID,
			Description: "review pull requests",
			Tags:        models.StringArray{"review", "quality"},
			RatingSum:   18,
			RatingCount: 4,
			IsPublic:    true,
		},
		{
			ID:          uuid.New(),
			Namespace:   "team",
			Name:        "review-new",
			OwnerID:     ownerID,
			Description: "review changes quickly",
			Tags:        models.StringArray{"review"},
			RatingSum:   5,
			RatingCount: 2,
			IsPublic:    true,
		},
		{
			ID:          uuid.New(),
			Namespace:   "team",
			Name:        "deploy-new",
			OwnerID:     ownerID,
			Description: "deploy service",
			Tags:        models.StringArray{"deploy"},
			RatingSum:   10,
			RatingCount: 2,
			IsPublic:    true,
		},
	}
	if err := db.Create(&skills).Error; err != nil {
		t.Fatalf("seed skills failed: %v", err)
	}

	router := gin.New()
	router.GET("/api/v1/search", SearchSkills(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=review&tag=review&sort=rating", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Results []models.Skill `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("unexpected search results: %+v", resp.Results)
	}
	if resp.Results[0].Name != "review-gold" {
		t.Fatalf("expected highest rated skill first, got %+v", resp.Results)
	}
}

func TestListAuditLogsReturnsUserEntriesNewestFirst(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "alice", Email: "alice@example.com"}
	otherUser := &models.User{ID: uuid.New(), Username: "bob", Email: "bob@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	if err := db.Create(otherUser).Error; err != nil {
		t.Fatalf("create other user failed: %v", err)
	}

	logs := []models.AuditLog{
		{ID: uuid.New(), UserID: &user.ID, Action: "skill.update", ResourceType: resourceTypeSkill, CreatedAt: time.Now().Add(-time.Hour)},
		{ID: uuid.New(), UserID: &user.ID, Action: "skill.rate", ResourceType: resourceTypeSkill, CreatedAt: time.Now()},
		{ID: uuid.New(), UserID: &otherUser.ID, Action: "skill.delete", ResourceType: resourceTypeSkill, CreatedAt: time.Now().Add(time.Minute)},
	}
	if err := db.Create(&logs).Error; err != nil {
		t.Fatalf("create audit logs failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.GET("/api/v1/user/audit-logs", ListAuditLogs(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/audit-logs?action=skill.rate&page=1&per_page=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Total   int               `json:"total"`
		Results []models.AuditLog `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Total != 1 || len(resp.Results) != 1 || resp.Results[0].Action != "skill.rate" {
		t.Fatalf("unexpected audit log response: %+v", resp)
	}
}

func TestPublishVersionReturnsVersionExistsConflict(t *testing.T) {
	db := newTestDatabase(t)
	mustExec(t, db.DB, `CREATE UNIQUE INDEX idx_skill_version ON skill_versions(skill_id, version)`)

	user := &models.User{ID: uuid.New(), Username: "testuser", Email: "test@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	skill := models.Skill{
		ID:            uuid.New(),
		Namespace:     "skill-home",
		Name:          "skill-home-manager",
		OwnerID:       user.ID,
		Description:   "manager",
		IsPublic:      true,
		LatestVersion: "0.2.1",
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	existingVersion := models.SkillVersion{
		ID:          uuid.New(),
		SkillID:     skill.ID,
		Version:     "0.2.1",
		StoragePath: "skills/skill-home/skill-home-manager/existing.zip",
		SizeBytes:   32,
		ScanStatus:  "pass",
		ScanResult:  models.JSON{"issues": []any{}},
		PublishedBy: user.ID,
		PublishedAt: time.Now(),
	}
	if err := db.Create(&existingVersion).Error; err != nil {
		t.Fatalf("create existing version failed: %v", err)
	}

	storageRoot := t.TempDir()
	objStorage, err := storage.NewObjectStorage(config.StorageConfig{
		Type:      "local",
		LocalPath: storageRoot,
	})
	if err != nil {
		t.Fatalf("NewObjectStorage returned error: %v", err)
	}

	router := newAuthedRouter(user)
	router.POST("/api/v1/skills/:namespace/:name/versions", PublishVersion(db, objStorage, validator.NewScanner()))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("version", "0.2.1"); err != nil {
		t.Fatalf("WriteField returned error: %v", err)
	}
	part, err := writer.CreateFormFile("skill", "skill-home-manager.zip")
	if err != nil {
		t.Fatalf("CreateFormFile returned error: %v", err)
	}
	if _, err := part.Write(mustZipArchive(t, map[string]string{"SKILL.md": "name: skill-home-manager"})); err != nil {
		t.Fatalf("part.Write returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/skills/skill-home/skill-home-manager/versions", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if got := resp["code"]; got != "VERSION_EXISTS" {
		t.Fatalf("unexpected code: %#v body=%s", got, rec.Body.String())
	}

	var versionCount int64
	if err := db.Model(&models.SkillVersion{}).Where("skill_id = ?", skill.ID).Count(&versionCount).Error; err != nil {
		t.Fatalf("count versions failed: %v", err)
	}
	if versionCount != 1 {
		t.Fatalf("expected exactly 1 version after conflict, got %d", versionCount)
	}
	if got := currentCatalogVersion(t, db); got != 1 {
		t.Fatalf("catalog_version = %d, want 1", got)
	}

	var fileCount int
	err = filepath.WalkDir(storageRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			fileCount++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir returned error: %v", err)
	}
	if fileCount != 0 {
		t.Fatalf("expected cleanup after conflict, found %d storage files", fileCount)
	}
}

func TestCreateSkillRecordReturnsAlreadyExistsOnConflict(t *testing.T) {
	db := newTestDatabase(t)
	mustExec(t, db.DB, `CREATE UNIQUE INDEX idx_namespace_name ON skills(namespace, name)`)

	existing := models.Skill{
		ID:        uuid.New(),
		Namespace: "skill-home",
		Name:      "skill-home-manager",
		OwnerID:   uuid.New(),
		IsPublic:  true,
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("create existing skill failed: %v", err)
	}

	candidate := models.Skill{
		Namespace: "skill-home",
		Name:      "skill-home-manager",
		OwnerID:   uuid.New(),
		IsPublic:  true,
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		return createSkillRecord(tx, &candidate)
	})
	if !errors.Is(err, errSkillAlreadyExists) {
		t.Fatalf("expected errSkillAlreadyExists, got %v", err)
	}

	var skillCount int64
	if err := db.Model(&models.Skill{}).Where("namespace = ? AND name = ?", "skill-home", "skill-home-manager").Count(&skillCount).Error; err != nil {
		t.Fatalf("count skills failed: %v", err)
	}
	if skillCount != 1 {
		t.Fatalf("expected exactly 1 skill after conflict, got %d", skillCount)
	}
	if got := currentCatalogVersion(t, db); got != 1 {
		t.Fatalf("catalog_version = %d, want 1", got)
	}
}

func TestCreateUserRecordReturnsEmailExistsOnConflict(t *testing.T) {
	db := newTestDatabase(t)
	mustExec(t, db.DB, `CREATE UNIQUE INDEX idx_users_email ON users(email)`)
	mustExec(t, db.DB, `CREATE UNIQUE INDEX idx_users_username ON users(username)`)

	existing := models.User{
		ID:       uuid.New(),
		Username: "existing-user",
		Email:    "existing@example.com",
		Password: "hash",
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("create existing user failed: %v", err)
	}

	candidate := models.User{
		Username: "new-user",
		Email:    "existing@example.com",
		Password: "hash",
	}

	err := createUserRecord(db.DB, &candidate)
	if !errors.Is(err, errEmailExists) {
		t.Fatalf("expected errEmailExists, got %v", err)
	}
}

func TestCreateUserRecordReturnsUsernameExistsOnConflict(t *testing.T) {
	db := newTestDatabase(t)
	mustExec(t, db.DB, `CREATE UNIQUE INDEX idx_users_email ON users(email)`)
	mustExec(t, db.DB, `CREATE UNIQUE INDEX idx_users_username ON users(username)`)

	existing := models.User{
		ID:       uuid.New(),
		Username: "existing-user",
		Email:    "existing@example.com",
		Password: "hash",
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("create existing user failed: %v", err)
	}

	candidate := models.User{
		Username: "existing-user",
		Email:    "new@example.com",
		Password: "hash",
	}

	err := createUserRecord(db.DB, &candidate)
	if !errors.Is(err, errUsernameExists) {
		t.Fatalf("expected errUsernameExists, got %v", err)
	}
}
