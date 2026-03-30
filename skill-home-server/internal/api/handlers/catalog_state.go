package handlers

import (
	"errors"

	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const catalogStateSingletonID uint = 1

// ensureCatalogState 确保目录状态单例存在。
func ensureCatalogState(tx *gorm.DB) (*models.CatalogState, error) {
	state := &models.CatalogState{}
	if err := tx.First(state, "id = ?", catalogStateSingletonID).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		state = &models.CatalogState{
			ID:             catalogStateSingletonID,
			CatalogVersion: 1,
		}
		if err := tx.Create(state).Error; err != nil {
			return nil, err
		}
	}

	return state, nil
}

// getCatalogState 获取目录状态，必要时创建默认单例。
func getCatalogState(db *storage.Database) (*models.CatalogState, error) {
	var state *models.CatalogState
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		state, err = ensureCatalogState(tx)
		return err
	}); err != nil {
		return nil, err
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
