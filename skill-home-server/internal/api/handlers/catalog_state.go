package handlers

import (
	"errors"

	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const catalogStateSingletonID uint = 1

// EnsureCatalogState 确保目录状态单例存在。
func EnsureCatalogState(db *storage.Database) (*models.CatalogState, error) {
	state := &models.CatalogState{}
	if err := db.First(state, "id = ?", catalogStateSingletonID).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		state = &models.CatalogState{
			ID:             catalogStateSingletonID,
			CatalogVersion: 1,
		}
		if err := db.Create(state).Error; err != nil {
			return nil, err
		}
	}

	return state, nil
}

func bumpCatalogVersionTx(tx *gorm.DB) error {
	state := &models.CatalogState{ID: catalogStateSingletonID}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(state).Error; err != nil {
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
