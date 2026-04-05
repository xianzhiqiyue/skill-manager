package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"github.com/skill-home/server/pkg/validator"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PublishVersion 发布新版本
func PublishVersion(db *storage.Database, objStorage *storage.ObjectStorage, scanner *validator.Scanner) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := normalizeNamespace(c.Param("namespace"))
		name := c.Param("name")
		user := c.MustGet("user").(*models.User)

		if err := validateNamespace(namespace); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}
		if err := validateSkillName(name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}

		versionValue := strings.TrimSpace(c.PostForm("version"))
		if versionValue == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "version is required"})
			return
		}
		if err := validateVersion(versionValue); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}

		// 获取技能
		var skill models.Skill
		if err := scopeNamespaceName(db.DB, namespace, name).First(&skill).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "Skill not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		// 检查权限
		if !canManageOwnedResource(user, skill.OwnerID) {
			c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "You don't have permission"})
			return
		}

		// 获取上传的文件
		file, err := c.FormFile("skill")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "Missing skill file"})
			return
		}

		content, err := readUploadedArchive(file, maxSkillArchiveBytes)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}
		archiveFormat, err := detectArchiveFormat(file.Filename, content)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}
		manifest := parseSkillArchiveManifest(content, archiveFormat)
		description := manifestStringValue(manifest, "description")
		descriptionZh := manifestStringValue(manifest, "description_zh")

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
		storagePath := fmt.Sprintf(
			"skills/%s/%s/%s.%s",
			storageSegment(namespace),
			storageSegment(name),
			uuid.New().String(),
			archiveExtension(archiveFormat),
		)
		if err := objStorage.Upload(c, storagePath, bytes.NewReader(content), int64(len(content))); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Failed to upload file"})
			return
		}

		// 创建版本记录
		version := models.SkillVersion{
			SkillID:     skill.ID,
			Version:     versionValue,
			Manifest:    manifest,
			StoragePath: storagePath,
			SizeBytes:   int64(len(content)),
			ScanStatus:  scanResult.Status,
			ScanResult:  models.JSON{"issues": scanResult.Issues},
			PublishedBy: user.ID,
			PublishedAt: time.Now(),
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&version)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return errSkillVersionExists
			}
			updates := map[string]interface{}{
				"latest_version": version.Version,
			}
			if description != "" {
				updates["description"] = description
			}
			if descriptionZh != "" {
				updates["description_zh"] = descriptionZh
			}
			if err := tx.Model(&skill).Updates(updates).Error; err != nil {
				return err
			}
			if skill.IsPublic {
				if err := bumpCatalogVersionTx(tx); err != nil {
					return err
				}
			}
			return writeAuditLogTx(tx, c, &user.ID, "version.publish", resourceTypeVersion, &version.ID, models.JSON{
				"namespace": namespace,
				"name":      name,
				"version":   version.Version,
			})
		}); err != nil {
			_ = objStorage.Delete(c, storagePath)
			if errors.Is(err, errSkillVersionExists) || isSkillVersionConflictError(err) {
				c.JSON(http.StatusConflict, gin.H{
					"code":    "VERSION_EXISTS",
					"message": "Version already exists",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"namespace":    namespace,
			"name":         name,
			"version":      version.Version,
			"download_url": resolvePublicDownloadURL(objStorage, skill.IsPublic, storagePath, namespace, name, version.Version),
			"published_at": version.PublishedAt,
		})
	}
}

// DeleteVersion 删除版本
func DeleteVersion(db *storage.Database, objStorage *storage.ObjectStorage) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := normalizeNamespace(c.Param("namespace"))
		name := c.Param("name")
		version := c.Param("version")
		user := c.MustGet("user").(*models.User)

		// 获取技能
		var skill models.Skill
		if err := scopeNamespaceName(db.DB, namespace, name).First(&skill).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "Skill not found"})
			return
		}

		// 检查权限
		if !canManageOwnedResource(user, skill.OwnerID) {
			c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "You don't have permission"})
			return
		}

		// 获取版本
		var skillVersion models.SkillVersion
		if err := db.First(&skillVersion, "skill_id = ? AND version = ?", skill.ID, version).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "Version not found"})
			return
		}

		// 删除文件
		if err := objStorage.Delete(c, skillVersion.StoragePath); err != nil {
			// 记录日志但不返回错误
		}

		// 删除记录
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Delete(&skillVersion).Error; err != nil {
				return err
			}

			var latest models.SkillVersion
			err := tx.Where("skill_id = ?", skill.ID).Order("published_at DESC").First(&latest).Error
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					if err := tx.Model(&skill).Update("latest_version", "").Error; err != nil {
						return err
					}
					if skill.IsPublic {
						if err := bumpCatalogVersionTx(tx); err != nil {
							return err
						}
					}
					return writeAuditLogTx(tx, c, &user.ID, "version.delete", resourceTypeVersion, &skillVersion.ID, models.JSON{
						"namespace": namespace,
						"name":      name,
						"version":   version,
					})
				}
				return err
			}
			if err := tx.Model(&skill).Update("latest_version", latest.Version).Error; err != nil {
				return err
			}
			if skill.IsPublic {
				if err := bumpCatalogVersionTx(tx); err != nil {
					return err
				}
			}
			return writeAuditLogTx(tx, c, &user.ID, "version.delete", resourceTypeVersion, &skillVersion.ID, models.JSON{
				"namespace": namespace,
				"name":      name,
				"version":   version,
			})
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Version deleted"})
	}
}

// DownloadSkill 下载技能
func DownloadSkill(db *storage.Database, objStorage *storage.ObjectStorage) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := normalizeNamespace(c.Param("namespace"))
		name := c.Param("name")
		version := c.Param("version")

		// 获取技能
		var skill models.Skill
		if err := scopeNamespaceName(db.DB, namespace, name).First(&skill).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "Skill not found"})
			return
		}

		// 获取版本
		var skillVersion models.SkillVersion
		if err := db.First(&skillVersion, "skill_id = ? AND version = ?", skill.ID, version).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "Version not found"})
			return
		}

		// 检查权限（私有技能需要认证）
		if !skill.IsPublic {
			user, exists := c.Get("user")
			if !exists {
				c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "Access denied"})
				return
			}
			owner, ok := user.(*models.User)
			if !ok || !canAccessSkill(owner, &skill) {
				c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "Access denied"})
				return
			}
		}

		sourceFormat, err := detectArchiveFormat(skillVersion.StoragePath, nil)
		if err == nil {
			targetFormat, err := resolveDownloadFormat(c.Query("format"), sourceFormat)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
				return
			}

			if skill.IsPublic {
				if publicURL := resolvePublicDownloadURL(objStorage, skill.IsPublic, skillVersion.StoragePath, namespace, name, version); (strings.HasPrefix(publicURL, "http://") || strings.HasPrefix(publicURL, "https://")) && targetFormat == sourceFormat {
					db.Model(&skill).UpdateColumn("download_count", gorm.Expr("download_count + 1"))
					c.Redirect(http.StatusFound, publicURL)
					return
				}
			}
		}

		// 读取文件
		reader, err := objStorage.Download(c, skillVersion.StoragePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Failed to download file"})
			return
		}
		defer reader.Close()

		content, err := io.ReadAll(reader)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Failed to read file"})
			return
		}

		sourceFormat, err = detectArchiveFormat(skillVersion.StoragePath, content)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Failed to detect archive format"})
			return
		}

		targetFormat, err := resolveDownloadFormat(c.Query("format"), sourceFormat)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}

		payload, err := convertArchive(content, sourceFormat, targetFormat)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Failed to convert archive"})
			return
		}

		// 更新下载计数
		db.Model(&skill).UpdateColumn("download_count", gorm.Expr("download_count + 1"))

		// 发送文件
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s.%s", name, version, archiveExtension(targetFormat)))
		c.Header("Content-Type", archiveContentType(targetFormat))
		if _, err := c.Writer.Write(payload); err != nil {
			c.Error(err)
		}
	}
}
