package handlers

import (
	"errors"

	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"gorm.io/gorm"
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
