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
		"id":              u.ID,
		"username":        u.Username,
		"display_name_zh": u.DisplayNameZh,
		"email":           u.Email,
		"avatar_url":      u.AvatarURL,
		"is_active":       u.IsActive,
		"is_admin":        u.IsAdmin,
		"is_super_admin":  u.IsSuperAdmin,
		"created_at":      u.CreatedAt,
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
		query := db.Model(&models.Skill{}).Preload("Owner")
		if user.IsSuperAdmin {
			query = query.Order("is_recommended DESC").Order("updated_at DESC").Order("name ASC")
		} else if user.IsAdmin {
			query = query.
				Where("is_public = ? OR owner_id = ?", true, user.ID).
				Order("is_recommended DESC").
				Order("updated_at DESC").
				Order("name ASC")
		} else {
			query = query.Where("owner_id = ?", user.ID)
		}
		if err := query.Find(&skills).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		populateSkillsComputedFields(skills)
		if err := populateSkillOwnerFields(db, skills); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, skills)
	}
}

type UserStatsResponse struct {
	UserID             uuid.UUID `json:"user_id"`
	Username           string    `json:"username"`
	DisplayNameZh      string    `json:"display_name_zh,omitempty"`
	SkillCount         int64     `json:"skill_count"`
	PublicSkillCount   int64     `json:"public_skill_count"`
	TotalLikeCount     int64     `json:"total_like_count"`
	TotalRatingCount   int64     `json:"total_rating_count"`
	AverageRating      float64   `json:"average_rating"`
	TotalDownloadCount int64     `json:"total_download_count"`
	TotalInstallCount  int64     `json:"total_install_count"`
}

type UpdateCurrentUserProfileRequest struct {
	DisplayNameZh *string `json:"display_name_zh,omitempty"`
	AvatarURL     *string `json:"avatar_url,omitempty"`
}

type UpdateCurrentUserPasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

func GetCurrentUserStats(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		stats, err := buildUserStats(db, user, false)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, stats)
	}
}

func GetPublicUserStats(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := normalizeNamespace(c.Param("username"))
		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "username is required"})
			return
		}

		var user models.User
		if err := db.First(&user, "username = ?", username).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "User not found"})
			return
		}

		stats, err := buildUserStats(db, &user, true)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, stats)
	}
}

func buildUserStats(db *storage.Database, user *models.User, publicOnly bool) (UserStatsResponse, error) {
	stats := UserStatsResponse{
		UserID:        user.ID,
		Username:      user.Username,
		DisplayNameZh: user.DisplayNameZh,
	}

	base := func() *gorm.DB {
		query := db.Model(&models.Skill{}).Where("owner_id = ?", user.ID)
		if publicOnly {
			query = query.Where("is_public = ?", true)
		}
		return query
	}

	if err := base().Count(&stats.SkillCount).Error; err != nil {
		return stats, err
	}
	if err := db.Model(&models.Skill{}).
		Where("owner_id = ? AND is_public = ?", user.ID, true).
		Count(&stats.PublicSkillCount).Error; err != nil {
		return stats, err
	}

	var aggregate struct {
		TotalLikeCount     int64 `gorm:"column:total_like_count"`
		TotalRatingCount   int64 `gorm:"column:total_rating_count"`
		TotalRatingSum     int64 `gorm:"column:total_rating_sum"`
		TotalDownloadCount int64 `gorm:"column:total_download_count"`
		TotalInstallCount  int64 `gorm:"column:total_install_count"`
	}
	if err := base().
		Select(`
			COALESCE(SUM(like_count), 0) AS total_like_count,
			COALESCE(SUM(rating_count), 0) AS total_rating_count,
			COALESCE(SUM(rating_sum), 0) AS total_rating_sum,
			COALESCE(SUM(download_count), 0) AS total_download_count,
			COALESCE(SUM(install_count), 0) AS total_install_count`).
		Scan(&aggregate).Error; err != nil {
		return stats, err
	}

	stats.TotalLikeCount = aggregate.TotalLikeCount
	stats.TotalRatingCount = aggregate.TotalRatingCount
	stats.TotalDownloadCount = aggregate.TotalDownloadCount
	stats.TotalInstallCount = aggregate.TotalInstallCount
	if aggregate.TotalRatingCount > 0 {
		stats.AverageRating = float64(aggregate.TotalRatingSum) / float64(aggregate.TotalRatingCount)
	}

	return stats, nil
}

func UpdateCurrentUserProfile(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		var req UpdateCurrentUserProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}

		updates := map[string]interface{}{}
		metadata := models.JSON{}
		if req.DisplayNameZh != nil {
			displayNameZh := strings.TrimSpace(*req.DisplayNameZh)
			if displayNameZh == "" {
				c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "display_name_zh is required"})
				return
			}
			if len(displayNameZh) > 64 {
				c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "display_name_zh is too long"})
				return
			}
			updates["display_name_zh"] = displayNameZh
			metadata["display_name_zh"] = displayNameZh
		}
		if req.AvatarURL != nil {
			avatarURL := strings.TrimSpace(*req.AvatarURL)
			if len(avatarURL) > 500 {
				c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "avatar_url is too long"})
				return
			}
			updates["avatar_url"] = avatarURL
			metadata["avatar_url_updated"] = true
		}
		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "No updates provided"})
			return
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(user).Updates(updates).Error; err != nil {
				return err
			}
			return writeAuditLogTx(tx, c, &user.ID, "user.profile.update", resourceTypeUser, &user.ID, metadata)
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		var updated models.User
		if err := db.First(&updated, "id = ?", user.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":              updated.ID,
			"username":        updated.Username,
			"display_name_zh": updated.DisplayNameZh,
			"email":           updated.Email,
			"avatar_url":      updated.AvatarURL,
			"is_active":       updated.IsActive,
			"is_admin":        updated.IsAdmin,
			"is_super_admin":  updated.IsSuperAdmin,
			"created_at":      updated.CreatedAt,
		})
	}
}

func UpdateCurrentUserPassword(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		var req UpdateCurrentUserPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}

		var current models.User
		if err := db.First(&current, "id = ?", user.ID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "User not found"})
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(current.Password), []byte(req.CurrentPassword)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_CREDENTIALS", "message": "Current password is incorrect"})
			return
		}

		password := strings.TrimSpace(req.NewPassword)
		if len(password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "new_password must be at least 6 characters"})
			return
		}
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Failed to process password"})
			return
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&current).Update("password", string(passwordHash)).Error; err != nil {
				return err
			}
			return writeAuditLogTx(tx, c, &user.ID, "user.password.update", resourceTypeUser, &user.ID, models.JSON{
				"password_changed": true,
			})
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Password updated"})
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
	ID            uuid.UUID `json:"id"`
	Username      string    `json:"username"`
	DisplayNameZh string    `json:"display_name_zh,omitempty"`
	Email         string    `json:"email"`
	AvatarURL     string    `json:"avatar_url,omitempty"`
	Role          string    `json:"role"`
	IsActive      bool      `json:"is_active"`
	IsAdmin       bool      `json:"is_admin"`
	IsSuperAdmin  bool      `json:"is_super_admin"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type AdminUserListResponse struct {
	Total   int64               `json:"total"`
	Page    int                 `json:"page"`
	PerPage int                 `json:"per_page"`
	Results []AdminUserResponse `json:"results"`
}

type AdminUpdateUserRequest struct {
	Password     *string `json:"password,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
	IsAdmin      *bool   `json:"is_admin,omitempty"`
	IsSuperAdmin *bool   `json:"is_super_admin,omitempty"`
}

func ListUsers(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		if !user.IsSuperAdmin {
			c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "Super admin required"})
			return
		}

		page, perPage := parsePagination(c.DefaultQuery("page", "1"), c.DefaultQuery("per_page", "20"))
		queryText := strings.TrimSpace(c.Query("q"))
		role := strings.TrimSpace(c.Query("role"))
		status := strings.TrimSpace(c.Query("status"))

		query := db.Model(&models.User{})
		if queryText != "" {
			pattern := "%" + strings.ToLower(queryText) + "%"
			query = query.Where(
				"LOWER(username) LIKE ? OR LOWER(email) LIKE ? OR LOWER(display_name_zh) LIKE ?",
				pattern,
				pattern,
				pattern,
			)
		}
		switch role {
		case "", "all":
		case "super_admin":
			query = query.Where("is_super_admin = ?", true)
		case "admin":
			query = query.Where("is_admin = ? AND is_super_admin = ?", true, false)
		case "member":
			query = query.Where("is_admin = ? AND is_super_admin = ?", false, false)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "invalid role filter"})
			return
		}
		switch status {
		case "", "all":
		case "active":
			query = query.Where("is_active = ?", true)
		case "inactive":
			query = query.Where("is_active = ?", false)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "invalid status filter"})
			return
		}

		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		var users []models.User
		if err := query.Order("created_at ASC").Limit(perPage).Offset((page - 1) * perPage).Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		response := make([]AdminUserResponse, 0, len(users))
		for _, item := range users {
			response = append(response, buildAdminUserResponse(item))
		}
		c.JSON(http.StatusOK, AdminUserListResponse{
			Total:   total,
			Page:    page,
			PerPage: perPage,
			Results: response,
		})
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

		if req.IsActive != nil && !*req.IsActive && target.ID == actor.ID {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "cannot deactivate yourself"})
			return
		}
		if req.IsSuperAdmin != nil && !*req.IsSuperAdmin && target.IsSuperAdmin {
			hasOther, err := hasOtherActiveSuperAdmin(db.DB, target.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
				return
			}
			if !hasOther {
				c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "cannot remove the last super admin"})
				return
			}
		}
		if req.IsActive != nil && !*req.IsActive && target.IsSuperAdmin {
			hasOther, err := hasOtherActiveSuperAdmin(db.DB, target.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
				return
			}
			if !hasOther {
				c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "cannot deactivate the last active super admin"})
				return
			}
		}

		updates := map[string]interface{}{}
		if req.IsActive != nil {
			updates["is_active"] = *req.IsActive
			target.IsActive = *req.IsActive
		}
		if req.IsAdmin != nil {
			updates["is_admin"] = *req.IsAdmin
			target.IsAdmin = *req.IsAdmin
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
			if req.IsAdmin != nil {
				metadata["is_admin"] = *req.IsAdmin
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

		c.JSON(http.StatusOK, buildAdminUserResponse(target))
	}
}

func buildAdminUserResponse(user models.User) AdminUserResponse {
	return AdminUserResponse{
		ID:            user.ID,
		Username:      user.Username,
		DisplayNameZh: user.DisplayNameZh,
		Email:         user.Email,
		AvatarURL:     user.AvatarURL,
		Role:          adminUserRole(user),
		IsActive:      user.IsActive,
		IsAdmin:       user.IsAdmin,
		IsSuperAdmin:  user.IsSuperAdmin,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
	}
}

func adminUserRole(user models.User) string {
	if user.IsSuperAdmin {
		return "super_admin"
	}
	if user.IsAdmin {
		return "admin"
	}
	return "member"
}

func hasOtherActiveSuperAdmin(db *gorm.DB, userID uuid.UUID) (bool, error) {
	var count int64
	if err := db.Model(&models.User{}).
		Where("id <> ? AND is_active = ? AND is_super_admin = ?", userID, true, true).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
