package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"gorm.io/gorm"
)

const (
	resourceTypeUser    = "user"
	resourceTypeSkill   = "skill"
	resourceTypeVersion = "skill_version"
	resourceTypeAPIKey  = "api_key"
)

func writeAuditLog(db *storage.Database, c *gin.Context, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata models.JSON) {
	if db == nil {
		return
	}
	_ = writeAuditLogTx(db.DB, c, userID, action, resourceType, resourceID, metadata)
}

func writeAuditLogTx(tx *gorm.DB, c *gin.Context, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata models.JSON) error {
	if tx == nil {
		return nil
	}

	log := models.AuditLog{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     metadata,
		CreatedAt:    time.Now(),
	}
	if c != nil {
		log.IPAddress = c.ClientIP()
		if c.Request != nil {
			log.UserAgent = c.Request.UserAgent()
		}
	}

	return tx.Create(&log).Error
}

// ListAuditLogs 获取用户审计日志
func ListAuditLogs(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		page, perPage := parsePagination(c.DefaultQuery("page", "1"), c.DefaultQuery("per_page", "20"))
		action := c.Query("action")

		query := db.Model(&models.AuditLog{}).Where("user_id = ?", user.ID)
		if action != "" {
			query = query.Where("action = ?", action)
		}

		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		var logs []models.AuditLog
		if err := query.Order("created_at DESC").Limit(perPage).Offset((page - 1) * perPage).Find(&logs).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total":    total,
			"page":     page,
			"per_page": perPage,
			"results":  logs,
		})
	}
}

// ListAdminAuditLogs 获取全局审计日志，仅 super admin 可访问。
func ListAdminAuditLogs(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		if !user.IsSuperAdmin {
			c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "Super admin required"})
			return
		}

		page, perPage := parsePagination(c.DefaultQuery("page", "1"), c.DefaultQuery("per_page", "20"))
		action := c.Query("action")
		userID := c.Query("user_id")

		query := db.Model(&models.AuditLog{})
		if action != "" {
			query = query.Where("action = ?", action)
		}
		if userID != "" {
			if _, err := uuid.Parse(userID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "invalid user_id"})
				return
			}
			query = query.Where("user_id = ?", userID)
		}

		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		var logs []models.AuditLog
		if err := query.Order("created_at DESC").Limit(perPage).Offset((page - 1) * perPage).Find(&logs).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total":    total,
			"page":     page,
			"per_page": perPage,
			"results":  logs,
		})
	}
}
