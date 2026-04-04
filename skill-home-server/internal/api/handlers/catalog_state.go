package handlers

import (
	"errors"
	"strings"
	"time"

	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const catalogStateSingletonID uint = 1
const catalogStateInitMaxAttempts = 10

// ensureCatalogState 确保目录状态单例存在。
func ensureCatalogState(tx *gorm.DB) (*models.CatalogState, error) {
	var lastInitErr error
	for attempt := 0; attempt < catalogStateInitMaxAttempts; attempt++ {
		state, err := loadCatalogState(tx)
		if err == nil {
			return state, nil
		}
		if isCatalogStateTransientLockError(err) {
			lastInitErr = err
			time.Sleep(5 * time.Millisecond * time.Duration(attempt+1))
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(defaultCatalogState()).Error; err != nil {
			if isCatalogStateTransientLockError(err) {
				lastInitErr = err
				time.Sleep(5 * time.Millisecond * time.Duration(attempt+1))
				continue
			}
			lastInitErr = err
		}

		state, err = loadCatalogState(tx)
		if err == nil {
			return state, nil
		}
		if isCatalogStateTransientLockError(err) {
			lastInitErr = err
			time.Sleep(5 * time.Millisecond * time.Duration(attempt+1))
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		time.Sleep(5 * time.Millisecond * time.Duration(attempt+1))
	}

	if lastInitErr != nil {
		return nil, lastInitErr
	}
	return nil, gorm.ErrRecordNotFound
}

// getCatalogState 获取目录状态，必要时创建默认单例。
func getCatalogState(db *storage.Database) (*models.CatalogState, error) {
	var state *models.CatalogState
	err := db.Transaction(func(tx *gorm.DB) error {
		current, err := ensureCatalogState(tx)
		if err != nil {
			return err
		}

		latestMutationAt, err := latestPublicCatalogMutationAtTx(tx)
		if err != nil {
			return err
		}
		if latestMutationAt.IsZero() || !latestMutationAt.After(current.UpdatedAt) {
			state = current
			return nil
		}

		if err := reconcileCatalogStateTx(tx, latestMutationAt); err != nil {
			return err
		}

		current, err = loadCatalogState(tx)
		if err != nil {
			return err
		}
		state = current
		return nil
	})
	if err != nil {
		return nil, err
	}
	return state, nil
}

func loadCatalogState(tx *gorm.DB) (*models.CatalogState, error) {
	state := &models.CatalogState{}
	if err := tx.First(state, "id = ?", catalogStateSingletonID).Error; err != nil {
		return nil, err
	}
	return state, nil
}

func defaultCatalogState() *models.CatalogState {
	return &models.CatalogState{
		ID:             catalogStateSingletonID,
		CatalogVersion: 1,
	}
}

func isCatalogStateTransientLockError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database table is locked") ||
		strings.Contains(msg, "database schema is locked")
}

func bumpCatalogVersionTx(tx *gorm.DB) error {
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(defaultCatalogState()).Error; err != nil {
		return err
	}

	result := tx.Model(&models.CatalogState{}).
		Where("id = ?", catalogStateSingletonID).
		Update("catalog_version", gorm.Expr("catalog_version + 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func reconcileCatalogStateTx(tx *gorm.DB, latestMutationAt time.Time) error {
	if latestMutationAt.IsZero() {
		return nil
	}

	result := tx.Model(&models.CatalogState{}).
		Where("id = ? AND updated_at < ?", catalogStateSingletonID, latestMutationAt).
		UpdateColumns(map[string]any{
			"catalog_version": gorm.Expr("catalog_version + 1"),
			"updated_at":      latestMutationAt,
		})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func latestPublicCatalogMutationAtTx(tx *gorm.DB) (time.Time, error) {
	latestSkillAt, err := latestPublicSkillMutationAtTx(tx)
	if err != nil {
		return time.Time{}, err
	}

	latestVersionAt, err := latestPublicSkillVersionMutationAtTx(tx)
	if err != nil {
		return time.Time{}, err
	}

	if latestVersionAt.After(latestSkillAt) {
		return latestVersionAt, nil
	}
	return latestSkillAt, nil
}

func latestPublicSkillMutationAtTx(tx *gorm.DB) (time.Time, error) {
	var skill models.Skill
	if err := tx.Unscoped().
		Model(&models.Skill{}).
		Where("is_public = ?", true).
		Order("COALESCE(deleted_at, updated_at, created_at) DESC").
		Limit(1).
		Take(&skill).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	return latestSkillTimestamp(skill.CreatedAt, skill.UpdatedAt, skill.DeletedAt), nil
}

func latestPublicSkillVersionMutationAtTx(tx *gorm.DB) (time.Time, error) {
	var version models.SkillVersion
	if err := tx.Unscoped().
		Model(&models.SkillVersion{}).
		Joins("JOIN skills ON skills.id = skill_versions.skill_id").
		Where("skills.is_public = ?", true).
		Order("COALESCE(skill_versions.deleted_at, skill_versions.published_at, skill_versions.created_at) DESC").
		Limit(1).
		Take(&version).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}

	latest := version.CreatedAt
	if version.PublishedAt.After(latest) {
		latest = version.PublishedAt
	}
	if version.DeletedAt.Valid && version.DeletedAt.Time.After(latest) {
		latest = version.DeletedAt.Time
	}
	return latest, nil
}

func latestSkillTimestamp(createdAt, updatedAt time.Time, deletedAt gorm.DeletedAt) time.Time {
	latest := createdAt
	if updatedAt.After(latest) {
		latest = updatedAt
	}
	if deletedAt.Valid && deletedAt.Time.After(latest) {
		latest = deletedAt.Time
	}
	return latest
}
