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
	"golang.org/x/crypto/bcrypt"
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
	)`)
	mustExec(t, db, `CREATE TABLE skills (
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
		is_deprecated NUMERIC DEFAULT 0,
		is_recommended NUMERIC DEFAULT 0,
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
	mustExec(t, db, `CREATE TABLE skill_likes (
		id TEXT PRIMARY KEY,
		skill_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		created_at DATETIME
	)`)
	mustExec(t, db, `CREATE UNIQUE INDEX idx_skill_like_user ON skill_likes(skill_id, user_id)`)
	mustExec(t, db, `CREATE TABLE skill_install_events (
		id TEXT PRIMARY KEY,
		skill_id TEXT NOT NULL,
		user_id TEXT,
		version TEXT,
		target TEXT,
		install_mode TEXT,
		client_version TEXT,
		created_at DATETIME
	)`)
	mustExec(t, db, `CREATE TABLE skill_share_events (
		id TEXT PRIMARY KEY,
		skill_id TEXT NOT NULL,
		user_id TEXT,
		channel TEXT,
		created_at DATETIME
	)`)
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
		"category":  "development",
		"tags":      "review",
		"version":   "1.0.0",
	}, newSkillArchive(t))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestEnsureBootstrapSuperAdminMarksConfiguredUser(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "zhuyuxiao314", Email: "zhuyue314@gmail.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	if err := EnsureBootstrapSuperAdmin(db, "zhuyuxiao314"); err != nil {
		t.Fatalf("EnsureBootstrapSuperAdmin returned error: %v", err)
	}

	var updated models.User
	if err := db.First(&updated, "id = ?", user.ID).Error; err != nil {
		t.Fatalf("load user failed: %v", err)
	}
	if !updated.IsSuperAdmin {
		t.Fatal("expected user to be promoted to super admin")
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

func TestPublicSkillDetailReturnsCredentialsFromLatestVersionManifest(t *testing.T) {
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
		StoragePath: "skills/team/github/old.zip",
		Manifest: models.JSON{
			"metadata": map[string]any{
				"openclaw": map[string]any{
					"credentials": []map[string]any{
						{
							"id":    "legacy_key",
							"env":   "LEGACY_KEY",
							"label": "Legacy Key",
						},
					},
				},
			},
		},
		SizeBytes:   64,
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
		StoragePath: "skills/team/github/latest.zip",
		Manifest: models.JSON{
			"metadata": map[string]any{
				"openclaw": map[string]any{
					"credentials": []map[string]any{
						{
							"id":          "openai_api_key",
							"env":         "OPENAI_API_KEY",
							"label":       "OpenAI API Key",
							"description": "Used to access OpenAI",
							"secret":      true,
							"required":    true,
							"input":       "password",
							"help_url":    "https://platform.openai.com/api-keys",
							"group":       "llm_provider",
						},
					},
				},
			},
		},
		SizeBytes:   96,
		ScanStatus:  "pass",
		PublishedBy: user.ID,
	}
	if err := db.Create(&latestVersion).Error; err != nil {
		t.Fatalf("create latest version failed: %v", err)
	}

	router := gin.New()
	router.GET("/api/v1/skills/:namespace/:name", middleware.OptionalAuth(db), GetSkill(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills/team/github", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Credentials []map[string]any `json:"credentials"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if len(resp.Credentials) != 1 {
		t.Fatalf("expected 1 credential, got %#v", resp.Credentials)
	}
	if got := resp.Credentials[0]["env"]; got != "OPENAI_API_KEY" {
		t.Fatalf("credential env = %#v, want OPENAI_API_KEY", got)
	}
	if got := resp.Credentials[0]["label"]; got != "OpenAI API Key" {
		t.Fatalf("credential label = %#v, want OpenAI API Key", got)
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
		"category":  "development",
		"tags":      "review",
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

func TestCreateSkillPersistsManifestCredentials(t *testing.T) {
	db := newTestDatabase(t)
	objStorage := newTestObjectStorage(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.POST("/api/v1/skills", CreateSkill(db, objStorage, validator.NewScanner()))

	req, _ := newCreateSkillRequest(t, map[string]string{
		"namespace": "team",
		"name":      "github",
		"category":  "integration",
		"tags":      "api",
		"version":   "1.0.0",
	}, mustZipArchive(t, map[string]string{
		"SKILL.md": `---
name: github
version: 1.0.0
description: GitHub skill
metadata:
  openclaw:
    credentials:
      - id: openai_api_key
        env: OPENAI_API_KEY
        label: OpenAI API Key
        description: Used to access OpenAI
        secret: true
        required: true
        input: password
        help_url: https://platform.openai.com/api-keys
        group: llm_provider
---
body
`,
	}))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var skill models.Skill
	if err := scopeNamespaceName(db.DB, "team", "github").First(&skill).Error; err != nil {
		t.Fatalf("load created skill failed: %v", err)
	}

	var version models.SkillVersion
	if err := db.Where("skill_id = ? AND version = ?", skill.ID, "1.0.0").First(&version).Error; err != nil {
		t.Fatalf("load created version failed: %v", err)
	}

	metadata, ok := version.Manifest["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("manifest metadata missing: %#v", version.Manifest)
	}
	openclaw, ok := metadata["openclaw"].(map[string]any)
	if !ok {
		t.Fatalf("manifest openclaw missing: %#v", metadata)
	}
	credentials, ok := openclaw["credentials"].([]any)
	if !ok || len(credentials) != 1 {
		t.Fatalf("manifest credentials = %#v, want 1 item", openclaw["credentials"])
	}
	credential, ok := credentials[0].(map[string]any)
	if !ok {
		t.Fatalf("credential item = %#v", credentials[0])
	}
	if got := credential["env"]; got != "OPENAI_API_KEY" {
		t.Fatalf("credential env = %#v, want OPENAI_API_KEY", got)
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
		"category":  "development",
		"tags":      "workflow",
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
		"category":  "integration",
		"tags":      "api",
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
			"category":  "development",
			"tags":      "review",
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

		body := bytes.NewBufferString(`{"description":"after","category":"development","tags":["review"],"license":"Apache-2.0","is_public":true}`)
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

	t.Run("super admin can publish another user's version", func(t *testing.T) {
		db := newTestDatabase(t)
		objStorage := newTestObjectStorage(t)
		owner := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
		admin := &models.User{ID: uuid.New(), Username: "admin", Email: "admin@example.com", IsSuperAdmin: true}
		if err := db.Create(owner).Error; err != nil {
			t.Fatalf("create owner failed: %v", err)
		}
		if err := db.Create(admin).Error; err != nil {
			t.Fatalf("create admin failed: %v", err)
		}
		skill := models.Skill{
			ID:            uuid.New(),
			Namespace:     "team",
			Name:          "github",
			OwnerID:       owner.ID,
			IsPublic:      true,
			LatestVersion: "1.0.0",
		}
		if err := db.Create(&skill).Error; err != nil {
			t.Fatalf("create skill failed: %v", err)
		}

		router := newAuthedRouter(admin)
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

		body := bytes.NewBufferString(`{"description":"after","category":"development","tags":["review"],"license":"Apache-2.0","is_public":false}`)
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

		body := bytes.NewBufferString(`{"description":"after","category":"development","tags":["review"],"license":"Apache-2.0","is_public":true}`)
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

func TestSuperAdminCanUpdateAndDeleteAnotherUsersSkill(t *testing.T) {
	db := newTestDatabase(t)
	owner := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	admin := &models.User{ID: uuid.New(), Username: "admin", Email: "admin@example.com", IsSuperAdmin: true}
	if err := db.Create(owner).Error; err != nil {
		t.Fatalf("create owner failed: %v", err)
	}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("create admin failed: %v", err)
	}

	skill := models.Skill{
		ID:          uuid.New(),
		Namespace:   "team",
		Name:        "github",
		OwnerID:     owner.ID,
		Description: "before",
		Category:    "development",
		Tags:        models.StringArray{"review"},
		License:     "MIT",
		IsPublic:    true,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	router := newAuthedRouter(admin)
	router.PUT("/api/v1/skills/:namespace/:name", UpdateSkill(db))
	router.DELETE("/api/v1/skills/:namespace/:name", DeleteSkill(db, newTestObjectStorage(t)))

	updateBody := bytes.NewBufferString(`{"description":"after","category":"development","tags":["review"],"license":"Apache-2.0","is_public":true}`)
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/skills/team/github", updateBody)
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status: %d body=%s", updateRec.Code, updateRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/skills/team/github", nil)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status: %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestSuperAdminCanManageUsers(t *testing.T) {
	db := newTestDatabase(t)
	admin := &models.User{ID: uuid.New(), Username: "admin", Email: "admin@example.com", IsSuperAdmin: true}
	target := &models.User{ID: uuid.New(), Username: "member", Email: "member@example.com", IsActive: true}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("create admin failed: %v", err)
	}
	if err := db.Create(target).Error; err != nil {
		t.Fatalf("create target failed: %v", err)
	}

	router := newAuthedRouter(admin)
	router.GET("/api/v1/admin/users", ListUsers(db))
	router.PUT("/api/v1/admin/users/:id", UpdateUserByAdmin(db))

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status: %d body=%s", listRec.Code, listRec.Body.String())
	}
	var listResp AdminUserListResponse
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list response failed: %v", err)
	}
	if listResp.Total != 2 || len(listResp.Results) != 2 {
		t.Fatalf("unexpected user list response: %+v", listResp)
	}

	updateBody := bytes.NewBufferString(`{"is_admin":true,"is_super_admin":true,"is_active":false,"password":"new-password"}`)
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/"+target.ID.String(), updateBody)
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status: %d body=%s", updateRec.Code, updateRec.Body.String())
	}

	var updated models.User
	if err := db.First(&updated, "id = ?", target.ID).Error; err != nil {
		t.Fatalf("reload target failed: %v", err)
	}
	if !updated.IsSuperAdmin {
		t.Fatal("expected target to become super admin")
	}
	if !updated.IsAdmin {
		t.Fatal("expected target to become admin")
	}
	if updated.IsActive {
		t.Fatal("expected target to become inactive")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte("new-password")); err != nil {
		t.Fatalf("expected password to be updated: %v", err)
	}

	var updateResp AdminUserResponse
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updateResp); err != nil {
		t.Fatalf("decode update response failed: %v", err)
	}
	if updateResp.Role != "super_admin" {
		t.Fatalf("role = %q, want super_admin", updateResp.Role)
	}
}

func TestListUsersSupportsFiltersAndPagination(t *testing.T) {
	db := newTestDatabase(t)
	users := []*models.User{
		{ID: uuid.New(), Username: "root", DisplayNameZh: "超管", Email: "root@example.com", IsActive: true, IsSuperAdmin: true},
		{ID: uuid.New(), Username: "catalog-admin", DisplayNameZh: "目录管理员", Email: "admin@example.com", IsActive: true, IsAdmin: true},
		{ID: uuid.New(), Username: "alice", DisplayNameZh: "爱丽丝", Email: "alice@example.com", IsActive: true},
		{ID: uuid.New(), Username: "bob", DisplayNameZh: "鲍勃", Email: "bob@example.com", IsActive: false},
	}
	if err := db.Create(users).Error; err != nil {
		t.Fatalf("seed users failed: %v", err)
	}
	if err := db.Model(users[3]).Update("is_active", false).Error; err != nil {
		t.Fatalf("set inactive user failed: %v", err)
	}

	router := newAuthedRouter(users[0])
	router.GET("/api/v1/admin/users", ListUsers(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?role=member&status=inactive&q=bob&page=1&per_page=1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list status: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp AdminUserListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Total != 1 || resp.Page != 1 || resp.PerPage != 1 || len(resp.Results) != 1 {
		t.Fatalf("unexpected pagination response: %+v", resp)
	}
	if resp.Results[0].Username != "bob" || resp.Results[0].Role != "member" || resp.Results[0].IsActive {
		t.Fatalf("unexpected filtered user: %+v", resp.Results[0])
	}
}

func TestUpdateUserByAdminProtectsSuperAdminAccess(t *testing.T) {
	db := newTestDatabase(t)
	admin := &models.User{ID: uuid.New(), Username: "root", Email: "root@example.com", IsActive: true, IsSuperAdmin: true}
	target := &models.User{ID: uuid.New(), Username: "member", Email: "member@example.com", IsActive: true}
	if err := db.Create([]*models.User{admin, target}).Error; err != nil {
		t.Fatalf("seed users failed: %v", err)
	}

	router := newAuthedRouter(admin)
	router.PUT("/api/v1/admin/users/:id", UpdateUserByAdmin(db))

	selfDeactivateReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/"+admin.ID.String(), bytes.NewBufferString(`{"is_active":false}`))
	selfDeactivateReq.Header.Set("Content-Type", "application/json")
	selfDeactivateRec := httptest.NewRecorder()
	router.ServeHTTP(selfDeactivateRec, selfDeactivateReq)
	if selfDeactivateRec.Code != http.StatusBadRequest {
		t.Fatalf("expected self deactivation 400, got %d body=%s", selfDeactivateRec.Code, selfDeactivateRec.Body.String())
	}

	removeLastReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/"+admin.ID.String(), bytes.NewBufferString(`{"is_super_admin":false}`))
	removeLastReq.Header.Set("Content-Type", "application/json")
	removeLastRec := httptest.NewRecorder()
	router.ServeHTTP(removeLastRec, removeLastReq)
	if removeLastRec.Code != http.StatusBadRequest {
		t.Fatalf("expected last super admin removal 400, got %d body=%s", removeLastRec.Code, removeLastRec.Body.String())
	}

	updateTargetReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/"+target.ID.String(), bytes.NewBufferString(`{"is_admin":true}`))
	updateTargetReq.Header.Set("Content-Type", "application/json")
	updateTargetRec := httptest.NewRecorder()
	router.ServeHTTP(updateTargetRec, updateTargetReq)
	if updateTargetRec.Code != http.StatusOK {
		t.Fatalf("expected target update 200, got %d body=%s", updateTargetRec.Code, updateTargetRec.Body.String())
	}
}

func TestCurrentUserCanUpdateProfileAndPassword(t *testing.T) {
	db := newTestDatabase(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password failed: %v", err)
	}
	user := &models.User{
		ID:            uuid.New(),
		Username:      "alice",
		DisplayNameZh: "旧名字",
		Email:         "alice@example.com",
		Password:      string(hash),
		IsActive:      true,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.PUT("/api/v1/user/profile", UpdateCurrentUserProfile(db))
	router.PUT("/api/v1/user/password", UpdateCurrentUserPassword(db))

	profileReq := httptest.NewRequest(http.MethodPut, "/api/v1/user/profile", bytes.NewBufferString(`{"display_name_zh":"新名字","avatar_url":"https://example.com/avatar.png"}`))
	profileReq.Header.Set("Content-Type", "application/json")
	profileRec := httptest.NewRecorder()
	router.ServeHTTP(profileRec, profileReq)
	if profileRec.Code != http.StatusOK {
		t.Fatalf("profile status: %d body=%s", profileRec.Code, profileRec.Body.String())
	}

	passwordReq := httptest.NewRequest(http.MethodPut, "/api/v1/user/password", bytes.NewBufferString(`{"current_password":"old-password","new_password":"new-password"}`))
	passwordReq.Header.Set("Content-Type", "application/json")
	passwordRec := httptest.NewRecorder()
	router.ServeHTTP(passwordRec, passwordReq)
	if passwordRec.Code != http.StatusOK {
		t.Fatalf("password status: %d body=%s", passwordRec.Code, passwordRec.Body.String())
	}

	var updated models.User
	if err := db.First(&updated, "id = ?", user.ID).Error; err != nil {
		t.Fatalf("reload user failed: %v", err)
	}
	if updated.DisplayNameZh != "新名字" || updated.AvatarURL != "https://example.com/avatar.png" {
		t.Fatalf("unexpected profile: %+v", updated)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte("new-password")); err != nil {
		t.Fatalf("expected new password hash: %v", err)
	}
}

func TestSuperAdminCanListGlobalAuditLogs(t *testing.T) {
	db := newTestDatabase(t)
	admin := &models.User{ID: uuid.New(), Username: "root", Email: "root@example.com", IsActive: true, IsSuperAdmin: true}
	target := &models.User{ID: uuid.New(), Username: "member", Email: "member@example.com", IsActive: true}
	if err := db.Create([]*models.User{admin, target}).Error; err != nil {
		t.Fatalf("seed users failed: %v", err)
	}

	router := newAuthedRouter(admin)
	router.PUT("/api/v1/admin/users/:id", UpdateUserByAdmin(db))
	router.GET("/api/v1/admin/audit-logs", ListAdminAuditLogs(db))

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/"+target.ID.String(), bytes.NewBufferString(`{"is_admin":true}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status: %d body=%s", updateRec.Code, updateRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?action=admin.user.update", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("audit list status: %d body=%s", listRec.Code, listRec.Body.String())
	}

	var resp struct {
		Total   int64             `json:"total"`
		Results []models.AuditLog `json:"results"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode audit response failed: %v", err)
	}
	if resp.Total != 1 || len(resp.Results) != 1 || resp.Results[0].Action != "admin.user.update" {
		t.Fatalf("unexpected audit response: %+v", resp)
	}
}

func TestAdminCanUpdateSkillRecommendation(t *testing.T) {
	db := newTestDatabase(t)
	admin := &models.User{ID: uuid.New(), Username: "catalog-admin", Email: "admin@example.com", IsAdmin: true}
	owner := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("create admin failed: %v", err)
	}
	if err := db.Create(owner).Error; err != nil {
		t.Fatalf("create owner failed: %v", err)
	}

	skill := &models.Skill{
		ID:            uuid.New(),
		Namespace:     "team",
		Name:          "reviewer",
		OwnerID:       owner.ID,
		Description:   "review skill",
		IsPublic:      true,
		IsRecommended: false,
	}
	if err := db.Create(skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	router := newAuthedRouter(admin)
	router.PATCH("/api/v1/admin/skills/:namespace/:name/recommendation", UpdateSkillRecommendation(db))

	body := bytes.NewBufferString(`{"is_recommended":true}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/skills/team/reviewer/recommendation", body)
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
	if !updated.IsRecommended {
		t.Fatal("expected skill to become recommended")
	}
	if got := currentCatalogVersion(t, db); got != 2 {
		t.Fatalf("catalog_version = %d, want 2", got)
	}
}

func TestNonAdminCannotUpdateSkillRecommendation(t *testing.T) {
	db := newTestDatabase(t)
	owner := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(owner).Error; err != nil {
		t.Fatalf("create owner failed: %v", err)
	}
	skill := &models.Skill{
		ID:          uuid.New(),
		Namespace:   "team",
		Name:        "reviewer",
		OwnerID:     owner.ID,
		Description: "review skill",
		IsPublic:    true,
	}
	if err := db.Create(skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	router := newAuthedRouter(owner)
	router.PATCH("/api/v1/admin/skills/:namespace/:name/recommendation", UpdateSkillRecommendation(db))

	body := bytes.NewBufferString(`{"is_recommended":true}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/skills/team/reviewer/recommendation", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
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
		"category":  "development",
		"tags":      "review",
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
			"category":  "integration",
			"tags":      "api",
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
			ID:            uuid.New(),
			Namespace:     "team",
			Name:          "private-github",
			OwnerID:       user.ID,
			Description:   "before",
			IsPublic:      true,
			IsRecommended: true,
		}
		if err := db.Create(&skill).Error; err != nil {
			t.Fatalf("create skill failed: %v", err)
		}
		if err := db.Model(&skill).Update("is_public", false).Error; err != nil {
			t.Fatalf("set skill private failed: %v", err)
		}

		router := newAuthedRouter(user)
		router.PUT("/api/v1/skills/:namespace/:name", UpdateSkill(db))

		body := bytes.NewBufferString(`{"description":"after","category":"development","tags":["review"],"license":"Apache-2.0","is_public":false}`)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/skills/team/private-github", body)
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
		if updated.IsRecommended {
			t.Fatal("expected private skill update to clear recommendation")
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
		"category":  "development",
		"tags":      "review",
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

func TestPublishVersionPersistsManifestCredentials(t *testing.T) {
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

	archive := mustZipArchive(t, map[string]string{
		"SKILL.md": `---
name: github
version: 1.1.0
description: GitHub skill
metadata:
  openclaw:
    credentials:
      - id: openai_api_key
        env: OPENAI_API_KEY
        label: OpenAI API Key
        description: Used to access OpenAI
        secret: true
        required: true
        input: password
        help_url: https://platform.openai.com/api-keys
        group: llm_provider
---
body
`,
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("version", "1.1.0"); err != nil {
		t.Fatalf("WriteField version returned error: %v", err)
	}
	part, err := writer.CreateFormFile("skill", "github.zip")
	if err != nil {
		t.Fatalf("CreateFormFile returned error: %v", err)
	}
	if _, err := part.Write(archive); err != nil {
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

	var version models.SkillVersion
	if err := db.Where("skill_id = ? AND version = ?", skill.ID, "1.1.0").First(&version).Error; err != nil {
		t.Fatalf("load created version failed: %v", err)
	}

	metadata, ok := version.Manifest["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("manifest metadata missing: %#v", version.Manifest)
	}
	openclaw, ok := metadata["openclaw"].(map[string]any)
	if !ok {
		t.Fatalf("manifest openclaw missing: %#v", metadata)
	}
	credentials, ok := openclaw["credentials"].([]any)
	if !ok || len(credentials) != 1 {
		t.Fatalf("manifest credentials = %#v, want 1 item", openclaw["credentials"])
	}
	credential, ok := credentials[0].(map[string]any)
	if !ok {
		t.Fatalf("credential item = %#v", credentials[0])
	}
	if got := credential["env"]; got != "OPENAI_API_KEY" {
		t.Fatalf("credential env = %#v, want OPENAI_API_KEY", got)
	}
	if got := credential["group"]; got != "llm_provider" {
		t.Fatalf("credential group = %#v, want llm_provider", got)
	}
}

func TestPublishVersionDerivesManifestRequiresFromCredentials(t *testing.T) {
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

	archive := mustZipArchive(t, map[string]string{
		"SKILL.md": `---
name: github
version: 1.1.0
description: GitHub skill
metadata:
  openclaw:
    credentials:
      - id: openai_api_key
        env: OPENAI_API_KEY
      - id: anthropic_api_key
        env: ANTHROPIC_API_KEY
---
body
`,
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("version", "1.1.0"); err != nil {
		t.Fatalf("WriteField version returned error: %v", err)
	}
	part, err := writer.CreateFormFile("skill", "github.zip")
	if err != nil {
		t.Fatalf("CreateFormFile returned error: %v", err)
	}
	if _, err := part.Write(archive); err != nil {
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

	var version models.SkillVersion
	if err := db.Where("skill_id = ? AND version = ?", skill.ID, "1.1.0").First(&version).Error; err != nil {
		t.Fatalf("load created version failed: %v", err)
	}

	requires, ok := version.Manifest["requires"].([]any)
	if !ok || len(requires) != 2 {
		t.Fatalf("manifest requires = %#v, want 2 envs", version.Manifest["requires"])
	}
	if requires[0] != "OPENAI_API_KEY" || requires[1] != "ANTHROPIC_API_KEY" {
		t.Fatalf("manifest requires = %#v, want derived envs", requires)
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
	if err := writer.WriteField("category", "productivity"); err != nil {
		t.Fatalf("WriteField category returned error: %v", err)
	}
	if err := writer.WriteField("tags", "workflow"); err != nil {
		t.Fatalf("WriteField tags returned error: %v", err)
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
		"category":    "ops",
		"version":     "1.0.0",
		"license":     "MIT",
		"tags":        "deploy, ci",
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

	if skill.Category != "ops" {
		t.Fatalf("unexpected category: %+v", skill.Category)
	}
	if len(skill.Tags) != 2 || skill.Tags[0] != "deployment" || skill.Tags[1] != "ci-cd" {
		t.Fatalf("unexpected tags: %+v", skill.Tags)
	}
}

func TestCreateSkillRejectsMissingCategory(t *testing.T) {
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
		"tags":        "deployment",
		"is_public":   "true",
	}, newSkillArchive(t))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "category") {
		t.Fatalf("unexpected response body: %s", rec.Body.String())
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
		Category:     "development",
		License:      "MIT",
		IsPublic:     true,
		IsDeprecated: false,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.PUT("/api/v1/skills/:namespace/:name", UpdateSkill(db))

	body := bytes.NewBufferString(`{"description":"after","category":"development","tags":["review"],"license":"Apache-2.0","is_public":true,"is_deprecated":true}`)
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
		Category:     "development",
		Tags:         models.StringArray{"review", "analysis"},
		License:      "MIT",
		IsPublic:     true,
		IsDeprecated: false,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.PUT("/api/v1/skills/:namespace/:name", UpdateSkill(db))

	body := bytes.NewBufferString(`{"description":"before","category":"development","tags":["review","analysis"],"license":"MIT","is_public":true,"is_deprecated":false}`)
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

func TestUpdateSkillAllowsDescriptionOnlyUpdateForLegacySkillWithoutTaxonomy(t *testing.T) {
	db := newTestDatabase(t)
	user := &models.User{ID: uuid.New(), Username: "owner", Email: "owner@example.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	skill := models.Skill{
		ID:            uuid.New(),
		Namespace:     "team",
		Name:          "legacy-reviewer",
		OwnerID:       user.ID,
		Description:   "Review code quickly.",
		DescriptionZh: "",
		License:       "MIT",
		IsPublic:      true,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.PUT("/api/v1/skills/:namespace/:name", UpdateSkill(db))

	body := bytes.NewBufferString(`{"description":"Review code quickly.","description_zh":"快速审查代码。","category":"","tags":[],"license":"MIT","is_public":true,"is_deprecated":false}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/skills/team/legacy-reviewer", body)
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
	if updated.DescriptionZh != "快速审查代码。" {
		t.Fatalf("unexpected description zh: %+v", updated.DescriptionZh)
	}
}

func TestUpdateSkillRejectsInvalidCategory(t *testing.T) {
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
		Category:     "development",
		Tags:         models.StringArray{"review"},
		License:      "MIT",
		IsPublic:     true,
		IsDeprecated: false,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	router := newAuthedRouter(user)
	router.PUT("/api/v1/skills/:namespace/:name", UpdateSkill(db))

	body := bytes.NewBufferString(`{"description":"after","category":"general","tags":["review"],"license":"Apache-2.0","is_public":true,"is_deprecated":false}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/skills/team/reviewer", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "category") {
		t.Fatalf("unexpected response body: %s", rec.Body.String())
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
			IsRecommended: true,
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
	if resp.Results[0].Name != "stable-reviewer" || resp.Results[1].Name != "fresh-reviewer" {
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
			ID:            uuid.New(),
			Namespace:     "team",
			Name:          "review-new",
			OwnerID:       ownerID,
			Description:   "review changes quickly",
			Tags:          models.StringArray{"review"},
			RatingSum:     5,
			RatingCount:   2,
			IsRecommended: true,
			IsPublic:      true,
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
	if resp.Results[0].Name != "review-new" {
		t.Fatalf("expected recommended skill first, got %+v", resp.Results)
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

func TestRegisterRequiresDisplayNameZh(t *testing.T) {
	db := newTestDatabase(t)

	router := gin.New()
	router.POST("/api/v1/auth/register", Register(db))

	body := strings.NewReader(`{"username":"testuser","email":"test@example.com","password":"secret123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRegisterPersistsDisplayNameZh(t *testing.T) {
	t.Setenv("SKILL_HOME_AUTH_JWT_SECRET", "test-secret")
	if err := config.Load(); err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	db := newTestDatabase(t)
	mustExec(t, db.DB, `CREATE UNIQUE INDEX idx_users_email ON users(email)`)
	mustExec(t, db.DB, `CREATE UNIQUE INDEX idx_users_username ON users(username)`)

	router := gin.New()
	router.POST("/api/v1/auth/register", Register(db))

	body := strings.NewReader(`{"username":"testuser","display_name_zh":"测试用户","email":"test@example.com","password":"secret123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.User.DisplayNameZh != "测试用户" {
		t.Fatalf("display_name_zh = %q, want 测试用户", resp.User.DisplayNameZh)
	}
	if resp.Token == "" {
		t.Fatal("expected auth token")
	}

	var user models.User
	if err := db.First(&user, "username = ?", "testuser").Error; err != nil {
		t.Fatalf("load registered user failed: %v", err)
	}
	if user.DisplayNameZh != "测试用户" {
		t.Fatalf("stored display_name_zh = %q, want 测试用户", user.DisplayNameZh)
	}
}

func TestLikeSkillUpdatesCountAndViewerState(t *testing.T) {
	db := newTestDatabase(t)
	owner := &models.User{ID: uuid.New(), Username: "owner", DisplayNameZh: "拥有者", Email: "owner@example.com"}
	viewer := &models.User{ID: uuid.New(), Username: "viewer", DisplayNameZh: "访问者", Email: "viewer@example.com"}
	if err := db.Create([]*models.User{owner, viewer}).Error; err != nil {
		t.Fatalf("create users failed: %v", err)
	}

	skill := models.Skill{
		ID:          uuid.New(),
		Namespace:   "team",
		Name:        "github",
		OwnerID:     owner.ID,
		Description: "public skill",
		IsPublic:    true,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	router := newAuthedRouter(viewer)
	router.POST("/api/v1/skills/:namespace/:name/like", LikeSkill(db))
	router.DELETE("/api/v1/skills/:namespace/:name/like", UnlikeSkill(db))

	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/skills/team/github/like", nil)
	postRec := httptest.NewRecorder()
	router.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", postRec.Code, postRec.Body.String())
	}

	var liked models.Skill
	if err := json.Unmarshal(postRec.Body.Bytes(), &liked); err != nil {
		t.Fatalf("decode liked skill failed: %v", err)
	}
	if liked.LikeCount != 1 || !liked.ViewerLiked {
		t.Fatalf("unexpected liked state: like_count=%d viewer_liked=%v", liked.LikeCount, liked.ViewerLiked)
	}
	if liked.OwnerUsername != "owner" || liked.OwnerDisplayNameZh != "拥有者" {
		t.Fatalf("unexpected owner fields: %+v", liked)
	}

	duplicateReq := httptest.NewRequest(http.MethodPost, "/api/v1/skills/team/github/like", nil)
	duplicateRec := httptest.NewRecorder()
	router.ServeHTTP(duplicateRec, duplicateReq)
	if duplicateRec.Code != http.StatusOK {
		t.Fatalf("expected duplicate like 200, got %d body=%s", duplicateRec.Code, duplicateRec.Body.String())
	}
	var duplicate models.Skill
	if err := json.Unmarshal(duplicateRec.Body.Bytes(), &duplicate); err != nil {
		t.Fatalf("decode duplicate like failed: %v", err)
	}
	if duplicate.LikeCount != 1 || !duplicate.ViewerLiked {
		t.Fatalf("duplicate like state = like_count=%d viewer_liked=%v, want 1/true", duplicate.LikeCount, duplicate.ViewerLiked)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/skills/team/github/like", nil)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected unlike 200, got %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}

	var unliked models.Skill
	if err := json.Unmarshal(deleteRec.Body.Bytes(), &unliked); err != nil {
		t.Fatalf("decode unliked skill failed: %v", err)
	}
	if unliked.LikeCount != 0 || unliked.ViewerLiked {
		t.Fatalf("unexpected unliked state: like_count=%d viewer_liked=%v", unliked.LikeCount, unliked.ViewerLiked)
	}
}

func TestRecordInstallEventIncrementsStats(t *testing.T) {
	db := newTestDatabase(t)
	owner := &models.User{ID: uuid.New(), Username: "owner", DisplayNameZh: "拥有者", Email: "owner@example.com"}
	viewer := &models.User{ID: uuid.New(), Username: "viewer", Email: "viewer@example.com"}
	if err := db.Create([]*models.User{owner, viewer}).Error; err != nil {
		t.Fatalf("create users failed: %v", err)
	}

	skill := models.Skill{
		ID:            uuid.New(),
		Namespace:     "team",
		Name:          "github",
		OwnerID:       owner.ID,
		Description:   "public skill",
		DownloadCount: 3,
		LikeCount:     2,
		RatingSum:     9,
		RatingCount:   2,
		IsPublic:      true,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill failed: %v", err)
	}

	router := newAuthedRouter(viewer)
	router.POST("/api/v1/skills/:namespace/:name/install-events", RecordInstallEvent(db))

	body := strings.NewReader(`{"version":"1.0.0","target":"codex","install_mode":"mirror","client_version":"1.2.3"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/skills/team/github/install-events", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		InstallCount int64        `json:"install_count"`
		Skill        models.Skill `json:"skill"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.InstallCount != 1 || resp.Skill.InstallCount != 1 {
		t.Fatalf("install_count = %d/%d, want 1", resp.InstallCount, resp.Skill.InstallCount)
	}

	var event models.SkillInstallEvent
	if err := db.First(&event, "skill_id = ?", skill.ID).Error; err != nil {
		t.Fatalf("load install event failed: %v", err)
	}
	if event.UserID == nil || *event.UserID != viewer.ID || event.Target != "codex" {
		t.Fatalf("unexpected install event: %+v", event)
	}

	stats, err := buildUserStats(db, owner, false)
	if err != nil {
		t.Fatalf("build stats failed: %v", err)
	}
	if stats.SkillCount != 1 || stats.TotalInstallCount != 1 || stats.TotalLikeCount != 2 || stats.TotalDownloadCount != 3 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if stats.AverageRating != 4.5 {
		t.Fatalf("average_rating = %v, want 4.5", stats.AverageRating)
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
