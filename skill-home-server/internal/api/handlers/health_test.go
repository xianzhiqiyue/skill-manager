package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthCheckUsesProvidedVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/health", HealthCheck("v9.9.9"))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if got := payload["version"]; got != "v9.9.9" {
		t.Fatalf("version = %v, want %q", got, "v9.9.9")
	}
}
