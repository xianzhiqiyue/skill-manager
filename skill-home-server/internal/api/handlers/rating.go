package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
		if !skill.IsPublic && !canManageOwnedResource(user, skill.OwnerID) {
			c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "Access denied"})
			return
		}

		now := time.Now()
		comment := strings.TrimSpace(req.Comment)
		userRating := models.SkillRating{}
		if err := db.Transaction(func(tx *gorm.DB) error {
			userRating = models.SkillRating{
				SkillID:   skill.ID,
				UserID:    user.ID,
				Rating:    req.Rating,
				Comment:   comment,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "skill_id"},
					{Name: "user_id"},
				},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"rating":     req.Rating,
					"comment":    comment,
					"updated_at": now,
				}),
			}).Create(&userRating).Error; err != nil {
				return err
			}
			if err := tx.Where("skill_id = ? AND user_id = ?", skill.ID, user.ID).First(&userRating).Error; err != nil {
				return err
			}

			var aggregate struct {
				RatingSum   int64 `gorm:"column:rating_sum"`
				RatingCount int64 `gorm:"column:rating_count"`
			}
			if err := tx.Model(&models.SkillRating{}).
				Select("COALESCE(SUM(rating), 0) AS rating_sum, COUNT(*) AS rating_count").
				Where("skill_id = ?", skill.ID).
				Scan(&aggregate).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Skill{}).
				Where("id = ?", skill.ID).
				Updates(map[string]interface{}{
					"rating_sum":   aggregate.RatingSum,
					"rating_count": aggregate.RatingCount,
				}).Error; err != nil {
				return err
			}
			if err := writeAuditLogTx(tx, c, &user.ID, "skill.rate", resourceTypeSkill, &skill.ID, models.JSON{
				"namespace": namespace,
				"name":      name,
				"rating":    req.Rating,
				"comment":   comment,
			}); err != nil {
				return err
			}
			return nil
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
