package registry

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoginPostsJSONAndReturnsAuthResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.URL.Path != "/api/v1/auth/login" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("unexpected content-type: %s", got)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request failed: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if req["email"] != "tester@example.com" || req["password"] != "secret-123" {
			t.Errorf("unexpected request payload: %+v", req)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_ = json.NewEncoder(w).Encode(AuthResponse{
			Token: "jwt_token",
			User: User{
				ID:       "user-1",
				Username: "tester",
				Email:    "tester@example.com",
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	resp, err := client.Login("tester@example.com", "secret-123")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if resp.Token != "jwt_token" || resp.User.Username != "tester" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestGetCatalogVersionRequestsExpectedEndpoint(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.URL.Path != "/api/v1/catalog/version" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_ = json.NewEncoder(w).Encode(CatalogVersionResponse{
			CatalogVersion: "2026.03.30",
			UpdatedAt:      time.Date(2026, 3, 30, 9, 15, 0, 0, time.UTC),
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	resp, err := client.GetCatalogVersion()
	if err != nil {
		t.Fatalf("GetCatalogVersion returned error: %v", err)
	}
	if resp.CatalogVersion != "2026.03.30" {
		t.Fatalf("unexpected catalog version: %+v", resp)
	}
	if !resp.UpdatedAt.Equal(time.Date(2026, 3, 30, 9, 15, 0, 0, time.UTC)) {
		t.Fatalf("unexpected updated_at: %+v", resp)
	}
}

func TestGetCatalogVersionReturnsServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(APIError{
			Code:    "INTERNAL_ERROR",
			Message: "boom",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	_, err := client.GetCatalogVersion()
	if err == nil {
		t.Fatal("GetCatalogVersion returned nil error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.Code != "INTERNAL_ERROR" || apiErr.Message != "boom" {
		t.Fatalf("unexpected APIError: %+v", apiErr)
	}
}

func TestSearchIncludesNamespaceTagsAndPagination(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.URL.Query().Get("q"); got != "lint" {
			t.Errorf("unexpected q: %s", got)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.URL.Query().Get("namespace"); got != "team" {
			t.Errorf("unexpected namespace: %s", got)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		tags := r.URL.Query()["tag"]
		if len(tags) != 2 || tags[0] != "go" || tags[1] != "cli" {
			t.Errorf("unexpected tags: %#v", tags)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Errorf("unexpected page: %s", got)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.URL.Query().Get("per_page"); got != "15" {
			t.Errorf("unexpected per_page: %s", got)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_ = json.NewEncoder(w).Encode(SearchResult{
			Total:   1,
			Page:    2,
			PerPage: 15,
			Results: []Skill{{Namespace: "team", Name: "lint"}},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	result, err := client.Search("lint", "@team", []string{"go", "cli"}, 2, 15)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if result.Total != 1 || len(result.Results) != 1 || result.Results[0].Name != "lint" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestListSkillsUsesSkillsEndpoint(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/skills" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.URL.Query().Get("namespace"); got != "team" {
			t.Errorf("unexpected namespace: %s", got)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(SearchResult{
			Total:   1,
			Page:    1,
			PerPage: 20,
			Results: []Skill{{Namespace: "team", Name: "reviewer"}},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	result, err := client.ListSkills(ListSkillsOptions{
		Namespace: "team",
		Page:      1,
		PerPage:   20,
	})
	if err != nil {
		t.Fatalf("ListSkills returned error: %v", err)
	}
	if len(result.Results) != 1 || result.Results[0].Name != "reviewer" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRateSkillPostsJSONAndAuthorization(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.URL.Path != "/api/v1/skills/team/reviewer/rating" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test" {
			t.Errorf("unexpected authorization header: %s", got)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var req RateSkillRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request failed: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if req.Rating != 5 || req.Comment != "great" {
			t.Errorf("unexpected request payload: %+v", req)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_ = json.NewEncoder(w).Encode(RateSkillResponse{
			Skill: Skill{
				Namespace:   "team",
				Name:        "reviewer",
				Rating:      4.5,
				RatingCount: 2,
			},
			UserRating: SkillRating{
				Rating:  5,
				Comment: "great",
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "sk_test")
	resp, err := client.RateSkill("team", "reviewer", &RateSkillRequest{
		Rating:  5,
		Comment: "great",
	})
	if err != nil {
		t.Fatalf("RateSkill returned error: %v", err)
	}
	if resp.Skill.Rating != 4.5 || resp.UserRating.Rating != 5 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestListAuditLogsUsesExpectedQueryParameters(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/user/audit-logs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Errorf("unexpected page: %s", got)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.URL.Query().Get("per_page"); got != "5" {
			t.Errorf("unexpected per_page: %s", got)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.URL.Query().Get("action"); got != "skill.rate" {
			t.Errorf("unexpected action: %s", got)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_ = json.NewEncoder(w).Encode(AuditLogList{
			Total:   1,
			Page:    2,
			PerPage: 5,
			Results: []AuditLog{{Action: "skill.rate", ResourceType: "skill"}},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "sk_test")
	result, err := client.ListAuditLogs(2, 5, "skill.rate")
	if err != nil {
		t.Fatalf("ListAuditLogs returned error: %v", err)
	}
	if result.Total != 1 || len(result.Results) != 1 || result.Results[0].Action != "skill.rate" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestDownloadRequestsZipFormat(t *testing.T) {
	t.Parallel()

	var paths []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)

		switch r.URL.Path {
		case "/api/v1/skills/team/reviewer":
			_ = json.NewEncoder(w).Encode(Skill{
				Namespace:     "team",
				Name:          "reviewer",
				LatestVersion: "1.0.0",
			})
		case "/api/v1/download/team/reviewer/1.0.0":
			if got := r.URL.Query().Get("format"); got != "zip" {
				t.Errorf("unexpected format: %s", got)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, _ = io.WriteString(w, "zip-bytes")
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	outputPath := t.TempDir() + "/skill.zip"
	if err := client.Download("team", "reviewer", "1.0.0", outputPath); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	if len(paths) != 2 || paths[0] != "/api/v1/skills/team/reviewer" || paths[1] != "/api/v1/download/team/reviewer/1.0.0" {
		t.Fatalf("unexpected request order: %#v", paths)
	}
}

func TestDownloadUsesAbsoluteDownloadURLWhenPresent(t *testing.T) {
	t.Parallel()

	var paths []string
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)

		if r.URL.Path != "/oss/team/reviewer/1.0.0.zip" {
			t.Fatalf("unexpected download path: %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, "oss-zip-bytes")
	}))
	defer downloadServer.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)

		if r.URL.Path != "/api/v1/skills/team/reviewer" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		_ = json.NewEncoder(w).Encode(Skill{
			Namespace:     "team",
			Name:          "reviewer",
			LatestVersion: "1.0.0",
			DownloadURL:   downloadServer.URL + "/oss/team/reviewer/1.0.0.zip",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	outputPath := t.TempDir() + "/skill.zip"
	if err := client.Download("team", "reviewer", "1.0.0", outputPath); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	if len(paths) != 2 || paths[0] != "/api/v1/skills/team/reviewer" || paths[1] != "/oss/team/reviewer/1.0.0.zip" {
		t.Fatalf("unexpected request order: %#v", paths)
	}
}

func TestDownloadDoesNotReuseAbsoluteDownloadURLForNonLatestVersion(t *testing.T) {
	t.Parallel()

	var paths []string
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected direct download request: %s", r.URL.Path)
	}))
	defer downloadServer.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)

		switch r.URL.Path {
		case "/api/v1/skills/team/reviewer":
			_ = json.NewEncoder(w).Encode(Skill{
				Namespace:     "team",
				Name:          "reviewer",
				LatestVersion: "1.0.0",
				DownloadURL:   downloadServer.URL + "/oss/team/reviewer/1.0.0.zip",
			})
		case "/api/v1/download/team/reviewer/0.9.0":
			if got := r.URL.Query().Get("format"); got != "zip" {
				t.Fatalf("unexpected format: %s", got)
			}
			_, _ = io.WriteString(w, "zip-bytes")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	outputPath := t.TempDir() + "/skill.zip"
	if err := client.Download("team", "reviewer", "0.9.0", outputPath); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	if len(paths) != 2 || paths[0] != "/api/v1/skills/team/reviewer" || paths[1] != "/api/v1/download/team/reviewer/0.9.0" {
		t.Fatalf("unexpected request order: %#v", paths)
	}
}

func TestDownloadFallsBackWhenRelativeDownloadURLPresent(t *testing.T) {
	t.Parallel()

	var paths []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)

		switch r.URL.Path {
		case "/api/v1/skills/team/reviewer":
			_ = json.NewEncoder(w).Encode(Skill{
				Namespace:     "team",
				Name:          "reviewer",
				LatestVersion: "1.0.0",
				DownloadURL:   "/api/v1/download/team/reviewer/1.0.0",
			})
		case "/api/v1/download/team/reviewer/1.0.0":
			if got := r.URL.Query().Get("format"); got != "zip" {
				t.Fatalf("unexpected format: %s", got)
			}
			_, _ = io.WriteString(w, "zip-bytes")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	outputPath := t.TempDir() + "/skill.zip"
	if err := client.Download("team", "reviewer", "1.0.0", outputPath); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	if len(paths) != 2 || paths[0] != "/api/v1/skills/team/reviewer" || paths[1] != "/api/v1/download/team/reviewer/1.0.0" {
		t.Fatalf("unexpected request order: %#v", paths)
	}
}

func TestDownloadFallsBackToLegacyEndpointWhenDownloadURLMissing(t *testing.T) {
	t.Parallel()

	var paths []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)

		switch r.URL.Path {
		case "/api/v1/skills/team/reviewer":
			_ = json.NewEncoder(w).Encode(Skill{
				Namespace:     "team",
				Name:          "reviewer",
				LatestVersion: "1.0.0",
			})
		case "/api/v1/download/team/reviewer/1.0.0":
			if got := r.URL.Query().Get("format"); got != "zip" {
				t.Fatalf("unexpected format: %s", got)
			}
			_, _ = io.WriteString(w, "zip-bytes")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	outputPath := t.TempDir() + "/skill.zip"
	if err := client.Download("team", "reviewer", "1.0.0", outputPath); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	if len(paths) != 2 || paths[0] != "/api/v1/skills/team/reviewer" || paths[1] != "/api/v1/download/team/reviewer/1.0.0" {
		t.Fatalf("unexpected request order: %#v", paths)
	}
}

func TestPublishFallsBackToVersionEndpointWhenSkillAlreadyExists(t *testing.T) {
	t.Parallel()

	var createCalls int
	var versionCalls int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/skills":
			createCalls++
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(APIError{
				Code:    "ALREADY_EXISTS",
				Message: "Skill already exists",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/skills/team/reviewer/versions":
			versionCalls++
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatalf("ParseMultipartForm returned error: %v", err)
			}
			if got := r.FormValue("version"); got != "1.2.3" {
				t.Fatalf("unexpected version: %q", got)
			}
			file, _, err := r.FormFile("skill")
			if err != nil {
				t.Fatalf("FormFile returned error: %v", err)
			}
			defer file.Close()
			body, err := io.ReadAll(file)
			if err != nil {
				t.Fatalf("ReadAll returned error: %v", err)
			}
			if string(body) != "zip-bytes" {
				t.Fatalf("unexpected uploaded body: %q", string(body))
			}
			_ = json.NewEncoder(w).Encode(PublishResponse{
				Namespace:   "team",
				Name:        "reviewer",
				Version:     "1.2.3",
				DownloadURL: "/api/v1/download/team/reviewer/1.2.3",
				PublishedAt: "2026-03-26T00:00:00Z",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "sk_test")
	skillPath := filepath.Join(t.TempDir(), "reviewer.zip")
	if err := os.WriteFile(skillPath, []byte("zip-bytes"), 0644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	resp, err := client.Publish(skillPath, &PublishRequest{
		Namespace: "team",
		Name:      "reviewer",
		Version:   "1.2.3",
	})
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if createCalls != 1 {
		t.Fatalf("unexpected createCalls: %d", createCalls)
	}
	if versionCalls != 1 {
		t.Fatalf("unexpected versionCalls: %d", versionCalls)
	}
	if resp.Version != "1.2.3" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
