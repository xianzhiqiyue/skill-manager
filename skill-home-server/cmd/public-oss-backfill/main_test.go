package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skill-home/server/internal/config"
	"github.com/skill-home/server/internal/storage"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRunBackfillDryRunReportsSuccessWithoutWriting(t *testing.T) {
	db := newBackfillTestDatabase(t)
	source := newBackfillTestObjectStorage(t)
	target := newBackfillTestObjectStorage(t)
	storagePath := "skills/team/reviewer/1.0.0.zip"

	insertPublicVersion(t, db, "team", "reviewer", "1.0.0", storagePath)
	mustUploadObject(t, source, storagePath, "archive")

	var output bytes.Buffer
	stats, err := runBackfill(context.Background(), db, source, target, backfillOptions{
		DryRun: true,
		Stdout: &output,
	})
	if err != nil {
		t.Fatalf("runBackfill() error = %v", err)
	}

	if stats.Succeeded != 1 || stats.Skipped != 0 || stats.Failed != 0 {
		t.Fatalf("stats = %+v, want succeeded=1 skipped=0 failed=0", stats)
	}

	exists, err := target.Exists(context.Background(), storagePath)
	if err != nil {
		t.Fatalf("target Exists() error = %v", err)
	}
	if exists {
		t.Fatal("dry-run wrote object to target storage")
	}

	if got := output.String(); got == "" || !containsAll(got, "dry-run", "team/reviewer@1.0.0") {
		t.Fatalf("output = %q, want dry-run record", got)
	}
}

func TestRunBackfillSkipsExistingTargetObject(t *testing.T) {
	db := newBackfillTestDatabase(t)
	target := newBackfillTestObjectStorage(t)
	storagePath := "skills/team/reviewer/1.0.0.zip"

	insertPublicVersion(t, db, "team", "reviewer", "1.0.0", storagePath)
	mustUploadObject(t, target, storagePath, "already-there")

	stats, err := runBackfill(context.Background(), db, nil, target, backfillOptions{
		Stdout: io.Discard,
	})
	if err != nil {
		t.Fatalf("runBackfill() error = %v", err)
	}

	if stats.Succeeded != 0 || stats.Skipped != 1 || stats.Failed != 0 {
		t.Fatalf("stats = %+v, want succeeded=0 skipped=1 failed=0", stats)
	}
}

func TestRunBackfillFailsWhenSourceObjectMissing(t *testing.T) {
	db := newBackfillTestDatabase(t)
	source := newBackfillTestObjectStorage(t)
	target := newBackfillTestObjectStorage(t)
	storagePath := "skills/team/reviewer/1.0.0.zip"

	insertPublicVersion(t, db, "team", "reviewer", "1.0.0", storagePath)

	stats, err := runBackfill(context.Background(), db, source, target, backfillOptions{
		Stdout: io.Discard,
	})
	if err != nil {
		t.Fatalf("runBackfill() error = %v", err)
	}

	if stats.Succeeded != 0 || stats.Skipped != 0 || stats.Failed != 1 {
		t.Fatalf("stats = %+v, want succeeded=0 skipped=0 failed=1", stats)
	}
}

func TestRunBackfillCopiesMissingTargetObject(t *testing.T) {
	db := newBackfillTestDatabase(t)
	source := newBackfillTestObjectStorage(t)
	target := newBackfillTestObjectStorage(t)
	storagePath := "skills/team/reviewer/1.0.0.zip"

	insertPublicVersion(t, db, "team", "reviewer", "1.0.0", storagePath)
	mustUploadObject(t, source, storagePath, "archive")

	stats, err := runBackfill(context.Background(), db, source, target, backfillOptions{
		Stdout: io.Discard,
	})
	if err != nil {
		t.Fatalf("runBackfill() error = %v", err)
	}

	if stats.Succeeded != 1 || stats.Skipped != 0 || stats.Failed != 0 {
		t.Fatalf("stats = %+v, want succeeded=1 skipped=0 failed=0", stats)
	}

	reader, err := target.Download(context.Background(), storagePath)
	if err != nil {
		t.Fatalf("target Download() error = %v", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if got, want := string(content), "archive"; got != want {
		t.Fatalf("copied content = %q, want %q", got, want)
	}
}

func TestLoadPublicSkillVersionsFiltersNonPublicDeletedAndEmptyPaths(t *testing.T) {
	db := newBackfillTestDatabase(t)

	insertSkillVersionRow(t, db, skillVersionFixture{
		Namespace:   "team",
		Name:        "public-active",
		Version:     "1.0.0",
		StoragePath: "skills/team/public-active/1.0.0.zip",
		IsPublic:    true,
	})
	insertSkillVersionRow(t, db, skillVersionFixture{
		Namespace:   "team",
		Name:        "private-skill",
		Version:     "1.0.0",
		StoragePath: "skills/team/private-skill/1.0.0.zip",
		IsPublic:    false,
	})
	insertSkillVersionRow(t, db, skillVersionFixture{
		Namespace:      "team",
		Name:           "deleted-skill",
		Version:        "1.0.0",
		StoragePath:    "skills/team/deleted-skill/1.0.0.zip",
		IsPublic:       true,
		SkillDeletedAt: ptrTime(time.Now().UTC()),
	})
	insertSkillVersionRow(t, db, skillVersionFixture{
		Namespace:        "team",
		Name:             "deleted-version",
		Version:          "1.0.0",
		StoragePath:      "skills/team/deleted-version/1.0.0.zip",
		IsPublic:         true,
		VersionDeletedAt: ptrTime(time.Now().UTC()),
	})
	insertSkillVersionRow(t, db, skillVersionFixture{
		Namespace:   "team",
		Name:        "empty-path",
		Version:     "1.0.0",
		StoragePath: "",
		IsPublic:    true,
	})

	rows, err := loadPublicSkillVersions(db)
	if err != nil {
		t.Fatalf("loadPublicSkillVersions() error = %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("loadPublicSkillVersions() returned %d rows, want 1: %+v", len(rows), rows)
	}
	if got := rows[0]; got.Namespace != "team" || got.Name != "public-active" || got.Version != "1.0.0" || got.StoragePath != "skills/team/public-active/1.0.0.zip" {
		t.Fatalf("loadPublicSkillVersions() row = %+v, want only active public version", got)
	}
}

func newBackfillTestDatabase(t *testing.T) *storage.Database {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	database := &storage.Database{DB: db}
	mustExec(t, db, `CREATE TABLE skills (
		id TEXT PRIMARY KEY,
		namespace TEXT NOT NULL,
		name TEXT NOT NULL,
		is_public NUMERIC DEFAULT 1,
		deleted_at DATETIME
	)`)
	mustExec(t, db, `CREATE TABLE skill_versions (
		id TEXT PRIMARY KEY,
		skill_id TEXT NOT NULL,
		version TEXT NOT NULL,
		storage_path TEXT,
		deleted_at DATETIME
	)`)
	return database
}

func newBackfillTestObjectStorage(t *testing.T) *storage.ObjectStorage {
	t.Helper()

	objStorage, err := storage.NewObjectStorage(config.StorageConfig{
		Type:      "local",
		LocalPath: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewObjectStorage() error = %v", err)
	}
	return objStorage
}

func insertPublicVersion(t *testing.T, db *storage.Database, namespace, name, version, storagePath string) {
	t.Helper()

	insertSkillVersionRow(t, db, skillVersionFixture{
		Namespace:   namespace,
		Name:        name,
		Version:     version,
		StoragePath: storagePath,
		IsPublic:    true,
	})
}

type skillVersionFixture struct {
	Namespace        string
	Name             string
	Version          string
	StoragePath      string
	IsPublic         bool
	SkillDeletedAt   *time.Time
	VersionDeletedAt *time.Time
}

func insertSkillVersionRow(t *testing.T, db *storage.Database, fixture skillVersionFixture) {
	t.Helper()

	skillID := uuid.NewString()
	versionID := uuid.NewString()

	if err := db.Exec(
		`INSERT INTO skills (id, namespace, name, is_public, deleted_at) VALUES (?, ?, ?, ?, ?)`,
		skillID, fixture.Namespace, fixture.Name, fixture.IsPublic, fixture.SkillDeletedAt,
	).Error; err != nil {
		t.Fatalf("insert skill failed: %v", err)
	}

	if err := db.Exec(
		`INSERT INTO skill_versions (id, skill_id, version, storage_path, deleted_at) VALUES (?, ?, ?, ?, ?)`,
		versionID, skillID, fixture.Version, fixture.StoragePath, fixture.VersionDeletedAt,
	).Error; err != nil {
		t.Fatalf("insert skill version failed: %v", err)
	}
}

func mustUploadObject(t *testing.T, objStorage *storage.ObjectStorage, key, content string) {
	t.Helper()

	if err := objStorage.Upload(context.Background(), key, bytes.NewReader([]byte(content)), int64(len(content))); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
}

func mustExec(t *testing.T, db *gorm.DB, sql string) {
	t.Helper()
	if err := db.Exec(sql).Error; err != nil {
		t.Fatalf("exec sql failed: %v\nsql=%s", err, sql)
	}
}

func containsAll(value string, substrings ...string) bool {
	for _, substring := range substrings {
		if !bytes.Contains([]byte(value), []byte(substring)) {
			return false
		}
	}
	return true
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
