package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	mustExec(t, db, `CREATE UNIQUE INDEX idx_skill_user_rating ON skill_ratings(skill_id, user_id)`)
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
