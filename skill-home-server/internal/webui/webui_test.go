package webui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterServesIndexAndStaticFiles(t *testing.T) {
	t.Setenv(envDistDir, "")
	gin.SetMode(gin.TestMode)

	distDir := t.TempDir()
	writeFile(t, filepath.Join(distDir, "index.html"), "<html>home</html>")
	writeFile(t, filepath.Join(distDir, "assets", "app.js"), "console.log('ok');")

	router := gin.New()
	Register(router, distDir)

	tests := []struct {
		name       string
		path       string
		wantCode   int
		wantSubstr string
	}{
		{name: "root", path: "/", wantCode: http.StatusOK, wantSubstr: "home"},
		{name: "static", path: "/assets/app.js", wantCode: http.StatusOK, wantSubstr: "console.log"},
		{name: "spa fallback", path: "/console/publish", wantCode: http.StatusOK, wantSubstr: "home"},
		{name: "api stays 404", path: "/api/v1/missing", wantCode: http.StatusNotFound, wantSubstr: "NOT_FOUND"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if body := rec.Body.String(); !strings.Contains(body, tt.wantSubstr) {
				t.Fatalf("body %q does not contain %q", body, tt.wantSubstr)
			}
		})
	}
}

func TestRegisterServesInstallScriptFromDeployRoot(t *testing.T) {
	t.Setenv(envDistDir, "")
	gin.SetMode(gin.TestMode)

	rootDir := t.TempDir()
	distDir := filepath.Join(rootDir, "web")
	writeFile(t, filepath.Join(distDir, "index.html"), "<html>home</html>")
	writeFile(t, filepath.Join(rootDir, "install.sh"), "#!/usr/bin/env bash\necho install\n")

	router := gin.New()
	Register(router, distDir)

	req := httptest.NewRequest(http.MethodGet, "/install.sh", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, "echo install") {
		t.Fatalf("body %q does not contain install script", body)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "shellscript") && !strings.Contains(contentType, "plain") {
		t.Fatalf("content-type = %q, want shell-compatible content type", contentType)
	}
}

func TestRegisterReturns404ForMissingInstallScript(t *testing.T) {
	t.Setenv(envDistDir, "")
	gin.SetMode(gin.TestMode)

	distDir := t.TempDir()
	writeFile(t, filepath.Join(distDir, "index.html"), "<html>home</html>")

	router := gin.New()
	Register(router, distDir)

	req := httptest.NewRequest(http.MethodGet, "/install.sh", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if body := rec.Body.String(); strings.Contains(body, "home") {
		t.Fatalf("body %q unexpectedly fell back to index", body)
	}
}

func TestResolveDistDirPrefersEnv(t *testing.T) {
	envDir := t.TempDir()
	writeFile(t, filepath.Join(envDir, "index.html"), "<html>env</html>")

	t.Setenv(envDistDir, envDir)
	if got := ResolveDistDir(); got != envDir {
		t.Fatalf("ResolveDistDir() = %q, want %q", got, envDir)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
