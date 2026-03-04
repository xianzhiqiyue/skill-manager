package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"golang.org/x/crypto/bcrypt"
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
		"id":         u.ID,
		"username":   u.Username,
		"email":      u.Email,
		"avatar_url": u.AvatarURL,
		"created_at": u.CreatedAt,
	})
}

// GetUserSkills 获取用户技能列表
func GetUserSkills(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		var skills []models.Skill
		if err := db.Where("owner_id = ?", user.ID).Find(&skills).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, skills)
	}
}

// CreateAPIKeyRequest 创建 API Key 请求
type CreateAPIKeyRequest struct {
	Name      string `json:"name"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// CreateAPIKeyResponse 创建 API Key 响应
type CreateAPIKeyResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Key       string    `json:"key"` // 仅创建时返回
	Prefix    string    `json:"prefix"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
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
			Name:      req.Name,
			Prefix:    key[:8],
			ExpiresAt: req.ExpiresAt,
		}

		if err := db.Create(&apiKey).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

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
		if err := db.First(&apiKey, "id = ? AND user_id = ?", keyID, user.ID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "API key not found"})
			return
		}

		if err := db.Delete(&apiKey).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

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
