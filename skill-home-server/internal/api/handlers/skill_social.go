package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type installEventRequest struct {
	Version       string `json:"version"`
	Target        string `json:"target"`
	InstallMode   string `json:"install_mode"`
	ClientVersion string `json:"client_version"`
}

type shareEventRequest struct {
	Channel string `json:"channel"`
}

func LikeSkill(db *storage.Database, objStorages ...*storage.ObjectStorage) gin.HandlerFunc {
	objStorage := firstObjectStorage(objStorages...)

	return func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		namespace := normalizeNamespace(c.Param("namespace"))
		name := c.Param("name")

		skill, err := loadSkillForSocialAction(db, namespace, name, user)
		if err != nil {
			handleSkillLoadError(c, err)
			return
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.SkillLike{
				SkillID: skill.ID,
				UserID:  user.ID,
			})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected > 0 {
				if err := tx.Model(&models.Skill{}).
					Where("id = ?", skill.ID).
					UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
					return err
				}
			}
			return writeAuditLogTx(tx, c, &user.ID, "skill.like", resourceTypeSkill, &skill.ID, models.JSON{
				"namespace": namespace,
				"name":      name,
			})
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		respondSkillDetail(c, db, objStorage, namespace, name, user)
	}
}

func UnlikeSkill(db *storage.Database, objStorages ...*storage.ObjectStorage) gin.HandlerFunc {
	objStorage := firstObjectStorage(objStorages...)

	return func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		namespace := normalizeNamespace(c.Param("namespace"))
		name := c.Param("name")

		skill, err := loadSkillForSocialAction(db, namespace, name, user)
		if err != nil {
			handleSkillLoadError(c, err)
			return
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			result := tx.Where("skill_id = ? AND user_id = ?", skill.ID, user.ID).Delete(&models.SkillLike{})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected > 0 {
				if err := tx.Model(&models.Skill{}).
					Where("id = ?", skill.ID).
					UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count > 0 THEN like_count - 1 ELSE 0 END")).Error; err != nil {
					return err
				}
			}
			return writeAuditLogTx(tx, c, &user.ID, "skill.unlike", resourceTypeSkill, &skill.ID, models.JSON{
				"namespace": namespace,
				"name":      name,
			})
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		respondSkillDetail(c, db, objStorage, namespace, name, user)
	}
}

func RecordInstallEvent(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := normalizeNamespace(c.Param("namespace"))
		name := c.Param("name")
		viewer := currentUserFromContext(c)

		var req installEventRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}
		req.Version = trimLimit(req.Version, 20)
		req.Target = trimLimit(req.Target, 32)
		req.InstallMode = trimLimit(req.InstallMode, 32)
		req.ClientVersion = trimLimit(req.ClientVersion, 64)

		skill, err := loadSkillForSocialAction(db, namespace, name, viewer)
		if err != nil {
			handleSkillLoadError(c, err)
			return
		}

		var userID *uuid.UUID
		if viewer != nil && viewer.ID != uuid.Nil {
			id := viewer.ID
			userID = &id
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&models.SkillInstallEvent{
				SkillID:       skill.ID,
				UserID:        userID,
				Version:       req.Version,
				Target:        req.Target,
				InstallMode:   req.InstallMode,
				ClientVersion: req.ClientVersion,
			}).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Skill{}).
				Where("id = ?", skill.ID).
				UpdateColumn("install_count", gorm.Expr("install_count + 1")).Error; err != nil {
				return err
			}
			if userID == nil {
				return nil
			}
			return writeAuditLogTx(tx, c, userID, "skill.install", resourceTypeSkill, &skill.ID, models.JSON{
				"namespace":      namespace,
				"name":           name,
				"version":        req.Version,
				"target":         req.Target,
				"install_mode":   req.InstallMode,
				"client_version": req.ClientVersion,
			})
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		var updated models.Skill
		if err := scopeNamespaceName(db.DB, namespace, name).First(&updated).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}
		populateSkillComputedFields(&updated)
		c.JSON(http.StatusOK, gin.H{
			"install_count": updated.InstallCount,
			"skill":         updated,
		})
	}
}

func RecordShareEvent(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := normalizeNamespace(c.Param("namespace"))
		name := c.Param("name")
		viewer := currentUserFromContext(c)

		var req shareEventRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}
		req.Channel = trimLimit(req.Channel, 32)

		skill, err := loadSkillForSocialAction(db, namespace, name, viewer)
		if err != nil {
			handleSkillLoadError(c, err)
			return
		}

		var userID *uuid.UUID
		if viewer != nil && viewer.ID != uuid.Nil {
			id := viewer.ID
			userID = &id
		}

		if err := db.Create(&models.SkillShareEvent{
			SkillID: skill.ID,
			UserID:  userID,
			Channel: req.Channel,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Share event recorded"})
	}
}

func loadSkillForSocialAction(db *storage.Database, namespace, name string, viewer *models.User) (*models.Skill, error) {
	var skill models.Skill
	if err := scopeNamespaceName(db.Preload("Versions").Preload("Owner"), namespace, name).First(&skill).Error; err != nil {
		return nil, err
	}
	if !canAccessSkill(viewer, &skill) {
		return nil, gorm.ErrRecordNotFound
	}
	return &skill, nil
}

func trimLimit(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
