package webui

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

const envDistDir = "SKILL_HOME_WEB_DIST_DIR"

// ResolveDistDir finds a usable frontend dist directory for the web UI.
func ResolveDistDir() string {
	candidates := make([]string, 0, 3)

	if fromEnv := strings.TrimSpace(os.Getenv(envDistDir)); fromEnv != "" {
		candidates = append(candidates, fromEnv)
	}

	candidates = append(candidates, "./web", "../skill-home-web/dist")

	for _, candidate := range candidates {
		if hasIndexFile(candidate) {
			return candidate
		}
	}

	return ""
}

// Register mounts the frontend UI on the root path while leaving API routes intact.
func Register(router *gin.Engine, distDir string) {
	indexPath := filepath.Join(distDir, "index.html")

	router.GET("/", func(c *gin.Context) {
		c.File(indexPath)
	})

	router.NoRoute(func(c *gin.Context) {
		requestPath := c.Request.URL.Path
		if strings.HasPrefix(requestPath, "/api/") || requestPath == "/health" {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    "NOT_FOUND",
				"message": "Route not found",
			})
			return
		}

		if relPath, ok := cleanRelativePath(requestPath); ok {
			candidate := filepath.Join(distDir, relPath)
			if isFile(candidate) {
				c.File(candidate)
				return
			}
		}

		c.File(indexPath)
	})
}

func cleanRelativePath(path string) (string, bool) {
	trimmed := strings.TrimPrefix(path, "/")
	if trimmed == "" {
		return "", false
	}

	cleaned := filepath.Clean(trimmed)
	if cleaned == "." || strings.HasPrefix(cleaned, "..") {
		return "", false
	}

	return cleaned, true
}

func hasIndexFile(dir string) bool {
	return isFile(filepath.Join(dir, "index.html"))
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
