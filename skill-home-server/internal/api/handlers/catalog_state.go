package handlers

import (
	"errors"
	"time"

	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const catalogStateSingletonID uint = 1
const catalogStateInitMaxAttempts = 5

// ensureCatalogState 确保目录状态单例存在。
func ensureCatalogState(tx *gorm.DB) (*models.CatalogState, error) {
	var lastInitErr error
	for attempt := 0; attempt < catalogStateInitMaxAttempts; attempt++ {
		state, err := loadCatalogState(tx)
		if err == nil {
			return state, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(defaultCatalogState()).Error; err != nil {
			lastInitErr = err
		}

		state, err = loadCatalogState(tx)
		if err == nil {
			return state, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		time.Sleep(time.Millisecond * time.Duration(attempt+1))
	}

	if lastInitErr != nil {
		return nil, lastInitErr
	}
	return nil, gorm.ErrRecordNotFound
}

// getCatalogState 获取目录状态，必要时创建默认单例。
func getCatalogState(db *storage.Database) (*models.CatalogState, error) {
	return ensureCatalogState(db.DB)
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
