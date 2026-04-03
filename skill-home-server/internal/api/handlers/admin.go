package handlers

import (
	"strings"

	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
)

func EnsureBootstrapSuperAdmin(db *storage.Database, username string) error {
	if db == nil {
		return nil
	}

	username = strings.TrimSpace(username)
	if username == "" {
		return nil
	}

	return db.Model(&models.User{}).
		Where("username = ?", username).
		Update("is_super_admin", true).
		Error
}
