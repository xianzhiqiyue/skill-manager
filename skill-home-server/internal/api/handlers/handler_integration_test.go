package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
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
