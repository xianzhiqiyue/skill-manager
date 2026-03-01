package storage

import (
	"fmt"

	"github.com/skill-home/server/internal/config"
	"github.com/skill-home/server/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database 数据库连接
type Database struct {
	*gorm.DB
}

// NewDatabase 创建数据库连接
func NewDatabase(cfg config.DatabaseConfig) (*Database, error) {
	dsn := cfg.GetDSN()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &Database{db}, nil
}

// AutoMigrate 自动迁移数据库
func AutoMigrate(db *Database) error {
	return db.AutoMigrate(
		&models.User{},
		&models.APIKey{},
		&models.Skill{},
		&models.SkillVersion{},
		&models.SkillRating{},
		&models.AuditLog{},
	)
}
