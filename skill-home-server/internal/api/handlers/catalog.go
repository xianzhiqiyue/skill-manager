package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/skill-home/server/internal/storage"
)

// GetCatalogVersion 返回目录版本状态。
func GetCatalogVersion(db *storage.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		state, err := getCatalogState(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"catalog_version": state.CatalogVersion,
			"updated_at":      state.UpdatedAt,
		})
	}
}
