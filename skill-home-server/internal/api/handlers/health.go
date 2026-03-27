package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// HealthCheck 健康检查
func HealthCheck(version string) gin.HandlerFunc {
	resolvedVersion := strings.TrimSpace(version)
	if resolvedVersion == "" {
		resolvedVersion = "dev"
	}

	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "skill-home-registry",
			"version": resolvedVersion,
		})
	}
}
