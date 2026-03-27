package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"gorm.io/gorm"
)

type CommunityTagRequest struct {
	Tag string `json:"tag"`
}

func AddCommunityTag(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := normalizeNamespace(c.Param("namespace"))
		name := c.Param("name")
		user := c.MustGet("user").(*models.User)

		var req CommunityTagRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}

		tag := normalizeTagValue(req.Tag)
		if tag == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "tag is required"})
			return
		}
		if len(tag) > 64 {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "tag is too long"})
			return
		}

		skill, err := loadSkillForCommunityTag(db, namespace, name, user)
		if err != nil {
			handleSkillLoadError(c, err)
			return
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			var existing models.SkillCommunityTag
			if err := tx.Where("skill_id = ? AND user_id = ? AND tag = ?", skill.ID, user.ID, tag).First(&existing).Error; err != nil {
				if err != gorm.ErrRecordNotFound {
					return err
				}
				if err := tx.Create(&models.SkillCommunityTag{
					SkillID: skill.ID,
					UserID:  user.ID,
					Tag:     tag,
				}).Error; err != nil {
					return err
				}
			}

			return writeAuditLogTx(tx, c, &user.ID, "skill.community_tag.add", resourceTypeSkill, &skill.ID, models.JSON{
				"namespace": namespace,
				"name":      name,
				"tag":       tag,
			})
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		respondSkillDetail(c, db, namespace, name, user)
	}
}

func RemoveCommunityTag(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := normalizeNamespace(c.Param("namespace"))
		name := c.Param("name")
		user := c.MustGet("user").(*models.User)
		tag := normalizeTagValue(c.Param("tag"))
		if tag == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "tag is required"})
			return
		}

		skill, err := loadSkillForCommunityTag(db, namespace, name, user)
		if err != nil {
			handleSkillLoadError(c, err)
			return
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("skill_id = ? AND user_id = ? AND tag = ?", skill.ID, user.ID, tag).Delete(&models.SkillCommunityTag{}).Error; err != nil {
				return err
			}

			return writeAuditLogTx(tx, c, &user.ID, "skill.community_tag.remove", resourceTypeSkill, &skill.ID, models.JSON{
				"namespace": namespace,
				"name":      name,
				"tag":       tag,
			})
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		respondSkillDetail(c, db, namespace, name, user)
	}
}

func loadSkillForCommunityTag(db *storage.Database, namespace, name string, user *models.User) (*models.Skill, error) {
	var skill models.Skill
	if err := scopeNamespaceName(db.Preload("Versions").Preload("Owner"), namespace, name).First(&skill).Error; err != nil {
		return nil, err
	}

	if !skill.IsPublic && (user == nil || user.ID != skill.OwnerID) {
		return nil, gorm.ErrRecordNotFound
	}

	return &skill, nil
}

func respondSkillDetail(c *gin.Context, db *storage.Database, namespace, name string, viewer *models.User) {
	var skill models.Skill
	if err := scopeNamespaceName(db.Preload("Versions").Preload("Owner"), namespace, name).First(&skill).Error; err != nil {
		handleSkillLoadError(c, err)
		return
	}

	if err := populateSkillDetailResponse(db, &skill, viewer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, skill)
}

func handleSkillLoadError(c *gin.Context, err error) {
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "Skill not found"})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
}
