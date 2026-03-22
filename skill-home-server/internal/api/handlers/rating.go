package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"gorm.io/gorm"
)

type rateSkillRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

// RateSkill 为技能评分
func RateSkill(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		namespace := normalizeNamespace(c.Param("namespace"))
		name := c.Param("name")

		var req rateSkillRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": err.Error()})
			return
		}
		if req.Rating < 1 || req.Rating > 5 {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "rating must be between 1 and 5"})
			return
		}
		if len(req.Comment) > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_INPUT", "message": "comment is too long"})
			return
		}

		var skill models.Skill
		if err := scopeNamespaceName(db.DB, namespace, name).First(&skill).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "Skill not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}
		if !skill.IsPublic && skill.OwnerID != user.ID {
			c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "Access denied"})
			return
		}

		now := time.Now()
		userRating := models.SkillRating{}
		if err := db.Transaction(func(tx *gorm.DB) error {
			var existing models.SkillRating
			err := tx.Where("skill_id = ? AND user_id = ?", skill.ID, user.ID).First(&existing).Error
			switch err {
			case nil:
				delta := req.Rating - existing.Rating
				existing.Rating = req.Rating
				existing.Comment = strings.TrimSpace(req.Comment)
				existing.UpdatedAt = now
				if err := tx.Save(&existing).Error; err != nil {
					return err
				}
				if err := tx.Model(&models.Skill{}).
					Where("id = ?", skill.ID).
					Update("rating_sum", gorm.Expr("rating_sum + ?", delta)).Error; err != nil {
					return err
				}
				if err := writeAuditLogTx(tx, c, &user.ID, "skill.rate", resourceTypeSkill, &skill.ID, models.JSON{
					"namespace": namespace,
					"name":      name,
					"rating":    req.Rating,
					"comment":   strings.TrimSpace(req.Comment),
				}); err != nil {
					return err
				}
				userRating = existing
				return nil
			case gorm.ErrRecordNotFound:
				newRating := models.SkillRating{
					SkillID:   skill.ID,
					UserID:    user.ID,
					Rating:    req.Rating,
					Comment:   strings.TrimSpace(req.Comment),
					CreatedAt: now,
					UpdatedAt: now,
				}
				if err := tx.Create(&newRating).Error; err != nil {
					return err
				}
				if err := tx.Model(&models.Skill{}).
					Where("id = ?", skill.ID).
					Updates(map[string]interface{}{
						"rating_sum":   gorm.Expr("rating_sum + ?", req.Rating),
						"rating_count": gorm.Expr("rating_count + 1"),
					}).Error; err != nil {
					return err
				}
				if err := writeAuditLogTx(tx, c, &user.ID, "skill.rate", resourceTypeSkill, &skill.ID, models.JSON{
					"namespace": namespace,
					"name":      name,
					"rating":    req.Rating,
					"comment":   strings.TrimSpace(req.Comment),
				}); err != nil {
					return err
				}
				userRating = newRating
				return nil
			default:
				return err
			}
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		if err := scopeNamespaceName(db.Preload("Owner").Preload("Versions"), namespace, name).First(&skill).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}
		populateSkillComputedFields(&skill)
		skill.UserRating = &userRating

		c.JSON(http.StatusOK, gin.H{
			"skill":       skill,
			"user_rating": userRating,
		})
	}
}
