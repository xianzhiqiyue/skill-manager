package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
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
	mustExec(t, db, `CREATE TABLE skill_ratings (
		id TEXT PRIMARY KEY,
		skill_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		rating INTEGER NOT NULL,
		comment TEXT,
		created_at DATETIME,
		updated_at DATETIME
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

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field %s failed: %v", key, err)
		}
	}
	part, err := writer.CreateFormFile("skill", "example.zip")
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
