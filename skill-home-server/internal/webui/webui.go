package webui

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

const envDistDir = "SKILL_HOME_WEB_DIST_DIR"
const envInstallScript = "SKILL_HOME_INSTALL_SCRIPT"
const envReleaseAssetsDir = "SKILL_HOME_RELEASES_DIR"

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
	installScriptPath := ResolveInstallScriptPath(distDir)
	releaseAssetsDir := ResolveReleaseAssetsDir(distDir)

	serveInstallScript := func(c *gin.Context) {
		if installScriptPath == "" {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    "NOT_FOUND",
				"message": "Install script not found",
			})
			return
		}

		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.File(installScriptPath)
	}

	router.GET("/install.sh", serveInstallScript)
	router.HEAD("/install.sh", serveInstallScript)

	serveReleaseAsset := func(c *gin.Context) {
		if releaseAssetsDir == "" {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    "NOT_FOUND",
				"message": "Release asset not found",
			})
			return
		}

		relPath, ok := cleanRelativePath(c.Param("assetPath"))
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    "NOT_FOUND",
				"message": "Release asset not found",
			})
			return
		}

		candidate := filepath.Join(releaseAssetsDir, relPath)
		if !isFile(candidate) {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    "NOT_FOUND",
				"message": "Release asset not found",
			})
			return
		}

		c.File(candidate)
	}

	router.GET("/releases/*assetPath", serveReleaseAsset)
	router.HEAD("/releases/*assetPath", serveReleaseAsset)

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

func ResolveInstallScriptPath(distDir string) string {
	candidates := make([]string, 0, 5)

	if fromEnv := strings.TrimSpace(os.Getenv(envInstallScript)); fromEnv != "" {
		candidates = append(candidates, fromEnv)
	}

	distParent := filepath.Dir(distDir)
	candidates = append(candidates,
		filepath.Join(distParent, "install.sh"),
		"./install.sh",
		"../skill-home-cli/install.sh",
		filepath.Join(distParent, "..", "skill-home-cli", "install.sh"),
	)

	for _, candidate := range candidates {
		if isFile(candidate) {
			return candidate
		}
	}

	return ""
}

func ResolveReleaseAssetsDir(distDir string) string {
	candidates := make([]string, 0, 4)

	if fromEnv := strings.TrimSpace(os.Getenv(envReleaseAssetsDir)); fromEnv != "" {
		candidates = append(candidates, fromEnv)
	}

	distParent := filepath.Dir(distDir)
	candidates = append(candidates,
		filepath.Join(distParent, "releases"),
		"./releases",
		filepath.Join(distParent, "..", "releases"),
	)

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	return ""
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
