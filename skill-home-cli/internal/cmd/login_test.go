package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestRunLoginWithAPIKeyValidatesAndSavesConfig(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()

	home := t.TempDir()
	t.Setenv("HOME", home)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/user" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test_api_key" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         "user-1",
			"username":   "tester",
			"email":      "tester@example.com",
			"created_at": "2026-03-27T00:00:00Z",
		})
	}))
	defer server.Close()

	if err := runLogin(&loginOptions{
		server: server.URL,
		apiKey: "sk_test_api_key",
	}); err != nil {
		t.Fatalf("runLogin returned error: %v", err)
	}

	if got := viper.GetString("registry.endpoint"); got != server.URL {
		t.Fatalf("unexpected registry.endpoint: %s", got)
	}
	if got := viper.GetString("registry.api_key"); got != "sk_test_api_key" {
		t.Fatalf("unexpected registry.api_key: %s", got)
	}
	if got := viper.GetString("local.default_namespace"); got != "@tester" {
		t.Fatalf("unexpected local.default_namespace: %s", got)
	}

	configPath := filepath.Join(home, ".config", "skill-home", "config.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(content), "api_key: sk_test_api_key") {
		t.Fatalf("expected saved api key, got:\n%s", string(content))
	}
	if !strings.Contains(string(content), "default_namespace: '@tester'") &&
		!strings.Contains(string(content), "default_namespace: \"@tester\"") {
		t.Fatalf("expected saved default namespace, got:\n%s", string(content))
	}
}

func TestRunLoginWithEmailPasswordCreatesCLIAPIKey(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()

	home := t.TempDir()
	t.Setenv("HOME", home)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login":
			var req map[string]string
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("Decode returned error: %v", err)
			}
			if req["email"] != "tester@example.com" || req["password"] != "secret-123" {
				t.Fatalf("unexpected login payload: %+v", req)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"token": "jwt_token",
				"user": map[string]any{
					"id":       "user-1",
					"username": "tester",
					"email":    "tester@example.com",
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/user/api-keys":
			if got := r.Header.Get("Authorization"); got != "Bearer jwt_token" {
				t.Fatalf("unexpected authorization header: %s", got)
			}

			var req map[string]string
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("Decode returned error: %v", err)
			}
			if req["name"] != "local cli" {
				t.Fatalf("unexpected api key name: %+v", req)
			}

			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":         "key-1",
				"name":       "local cli",
				"key":        "sk_generated",
				"prefix":     "sk_gener",
				"created_at": "2026-03-27T00:00:00Z",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if err := runLogin(&loginOptions{
		server:     server.URL,
		email:      "tester@example.com",
		password:   "secret-123",
		apiKeyName: "local cli",
	}); err != nil {
		t.Fatalf("runLogin returned error: %v", err)
	}

	if got := viper.GetString("registry.endpoint"); got != server.URL {
		t.Fatalf("unexpected registry.endpoint: %s", got)
	}
	if got := viper.GetString("registry.api_key"); got != "sk_generated" {
		t.Fatalf("unexpected registry.api_key: %s", got)
	}
	if got := viper.GetString("local.default_namespace"); got != "@tester" {
		t.Fatalf("unexpected local.default_namespace: %s", got)
	}

	configPath := filepath.Join(home, ".config", "skill-home", "config.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(content), "api_key: sk_generated") {
		t.Fatalf("expected saved api key, got:\n%s", string(content))
	}
	if !strings.Contains(string(content), "default_namespace: '@tester'") &&
		!strings.Contains(string(content), "default_namespace: \"@tester\"") {
		t.Fatalf("expected saved default namespace, got:\n%s", string(content))
	}
}
