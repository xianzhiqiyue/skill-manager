package registry

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/download/team/reviewer/1.0.0" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.URL.Query().Get("format"); got != "zip" {
			t.Errorf("unexpected format: %s", got)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, _ = io.WriteString(w, "zip-bytes")
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	outputPath := t.TempDir() + "/skill.zip"
	if err := client.Download("team", "reviewer", "1.0.0", outputPath); err != nil {
		t.Fatalf("Download returned error: %v", err)
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
