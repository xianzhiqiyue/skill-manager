package models

import (
	"time"

	"gorm.io/gorm"
)

// CatalogState 目录状态单例
type CatalogState struct {
	ID             uint      `gorm:"primaryKey;autoIncrement:false" json:"-"`
	CatalogVersion int64     `gorm:"not null;default:1" json:"catalog_version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// BeforeCreate 创建前钩子
func (s *CatalogState) BeforeCreate(tx *gorm.DB) error {
	if s.ID == 0 {
		s.ID = 1
	}
	if s.CatalogVersion == 0 {
		s.CatalogVersion = 1
	}
	return nil
}
