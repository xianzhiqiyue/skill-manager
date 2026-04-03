package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// GetCurrentUser 获取当前用户
func GetCurrentUser(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Not authenticated"})
		return
	}

	u := user.(*models.User)
	c.JSON(http.StatusOK, gin.H{
		"id":             u.ID,
		"username":       u.Username,
		"email":          u.Email,
		"avatar_url":     u.AvatarURL,
		"is_active":      u.IsActive,
		"is_super_admin": u.IsSuperAdmin,
		"created_at":     u.CreatedAt,
	})
}

// APIKeySummaryResponse API Key 列表项
type APIKeySummaryResponse struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// GetUserSkills 获取用户技能列表
func GetUserSkills(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		var skills []models.Skill
		query := db.Model(&models.Skill{})
		if !user.IsSuperAdmin {
			query = query.Where("owner_id = ?", user.ID)
		}
		if err := query.Find(&skills).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, skills)
	}
}

// ListAPIKeys 获取当前用户的 API Key 列表
func ListAPIKeys(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		var apiKeys []models.APIKey
		if err := db.
			Where("user_id = ?", user.ID).
			Order("created_at DESC").
			Find(&apiKeys).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		response := make([]APIKeySummaryResponse, 0, len(apiKeys))
		for _, apiKey := range apiKeys {
			response = append(response, APIKeySummaryResponse{
				ID:         apiKey.ID,
				Name:       apiKey.Name,
				Prefix:     apiKey.Prefix,
				LastUsedAt: apiKey.LastUsedAt,
				ExpiresAt:  apiKey.ExpiresAt,
				CreatedAt:  apiKey.CreatedAt,
			})
		}

		c.JSON(http.StatusOK, response)
	}
}

// CreateAPIKeyRequest 创建 API Key 请求
type CreateAPIKeyRequest struct {
	Name      string     `json:"name"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// CreateAPIKeyResponse 创建 API Key 响应
type CreateAPIKeyResponse struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Key       string     `json:"key"` // 仅创建时返回
	Prefix    string     `json:"prefix"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// CreateAPIKey 创建 API Key
func CreateAPIKey(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateAPIKeyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}

		user := c.MustGet("user").(*models.User)
		name := strings.TrimSpace(req.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "name is required"})
			return
		}

		// 生成 API Key
		key, err := generateAPIKey()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Failed to generate API key"})
			return
		}
		keyHash, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Failed to generate API key"})
			return
		}

		apiKey := models.APIKey{
			UserID:    user.ID,
			KeyHash:   string(keyHash),
			Name:      name,
			Prefix:    key[:8],
			ExpiresAt: req.ExpiresAt,
		}

		if err := db.Create(&apiKey).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		writeAuditLog(db, c, &user.ID, "api_key.create", resourceTypeAPIKey, &apiKey.ID, models.JSON{
			"name":       apiKey.Name,
			"prefix":     apiKey.Prefix,
			"expires_at": apiKey.ExpiresAt,
		})

		c.JSON(http.StatusCreated, CreateAPIKeyResponse{
			ID:        apiKey.ID,
			Name:      apiKey.Name,
			Key:       key,
			Prefix:    apiKey.Prefix,
			ExpiresAt: apiKey.ExpiresAt,
			CreatedAt: apiKey.CreatedAt,
		})
	}
}

// RevokeAPIKey 撤销 API Key
func RevokeAPIKey(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		keyID := c.Param("id")
		user := c.MustGet("user").(*models.User)

		var apiKey models.APIKey
		query := db.Model(&models.APIKey{})
		if !user.IsSuperAdmin {
			query = query.Where("user_id = ?", user.ID)
		}
		if err := query.First(&apiKey, "id = ?", keyID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "API key not found"})
			return
		}

		if err := db.Delete(&apiKey).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		writeAuditLog(db, c, &user.ID, "api_key.revoke", resourceTypeAPIKey, &apiKey.ID, models.JSON{
			"name":   apiKey.Name,
			"prefix": apiKey.Prefix,
		})

		c.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
	}
}

// generateAPIKey 生成随机 API Key
func generateAPIKey() (string, error) {
	// 生成 32 字节随机数据
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// 使用 base64 URL 编码
	return "sk_" + base64.RawURLEncoding.EncodeToString(b), nil
}

type AdminUserResponse struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	IsActive     bool      `json:"is_active"`
	IsSuperAdmin bool      `json:"is_super_admin"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AdminUpdateUserRequest struct {
	Password     *string `json:"password,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
	IsSuperAdmin *bool   `json:"is_super_admin,omitempty"`
}

func ListUsers(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		if !user.IsSuperAdmin {
			c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "Super admin required"})
			return
		}

		var users []models.User
		if err := db.Order("created_at ASC").Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		response := make([]AdminUserResponse, 0, len(users))
		for _, item := range users {
			response = append(response, AdminUserResponse{
				ID:           item.ID,
				Username:     item.Username,
				Email:        item.Email,
				AvatarURL:    item.AvatarURL,
				IsActive:     item.IsActive,
				IsSuperAdmin: item.IsSuperAdmin,
				CreatedAt:    item.CreatedAt,
				UpdatedAt:    item.UpdatedAt,
			})
		}
		c.JSON(http.StatusOK, response)
	}
}

func UpdateUserByAdmin(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor := c.MustGet("user").(*models.User)
		if !actor.IsSuperAdmin {
			c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "Super admin required"})
			return
		}

		var req AdminUpdateUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}

		var target models.User
		if err := db.First(&target, "id = ?", c.Param("id")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "User not found"})
			return
		}

		updates := map[string]interface{}{}
		if req.IsActive != nil {
			updates["is_active"] = *req.IsActive
			target.IsActive = *req.IsActive
		}
		if req.IsSuperAdmin != nil {
			updates["is_super_admin"] = *req.IsSuperAdmin
			target.IsSuperAdmin = *req.IsSuperAdmin
		}
		if req.Password != nil {
			password := strings.TrimSpace(*req.Password)
			if len(password) < 6 {
				c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "password must be at least 6 characters"})
				return
			}
			passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Failed to process password"})
				return
			}
			updates["password"] = string(passwordHash)
		}

		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "No updates provided"})
			return
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&target).Updates(updates).Error; err != nil {
				return err
			}
			metadata := models.JSON{
				"target_user_id": target.ID.String(),
				"username":       target.Username,
			}
			if req.IsActive != nil {
				metadata["is_active"] = *req.IsActive
			}
			if req.IsSuperAdmin != nil {
				metadata["is_super_admin"] = *req.IsSuperAdmin
			}
			if req.Password != nil {
				metadata["password_reset"] = true
			}
			return writeAuditLogTx(tx, c, &actor.ID, "admin.user.update", resourceTypeUser, &target.ID, metadata)
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		if err := db.First(&target, "id = ?", target.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, AdminUserResponse{
			ID:           target.ID,
			Username:     target.Username,
			Email:        target.Email,
			AvatarURL:    target.AvatarURL,
			IsActive:     target.IsActive,
			IsSuperAdmin: target.IsSuperAdmin,
			CreatedAt:    target.CreatedAt,
			UpdatedAt:    target.UpdatedAt,
		})
	}
}
