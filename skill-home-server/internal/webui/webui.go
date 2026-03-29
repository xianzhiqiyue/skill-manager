package webui

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	serverconfig "github.com/skill-home/server/internal/config"
)

const envDistDir = "SKILL_HOME_WEB_DIST_DIR"
const envInstallScript = "SKILL_HOME_INSTALL_SCRIPT"
const envReleaseAssetsDir = "SKILL_HOME_RELEASES_DIR"
const installScriptReleasesBasePlaceholder = "__SKILL_HOME_RELEASES_BASE_URL__"

type Options struct {
	BasePath string
}

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

// Register mounts the frontend UI on the configured base path while leaving API routes intact.
func Register(router *gin.Engine, distDir string, opts ...Options) {
	options := Options{}
	if len(opts) > 0 {
		options = opts[0]
	}
	basePath := serverconfig.NormalizeBasePath(options.BasePath)

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

		content, err := os.ReadFile(installScriptPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Install script could not be loaded",
			})
			return
		}

		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.Data(http.StatusOK, "text/plain; charset=utf-8", injectInstallScriptReleasesBaseURL(content, publicReleasesBaseURL(c, basePath)))
	}

	if basePath != "" {
		redirectTarget := serverconfig.JoinBasePath(basePath, "/")
		router.GET(basePath, func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, redirectTarget)
		})
		router.HEAD(basePath, func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, redirectTarget)
		})
	}

	router.GET(serverconfig.JoinBasePath(basePath, "/install.sh"), serveInstallScript)
	router.HEAD(serverconfig.JoinBasePath(basePath, "/install.sh"), serveInstallScript)

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

	router.GET(serverconfig.JoinBasePath(basePath, "/releases/*assetPath"), serveReleaseAsset)
	router.HEAD(serverconfig.JoinBasePath(basePath, "/releases/*assetPath"), serveReleaseAsset)

	router.GET(serverconfig.JoinBasePath(basePath, "/"), func(c *gin.Context) {
		c.File(indexPath)
	})

	router.NoRoute(func(c *gin.Context) {
		requestPath := c.Request.URL.Path
		relativePath, ok := stripBasePath(requestPath, basePath)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    "NOT_FOUND",
				"message": "Route not found",
			})
			return
		}

		if strings.HasPrefix(relativePath, "/api/") || relativePath == "/health" {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    "NOT_FOUND",
				"message": "Route not found",
			})
			return
		}

		if relPath, ok := cleanRelativePath(relativePath); ok {
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

func stripBasePath(requestPath string, basePath string) (string, bool) {
	normalizedRequestPath := strings.TrimSpace(requestPath)
	if normalizedRequestPath == "" {
		normalizedRequestPath = "/"
	}
	if !strings.HasPrefix(normalizedRequestPath, "/") {
		normalizedRequestPath = "/" + normalizedRequestPath
	}

	normalizedBasePath := serverconfig.NormalizeBasePath(basePath)
	if normalizedBasePath == "" {
		return normalizedRequestPath, true
	}

	if normalizedRequestPath == normalizedBasePath || normalizedRequestPath == normalizedBasePath+"/" {
		return "/", true
	}

	if strings.HasPrefix(normalizedRequestPath, normalizedBasePath+"/") {
		return strings.TrimPrefix(normalizedRequestPath, normalizedBasePath), true
	}

	return "", false
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

func publicReleasesBaseURL(c *gin.Context, basePath string) string {
	return strings.TrimRight(publicBaseURL(c, basePath), "/") + "/releases"
}

func publicBaseURL(c *gin.Context, basePath string) string {
	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = c.Request.Host
	}

	scheme := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0])
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	return scheme + "://" + host + serverconfig.NormalizeBasePath(basePath)
}

func injectInstallScriptReleasesBaseURL(script []byte, releasesBaseURL string) []byte {
	if len(script) == 0 || releasesBaseURL == "" {
		return script
	}
	return []byte(strings.ReplaceAll(string(script), installScriptReleasesBasePlaceholder, releasesBaseURL))
}
