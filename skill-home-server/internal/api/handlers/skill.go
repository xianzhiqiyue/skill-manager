package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"github.com/skill-home/server/pkg/validator"
	"gorm.io/gorm"
)

// ListSkills 列出技能
func ListSkills(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var skills []models.Skill
		query := db.Where("is_public = ?", true)

		// 分页
		page := c.DefaultQuery("page", "1")
		perPage := c.DefaultQuery("per_page", "20")

		// 搜索
		if q := c.Query("q"); q != "" {
			query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+q+"%", "%"+q+"%")
		}

		// 标签筛选
		if tag := c.Query("tag"); tag != "" {
			query = query.Where("? = ANY(tags)", tag)
		}

		var total int64
		query.Model(&models.Skill{}).Count(&total)

		if err := query.Order("download_count DESC").Find(&skills).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total":    total,
			"page":     page,
			"per_page": perPage,
			"results":  skills,
		})
	}
}

// GetSkill 获取技能详情
func GetSkill(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := c.Param("namespace")
		name := c.Param("name")

		var skill models.Skill
		if err := db.Preload("Versions").First(&skill, "namespace = ? AND name = ?", namespace, name).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "Skill not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		// 检查权限
		if !skill.IsPublic {
			user, exists := c.Get("user")
			if !exists || user.(*models.User).ID != skill.OwnerID {
				c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "Access denied"})
				return
			}
		}

		c.JSON(http.StatusOK, skill)
	}
}

// SearchSkills 搜索技能
func SearchSkills(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := c.Query("q")
		if q == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "Query is required"})
			return
		}

		var skills []models.Skill
		query := db.Where("is_public = ?", true).
			Where("name ILIKE ? OR description ILIKE ?", "%"+q+"%", "%"+q+"%")

		if err := query.Order("download_count DESC").Find(&skills).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total":   len(skills),
			"results": skills,
		})
	}
}

// ListVersions 列出技能版本
func ListVersions(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := c.Param("namespace")
		name := c.Param("name")

		var skill models.Skill
		if err := db.First(&skill, "namespace = ? AND name = ?", namespace, name).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "Skill not found"})
			return
		}

		var versions []models.SkillVersion
		if err := db.Where("skill_id = ?", skill.ID).Order("published_at DESC").Find(&versions).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, versions)
	}
}

// CreateSkillRequest 创建技能请求
type CreateSkillRequest struct {
	Namespace     string   `json:"namespace"`
	Name          string   `json:"name" binding:"required"`
	Description   string   `json:"description"`
	Tags          []string `json:"tags"`
	License       string   `json:"license"`
	IsPublic      bool     `json:"is_public"`
}

// CreateSkill 创建技能
func CreateSkill(db *storage.Database, objStorage *storage.ObjectStorage, scanner *validator.Scanner) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		// 解析表单
		namespace := c.PostForm("namespace")
		if namespace == "" {
			namespace = user.Username
		}
		name := c.PostForm("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "name is required"})
			return
		}

		// 检查技能是否已存在
		var existingSkill models.Skill
		if err := db.Where("namespace = ? AND name = ?", namespace, name).First(&existingSkill).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"code": "ALREADY_EXISTS", "message": "Skill already exists"})
			return
		}

		// 获取上传的文件
		file, err := c.FormFile("skill")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "skill file is required"})
			return
		}

		// 打开文件
		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Failed to open file"})
			return
		}
		defer src.Close()

		// 读取文件内容
		content, err := io.ReadAll(src)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Failed to read file"})
			return
		}

		// 安全扫描
		scanResult := scanner.ScanContent(string(content))
		if scanResult.Status == "fail" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    "VALIDATION_FAILED",
				"message": "Security scan failed",
				"details": scanResult.Issues,
			})
			return
		}

		// 存储文件
		storagePath := fmt.Sprintf("skills/%s/%s/%s.tar.gz", namespace, name, uuid.New().String())
		if err := objStorage.Upload(c, storagePath, bytes.NewReader(content), int64(len(content))); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Failed to upload file"})
			return
		}

		// 创建技能记录
		skill := models.Skill{
			Namespace:   namespace,
			Name:        name,
			OwnerID:     user.ID,
			Description: c.PostForm("description"),
			Tags:        []string{},
			License:     c.PostForm("license"),
			IsPublic:    c.PostForm("is_public") == "true",
		}

		if err := db.Create(&skill).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		// 创建版本记录
		version := models.SkillVersion{
			SkillID:     skill.ID,
			Version:     c.PostForm("version"),
			StoragePath: storagePath,
			SizeBytes:   int64(len(content)),
			ScanStatus:  scanResult.Status,
			ScanResult:  models.JSON{"issues": scanResult.Issues},
			PublishedBy: user.ID,
		}

		if err := db.Create(&version).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		// 更新技能最新版本
		skill.LatestVersion = version.Version
		db.Save(&skill)

		c.JSON(http.StatusCreated, gin.H{
			"namespace":   namespace,
			"name":        name,
			"version":     version.Version,
			"download_url": fmt.Sprintf("/api/v1/download/%s/%s/%s", namespace, name, version.Version),
		})
	}
}

// UpdateSkill 更新技能
func UpdateSkill(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := c.Param("namespace")
		name := c.Param("name")
		user := c.MustGet("user").(*models.User)

		var skill models.Skill
		if err := db.First(&skill, "namespace = ? AND name = ?", namespace, name).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "Skill not found"})
			return
		}

		// 检查权限
		if skill.OwnerID != user.ID {
			c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "You don't have permission to update this skill"})
			return
		}

		var req CreateSkillRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}

		// 更新字段
		skill.Description = req.Description
		skill.Tags = req.Tags
		skill.License = req.License
		skill.IsPublic = req.IsPublic

		if err := db.Save(&skill).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, skill)
	}
}

// DeleteSkill 删除技能
func DeleteSkill(db *storage.Database, objStorage *storage.ObjectStorage) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := c.Param("namespace")
		name := c.Param("name")
		user := c.MustGet("user").(*models.User)

		var skill models.Skill
		if err := db.Preload("Versions").First(&skill, "namespace = ? AND name = ?", namespace, name).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "Skill not found"})
			return
		}

		// 检查权限
		if skill.OwnerID != user.ID {
			c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "You don't have permission to delete this skill"})
			return
		}

		// 删除所有版本文件
		// for _, version := range skill.Versions {
		// 	objStorage.Delete(c, version.StoragePath)
		// }

		// 删除技能
		if err := db.Delete(&skill).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Skill deleted"})
	}
}
